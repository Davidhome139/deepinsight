package agentsystem

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MarketplaceService manages agent/workflow sharing and discovery
type MarketplaceService struct {
	db             *gorm.DB
	agentService   *CustomAgentService
	workflowEngine *WorkflowEngine
}

// NewMarketplaceService creates a new marketplace service
func NewMarketplaceService(db *gorm.DB, agentService *CustomAgentService, workflowEngine *WorkflowEngine) *MarketplaceService {
	return &MarketplaceService{
		db:             db,
		agentService:   agentService,
		workflowEngine: workflowEngine,
	}
}

// PublishAgent publishes an agent to the marketplace
func (s *MarketplaceService) PublishAgent(ctx context.Context, userID uint, agentID string, listing *MarketplaceListing) (*models.MarketplaceItem, error) {
	// Get the agent
	agent, err := s.agentService.GetAgent(ctx, agentID, userID)
	if err != nil {
		return nil, err
	}

	// Get user info for author name
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Export agent data
	packageData, err := s.agentService.ExportAgent(ctx, agentID, userID)
	if err != nil {
		return nil, err
	}
	packageJSON, _ := json.Marshal(packageData)

	// Parse tags
	tagsJSON, _ := json.Marshal(listing.Tags)

	// Parse required tools from agent
	var toolBindings []map[string]interface{}
	json.Unmarshal(agent.ToolBindings, &toolBindings)
	requiredTools := make([]string, 0)
	for _, binding := range toolBindings {
		if tool, ok := binding["tool"].(string); ok {
			requiredTools = append(requiredTools, tool)
		}
	}
	requiredToolsJSON, _ := json.Marshal(requiredTools)

	// Parse screenshots
	screenshotsJSON, _ := json.Marshal(listing.Screenshots)

	item := &models.MarketplaceItem{
		ID:            uuid.New().String(),
		AuthorID:      userID,
		AuthorName:    user.Username,
		Type:          "agent",
		SourceID:      agentID,
		Name:          listing.Name,
		Description:   listing.Description,
		Icon:          agent.Icon,
		Version:       agent.Version,
		License:       listing.License,
		Tags:          tagsJSON,
		Category:      listing.Category,
		RequiredTools: requiredToolsJSON,
		MinModelCap:   listing.MinModelCapability,
		Documentation: listing.Documentation,
		Screenshots:   screenshotsJSON,
		DemoVideo:     listing.DemoVideo,
		PackageData:   packageJSON,
		Status:        "pending", // Requires review
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(item).Error; err != nil {
		return nil, fmt.Errorf("failed to publish agent: %w", err)
	}

	// Update agent with marketplace reference
	agent.IsPublic = true
	agent.MarketplaceID = &item.ID
	s.db.WithContext(ctx).Save(agent)

	return item, nil
}

// MarketplaceListing contains listing details for publishing
type MarketplaceListing struct {
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	License            string   `json:"license"`
	Tags               []string `json:"tags"`
	Category           string   `json:"category"`
	MinModelCapability string   `json:"min_model_capability"`
	Documentation      string   `json:"documentation"`
	Screenshots        []string `json:"screenshots"`
	DemoVideo          string   `json:"demo_video"`
}

// PublishWorkflow publishes a workflow to the marketplace
func (s *MarketplaceService) PublishWorkflow(ctx context.Context, userID uint, workflowID string, listing *MarketplaceListing) (*models.MarketplaceItem, error) {
	workflow, err := s.workflowEngine.GetWorkflow(ctx, workflowID, userID)
	if err != nil {
		return nil, err
	}

	var user models.User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	packageData, err := s.workflowEngine.ExportWorkflow(ctx, workflowID, userID)
	if err != nil {
		return nil, err
	}
	packageJSON, _ := json.Marshal(packageData)
	tagsJSON, _ := json.Marshal(listing.Tags)
	screenshotsJSON, _ := json.Marshal(listing.Screenshots)

	item := &models.MarketplaceItem{
		ID:            uuid.New().String(),
		AuthorID:      userID,
		AuthorName:    user.Username,
		Type:          "workflow",
		SourceID:      workflowID,
		Name:          listing.Name,
		Description:   listing.Description,
		Icon:          workflow.Icon,
		Version:       workflow.Version,
		License:       listing.License,
		Tags:          tagsJSON,
		Category:      listing.Category,
		MinModelCap:   listing.MinModelCapability,
		Documentation: listing.Documentation,
		Screenshots:   screenshotsJSON,
		DemoVideo:     listing.DemoVideo,
		PackageData:   packageJSON,
		Status:        "pending",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(item).Error; err != nil {
		return nil, fmt.Errorf("failed to publish workflow: %w", err)
	}

	workflow.IsPublic = true
	workflow.MarketplaceID = &item.ID
	s.db.WithContext(ctx).Save(workflow)

	return item, nil
}

// SearchItems searches the marketplace
func (s *MarketplaceService) SearchItems(ctx context.Context, filter MarketplaceFilter) (*MarketplaceSearchResult, error) {
	query := s.db.WithContext(ctx).Model(&models.MarketplaceItem{}).Where("status = ?", "published")

	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.Query != "" {
		searchPattern := "%" + filter.Query + "%"
		query = query.Where("name ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}
	if len(filter.Tags) > 0 {
		for _, tag := range filter.Tags {
			query = query.Where("tags::text ILIKE ?", "%"+tag+"%")
		}
	}
	if filter.MinRating > 0 {
		query = query.Where("avg_rating >= ?", filter.MinRating)
	}

	// Count total
	var total int64
	query.Count(&total)

	// Apply sorting
	switch filter.SortBy {
	case "downloads":
		query = query.Order("downloads DESC")
	case "rating":
		query = query.Order("avg_rating DESC")
	case "stars":
		query = query.Order("stars DESC")
	case "recent":
		query = query.Order("published_at DESC")
	case "benchmark":
		query = query.Order("benchmark_score DESC")
	default:
		query = query.Order("downloads DESC")
	}

	// Pagination
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	var items []models.MarketplaceItem
	if err := query.Offset(filter.Offset).Limit(filter.Limit).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return &MarketplaceSearchResult{
		Items:  items,
		Total:  total,
		Offset: filter.Offset,
		Limit:  filter.Limit,
	}, nil
}

// MarketplaceFilter defines search filters
type MarketplaceFilter struct {
	Query     string   `json:"query"`
	Type      string   `json:"type"` // agent, workflow
	Category  string   `json:"category"`
	Tags      []string `json:"tags"`
	MinRating float64  `json:"min_rating"`
	SortBy    string   `json:"sort_by"` // downloads, rating, stars, recent, benchmark
	Offset    int      `json:"offset"`
	Limit     int      `json:"limit"`
}

// MarketplaceSearchResult contains search results
type MarketplaceSearchResult struct {
	Items  []models.MarketplaceItem `json:"items"`
	Total  int64                    `json:"total"`
	Offset int                      `json:"offset"`
	Limit  int                      `json:"limit"`
}

// GetItem retrieves a marketplace item
func (s *MarketplaceService) GetItem(ctx context.Context, id string) (*models.MarketplaceItem, error) {
	var item models.MarketplaceItem
	if err := s.db.WithContext(ctx).Where("id = ? AND status = ?", id, "published").First(&item).Error; err != nil {
		return nil, fmt.Errorf("item not found: %w", err)
	}
	return &item, nil
}

// DownloadItem downloads and imports a marketplace item
func (s *MarketplaceService) DownloadItem(ctx context.Context, itemID string, userID uint) (interface{}, error) {
	item, err := s.GetItem(ctx, itemID)
	if err != nil {
		return nil, err
	}

	// Record download
	download := &models.MarketplaceDownload{
		ID:        uuid.New().String(),
		ItemID:    itemID,
		UserID:    userID,
		Version:   item.Version,
		CreatedAt: time.Now(),
	}
	s.db.WithContext(ctx).Create(download)

	// Update download count
	s.db.WithContext(ctx).Model(item).UpdateColumn("downloads", gorm.Expr("downloads + 1"))

	// Parse package data
	var packageData map[string]interface{}
	if err := json.Unmarshal(item.PackageData, &packageData); err != nil {
		return nil, fmt.Errorf("invalid package data: %w", err)
	}

	// Import based on type
	switch item.Type {
	case "agent":
		return s.agentService.ImportAgent(ctx, userID, packageData)
	case "workflow":
		return s.workflowEngine.ImportWorkflow(ctx, userID, packageData)
	default:
		return nil, fmt.Errorf("unknown item type: %s", item.Type)
	}
}

// StarItem adds a star to an item
func (s *MarketplaceService) StarItem(ctx context.Context, itemID string, userID uint) error {
	// Check if already starred
	var existing models.MarketplaceStar
	if err := s.db.WithContext(ctx).Where("item_id = ? AND user_id = ?", itemID, userID).First(&existing).Error; err == nil {
		return fmt.Errorf("already starred")
	}

	star := &models.MarketplaceStar{
		ID:        uuid.New().String(),
		ItemID:    itemID,
		UserID:    userID,
		CreatedAt: time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(star).Error; err != nil {
		return fmt.Errorf("failed to star: %w", err)
	}

	// Update star count
	s.db.WithContext(ctx).Model(&models.MarketplaceItem{}).Where("id = ?", itemID).UpdateColumn("stars", gorm.Expr("stars + 1"))

	return nil
}

// UnstarItem removes a star from an item
func (s *MarketplaceService) UnstarItem(ctx context.Context, itemID string, userID uint) error {
	result := s.db.WithContext(ctx).Where("item_id = ? AND user_id = ?", itemID, userID).Delete(&models.MarketplaceStar{})
	if result.RowsAffected == 0 {
		return fmt.Errorf("star not found")
	}

	s.db.WithContext(ctx).Model(&models.MarketplaceItem{}).Where("id = ?", itemID).UpdateColumn("stars", gorm.Expr("stars - 1"))

	return nil
}

// AddReview adds a review to an item
func (s *MarketplaceService) AddReview(ctx context.Context, itemID string, userID uint, review *ReviewInput) (*models.MarketplaceReview, error) {
	// Get user info
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Check if user has downloaded the item
	var downloadCount int64
	s.db.WithContext(ctx).Model(&models.MarketplaceDownload{}).Where("item_id = ? AND user_id = ?", itemID, userID).Count(&downloadCount)

	newReview := &models.MarketplaceReview{
		ID:         uuid.New().String(),
		ItemID:     itemID,
		UserID:     userID,
		Username:   user.Username,
		Rating:     review.Rating,
		Title:      review.Title,
		Comment:    review.Comment,
		IsVerified: downloadCount > 0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(newReview).Error; err != nil {
		return nil, fmt.Errorf("failed to add review: %w", err)
	}

	// Update item rating
	s.updateItemRating(ctx, itemID)

	return newReview, nil
}

// ReviewInput contains review data
type ReviewInput struct {
	Rating  int    `json:"rating"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
}

// GetReviews retrieves reviews for an item
func (s *MarketplaceService) GetReviews(ctx context.Context, itemID string, limit int) ([]models.MarketplaceReview, error) {
	var reviews []models.MarketplaceReview
	if err := s.db.WithContext(ctx).
		Where("item_id = ?", itemID).
		Order("created_at DESC").
		Limit(limit).
		Find(&reviews).Error; err != nil {
		return nil, fmt.Errorf("failed to get reviews: %w", err)
	}
	return reviews, nil
}

func (s *MarketplaceService) updateItemRating(ctx context.Context, itemID string) {
	var result struct {
		AvgRating float64
		Count     int64
	}

	s.db.WithContext(ctx).Model(&models.MarketplaceReview{}).
		Select("AVG(rating) as avg_rating, COUNT(*) as count").
		Where("item_id = ?", itemID).
		Scan(&result)

	s.db.WithContext(ctx).Model(&models.MarketplaceItem{}).
		Where("id = ?", itemID).
		Updates(map[string]interface{}{
			"avg_rating":   result.AvgRating,
			"rating_count": result.Count,
		})
}

// GetCategories returns available categories with counts
func (s *MarketplaceService) GetCategories(ctx context.Context) ([]CategoryInfo, error) {
	var results []struct {
		Category string
		Count    int64
	}

	if err := s.db.WithContext(ctx).Model(&models.MarketplaceItem{}).
		Select("category, COUNT(*) as count").
		Where("status = ?", "published").
		Group("category").
		Order("count DESC").
		Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	categories := make([]CategoryInfo, len(results))
	for i, r := range results {
		categories[i] = CategoryInfo{
			Name:  r.Category,
			Count: r.Count,
		}
	}

	return categories, nil
}

// CategoryInfo contains category information
type CategoryInfo struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// GetFeatured returns featured items
func (s *MarketplaceService) GetFeatured(ctx context.Context, limit int) ([]models.MarketplaceItem, error) {
	var items []models.MarketplaceItem
	if err := s.db.WithContext(ctx).
		Where("status = ? AND is_featured = ?", "published", true).
		Order("downloads DESC").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to get featured: %w", err)
	}
	return items, nil
}

// GetTrending returns trending items (high activity in recent period)
func (s *MarketplaceService) GetTrending(ctx context.Context, days int, limit int) ([]models.MarketplaceItem, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	// Get recent download counts
	var downloadCounts []struct {
		ItemID string
		Count  int64
	}
	s.db.WithContext(ctx).Model(&models.MarketplaceDownload{}).
		Select("item_id, COUNT(*) as count").
		Where("created_at >= ?", startDate).
		Group("item_id").
		Order("count DESC").
		Limit(limit).
		Scan(&downloadCounts)

	if len(downloadCounts) == 0 {
		return []models.MarketplaceItem{}, nil
	}

	itemIDs := make([]string, len(downloadCounts))
	for i, dc := range downloadCounts {
		itemIDs[i] = dc.ItemID
	}

	var items []models.MarketplaceItem
	if err := s.db.WithContext(ctx).
		Where("id IN ? AND status = ?", itemIDs, "published").
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to get trending: %w", err)
	}

	// Sort by download count
	countMap := make(map[string]int64)
	for _, dc := range downloadCounts {
		countMap[dc.ItemID] = dc.Count
	}
	sort.Slice(items, func(i, j int) bool {
		return countMap[items[i].ID] > countMap[items[j].ID]
	})

	return items, nil
}

// GetUserItems returns items published by a user
func (s *MarketplaceService) GetUserItems(ctx context.Context, userID uint) ([]models.MarketplaceItem, error) {
	var items []models.MarketplaceItem
	if err := s.db.WithContext(ctx).
		Where("author_id = ?", userID).
		Order("created_at DESC").
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to get user items: %w", err)
	}
	return items, nil
}

// UpdateListing updates a marketplace listing
func (s *MarketplaceService) UpdateListing(ctx context.Context, itemID string, userID uint, updates map[string]interface{}) (*models.MarketplaceItem, error) {
	var item models.MarketplaceItem
	if err := s.db.WithContext(ctx).Where("id = ? AND author_id = ?", itemID, userID).First(&item).Error; err != nil {
		return nil, fmt.Errorf("item not found or unauthorized: %w", err)
	}

	// Only allow updating certain fields
	allowedFields := map[string]bool{
		"name": true, "description": true, "documentation": true,
		"demo_video": true, "tags": true, "category": true,
	}

	filteredUpdates := make(map[string]interface{})
	for k, v := range updates {
		if allowedFields[k] {
			filteredUpdates[k] = v
		}
	}

	if err := s.db.WithContext(ctx).Model(&item).Updates(filteredUpdates).Error; err != nil {
		return nil, fmt.Errorf("failed to update listing: %w", err)
	}

	return &item, nil
}

// UnpublishItem removes an item from the marketplace
func (s *MarketplaceService) UnpublishItem(ctx context.Context, itemID string, userID uint) error {
	result := s.db.WithContext(ctx).
		Model(&models.MarketplaceItem{}).
		Where("id = ? AND author_id = ?", itemID, userID).
		Update("status", "archived")

	if result.Error != nil {
		return fmt.Errorf("failed to unpublish: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("item not found or unauthorized")
	}

	return nil
}

// ForkItem creates a fork of a marketplace item
func (s *MarketplaceService) ForkItem(ctx context.Context, itemID string, userID uint, newName string) (interface{}, error) {
	item, err := s.GetItem(ctx, itemID)
	if err != nil {
		return nil, err
	}

	// Increment fork count
	s.db.WithContext(ctx).Model(item).UpdateColumn("fork_count", gorm.Expr("fork_count + 1"))

	// Parse package data and import
	var packageData map[string]interface{}
	if err := json.Unmarshal(item.PackageData, &packageData); err != nil {
		return nil, fmt.Errorf("invalid package data: %w", err)
	}

	// Modify name
	if agentData, ok := packageData["agent"].(map[string]interface{}); ok {
		agentData["name"] = newName
	}
	if workflowData, ok := packageData["workflow"].(map[string]interface{}); ok {
		workflowData["name"] = newName
	}

	switch item.Type {
	case "agent":
		return s.agentService.ImportAgent(ctx, userID, packageData)
	case "workflow":
		return s.workflowEngine.ImportWorkflow(ctx, userID, packageData)
	default:
		return nil, fmt.Errorf("unknown item type: %s", item.Type)
	}
}

// ABTestService handles A/B testing for agents and workflows
type ABTestService struct {
	db           *gorm.DB
	agentService *CustomAgentService
}

// NewABTestService creates a new A/B test service
func NewABTestService(db *gorm.DB, agentService *CustomAgentService) *ABTestService {
	return &ABTestService{
		db:           db,
		agentService: agentService,
	}
}

// CreateTest creates a new A/B test
func (s *ABTestService) CreateTest(ctx context.Context, userID uint, test *models.ABTest) (*models.ABTest, error) {
	test.ID = uuid.New().String()
	test.UserID = userID
	test.Status = "draft"

	resultsA := map[string]interface{}{"runs": 0, "successes": 0, "avgLatency": 0, "avgRating": 0}
	resultsB := map[string]interface{}{"runs": 0, "successes": 0, "avgLatency": 0, "avgRating": 0}
	test.ResultsA, _ = json.Marshal(resultsA)
	test.ResultsB, _ = json.Marshal(resultsB)

	if err := s.db.WithContext(ctx).Create(test).Error; err != nil {
		return nil, fmt.Errorf("failed to create test: %w", err)
	}

	return test, nil
}

// StartTest starts an A/B test
func (s *ABTestService) StartTest(ctx context.Context, testID string, userID uint) error {
	now := time.Now()
	result := s.db.WithContext(ctx).
		Model(&models.ABTest{}).
		Where("id = ? AND user_id = ? AND status = ?", testID, userID, "draft").
		Updates(map[string]interface{}{
			"status":     "running",
			"started_at": now,
		})

	if result.RowsAffected == 0 {
		return fmt.Errorf("test not found or already started")
	}
	return result.Error
}

// RecordTestRun records a run in an A/B test
func (s *ABTestService) RecordTestRun(ctx context.Context, testID string, variantID string, executionID string, success bool, latencyMs int, rating *int) error {
	run := &models.ABTestRun{
		ID:          uuid.New().String(),
		TestID:      testID,
		VariantID:   variantID,
		ExecutionID: executionID,
		Success:     success,
		LatencyMs:   latencyMs,
		UserRating:  rating,
		CreatedAt:   time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(run).Error; err != nil {
		return fmt.Errorf("failed to record run: %w", err)
	}

	// Update test results
	go s.updateTestResults(testID, variantID)

	return nil
}

func (s *ABTestService) updateTestResults(testID, variantID string) {
	var test models.ABTest
	if err := s.db.First(&test, "id = ?", testID).Error; err != nil {
		return
	}

	// Calculate stats for this variant
	var stats struct {
		Runs       int64
		Successes  int64
		AvgLatency float64
		AvgRating  float64
	}

	s.db.Model(&models.ABTestRun{}).
		Select("COUNT(*) as runs, SUM(CASE WHEN success THEN 1 ELSE 0 END) as successes, AVG(latency_ms) as avg_latency, AVG(COALESCE(user_rating, 0)) as avg_rating").
		Where("test_id = ? AND variant_id = ?", testID, variantID).
		Scan(&stats)

	results := map[string]interface{}{
		"runs":       stats.Runs,
		"successes":  stats.Successes,
		"avgLatency": stats.AvgLatency,
		"avgRating":  stats.AvgRating,
	}
	resultsJSON, _ := json.Marshal(results)

	updateField := "results_a"
	if variantID == test.VariantBID {
		updateField = "results_b"
	}

	s.db.Model(&test).Update(updateField, resultsJSON)

	// Check if test should complete
	s.checkTestCompletion(&test)
}

func (s *ABTestService) checkTestCompletion(test *models.ABTest) {
	var resultsA, resultsB map[string]interface{}
	json.Unmarshal(test.ResultsA, &resultsA)
	json.Unmarshal(test.ResultsB, &resultsB)

	runsA := resultsA["runs"].(float64)
	runsB := resultsB["runs"].(float64)

	if runsA >= float64(test.MinSampleSize) && runsB >= float64(test.MinSampleSize) {
		// Determine winner based on criteria
		var winCriteria map[string]interface{}
		json.Unmarshal(test.WinCriteria, &winCriteria)

		metric, _ := winCriteria["metric"].(string)
		if metric == "" {
			metric = "successRate"
		}

		var scoreA, scoreB float64
		switch metric {
		case "successRate":
			scoreA = resultsA["successes"].(float64) / runsA
			scoreB = resultsB["successes"].(float64) / runsB
		case "avgLatency":
			scoreA = -resultsA["avgLatency"].(float64) // Lower is better
			scoreB = -resultsB["avgLatency"].(float64)
		case "avgRating":
			scoreA = resultsA["avgRating"].(float64)
			scoreB = resultsB["avgRating"].(float64)
		}

		now := time.Now()
		var winnerID *string
		if scoreA > scoreB {
			winnerID = &test.VariantAID
		} else if scoreB > scoreA {
			winnerID = &test.VariantBID
		}

		s.db.Model(test).Updates(map[string]interface{}{
			"status":       "completed",
			"winner_id":    winnerID,
			"completed_at": now,
		})
	}
}

// GetTest retrieves an A/B test
func (s *ABTestService) GetTest(ctx context.Context, testID string, userID uint) (*models.ABTest, error) {
	var test models.ABTest
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", testID, userID).First(&test).Error; err != nil {
		return nil, fmt.Errorf("test not found: %w", err)
	}
	return &test, nil
}

// ListTests returns user's A/B tests
func (s *ABTestService) ListTests(ctx context.Context, userID uint) ([]models.ABTest, error) {
	var tests []models.ABTest
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&tests).Error; err != nil {
		return nil, fmt.Errorf("failed to list tests: %w", err)
	}
	return tests, nil
}
