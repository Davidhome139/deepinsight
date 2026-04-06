package branch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"backend/internal/models"
	"backend/internal/pkg/database"
	"backend/internal/services/ai"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BranchService handles conversation branching, message editing, and parallel exploration
type BranchService struct {
	db        *gorm.DB
	aiService ai.AIService
	mu        sync.RWMutex
}

// NewBranchService creates a new branch service
func NewBranchService(aiService ai.AIService) *BranchService {
	return &BranchService{
		db:        database.DB,
		aiService: aiService,
	}
}

// =============================================================================
// Branch Operations
// =============================================================================

// CreateBranch creates a new branch from a specific message
func (s *BranchService) CreateBranch(conversationID uint, forkPointMsgID *uint, name, description string) (*models.Branch, error) {
	branch := &models.Branch{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		Name:           name,
		Description:    description,
		ForkPointMsgID: forkPointMsgID,
		IsMain:         false,
		IsActive:       true,
		Status:         models.BranchStatusActive,
	}

	// If forking from a message, get its branch as parent
	if forkPointMsgID != nil {
		var msg models.Message
		if err := s.db.First(&msg, *forkPointMsgID).Error; err != nil {
			return nil, fmt.Errorf("fork point message not found: %w", err)
		}
		branch.ParentBranchID = msg.BranchID
	}

	if err := s.db.Create(branch).Error; err != nil {
		return nil, err
	}

	// Update conversation branch count
	s.db.Model(&models.Conversation{}).Where("id = ?", conversationID).
		UpdateColumn("branch_count", gorm.Expr("branch_count + 1"))

	return branch, nil
}

// CreateMainBranch creates the main/default branch for a conversation
func (s *BranchService) CreateMainBranch(conversationID uint) (*models.Branch, error) {
	branch := &models.Branch{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		Name:           "Main",
		Description:    "Main conversation branch",
		IsMain:         true,
		IsActive:       true,
		Status:         models.BranchStatusActive,
	}

	if err := s.db.Create(branch).Error; err != nil {
		return nil, err
	}

	// Set as active branch for conversation
	s.db.Model(&models.Conversation{}).Where("id = ?", conversationID).
		Updates(map[string]interface{}{
			"active_branch_id": branch.ID,
			"branch_count":     gorm.Expr("branch_count + 1"),
		})

	return branch, nil
}

// GetBranches returns all branches for a conversation
func (s *BranchService) GetBranches(conversationID uint) ([]models.Branch, error) {
	var branches []models.Branch
	err := s.db.Where("conversation_id = ?", conversationID).
		Order("created_at asc").
		Find(&branches).Error
	return branches, err
}

// GetBranch returns a specific branch
func (s *BranchService) GetBranch(branchID string) (*models.Branch, error) {
	var branch models.Branch
	err := s.db.First(&branch, "id = ?", branchID).Error
	if err != nil {
		return nil, err
	}
	return &branch, nil
}

// SwitchBranch sets a branch as the active branch for a conversation
func (s *BranchService) SwitchBranch(conversationID uint, branchID string) error {
	// Verify branch belongs to conversation
	var branch models.Branch
	if err := s.db.First(&branch, "id = ? AND conversation_id = ?", branchID, conversationID).Error; err != nil {
		return fmt.Errorf("branch not found: %w", err)
	}

	return s.db.Model(&models.Conversation{}).Where("id = ?", conversationID).
		Update("active_branch_id", branchID).Error
}

// DeleteBranch soft-deletes a branch (cannot delete main branch)
func (s *BranchService) DeleteBranch(branchID string) error {
	var branch models.Branch
	if err := s.db.First(&branch, "id = ?", branchID).Error; err != nil {
		return err
	}

	if branch.IsMain {
		return fmt.Errorf("cannot delete main branch")
	}

	// Soft delete the branch
	if err := s.db.Delete(&branch).Error; err != nil {
		return err
	}

	// Update conversation branch count
	s.db.Model(&models.Conversation{}).Where("id = ?", branch.ConversationID).
		UpdateColumn("branch_count", gorm.Expr("branch_count - 1"))

	return nil
}

// GetBranchTree returns the branch tree structure for visualization
func (s *BranchService) GetBranchTree(conversationID uint) (*models.BranchTree, error) {
	branches, err := s.GetBranches(conversationID)
	if err != nil {
		return nil, err
	}

	if len(branches) == 0 {
		return nil, nil
	}

	// Build tree structure
	branchMap := make(map[string]*models.BranchTree)
	var root *models.BranchTree

	for i := range branches {
		b := &branches[i]
		branchMap[b.ID] = &models.BranchTree{
			Branch:   b,
			Children: []*models.BranchTree{},
		}
		if b.IsMain {
			root = branchMap[b.ID]
		}
	}

	// Link children to parents
	for _, bt := range branchMap {
		if bt.Branch.ParentBranchID != nil {
			if parent, ok := branchMap[*bt.Branch.ParentBranchID]; ok {
				parent.Children = append(parent.Children, bt)
			}
		}
	}

	return root, nil
}

// =============================================================================
// Message Editing Operations
// =============================================================================

// EditMessage edits a message and optionally regenerates the response
func (s *BranchService) EditMessage(msgID uint, newContent string, userID uint, regenerate bool) (*models.Message, error) {
	var msg models.Message
	if err := s.db.First(&msg, msgID).Error; err != nil {
		return nil, fmt.Errorf("message not found: %w", err)
	}

	// Save current version to history
	version := &models.MessageVersion{
		MessageID: msg.ID,
		Version:   msg.Version,
		Content:   msg.Content,
		EditedBy:  userID,
	}
	s.db.Create(version)

	// Update message
	msg.Content = newContent
	msg.Version++
	msg.IsEdited = true
	if msg.OriginalID == nil {
		originalID := msg.ID
		msg.OriginalID = &originalID
	}

	if err := s.db.Save(&msg).Error; err != nil {
		return nil, err
	}

	return &msg, nil
}

// GetMessageVersions returns all versions of a message
func (s *BranchService) GetMessageVersions(msgID uint) ([]models.MessageVersion, error) {
	var versions []models.MessageVersion
	err := s.db.Where("message_id = ?", msgID).
		Order("version desc").
		Find(&versions).Error
	return versions, err
}

// RevertMessage reverts a message to a previous version
func (s *BranchService) RevertMessage(msgID uint, toVersion int, userID uint) (*models.Message, error) {
	// Find the version to revert to
	var version models.MessageVersion
	if err := s.db.First(&version, "message_id = ? AND version = ?", msgID, toVersion).Error; err != nil {
		return nil, fmt.Errorf("version not found: %w", err)
	}

	// Edit message with the old content
	return s.EditMessage(msgID, version.Content, userID, false)
}

// CreateBranchFromEdit creates a new branch when editing, preserving original history
func (s *BranchService) CreateBranchFromEdit(msgID uint, newContent string, userID uint, branchName string) (*models.Branch, *models.Message, error) {
	var msg models.Message
	if err := s.db.First(&msg, msgID).Error; err != nil {
		return nil, nil, fmt.Errorf("message not found: %w", err)
	}

	// Create a new branch from this point
	branch, err := s.CreateBranch(msg.ConversationID, &msgID, branchName, "Branch created from edit")
	if err != nil {
		return nil, nil, err
	}

	// Create new message in the new branch with edited content
	newMsg := &models.Message{
		ConversationID: msg.ConversationID,
		Role:           msg.Role,
		Content:        newContent,
		Model:          msg.Model,
		Status:         msg.Status,
		ParentID:       msg.ParentID,
		BranchID:       &branch.ID,
		Version:        1,
		OriginalID:     &msg.ID,
	}

	if err := s.db.Create(newMsg).Error; err != nil {
		return nil, nil, err
	}

	return branch, newMsg, nil
}

// =============================================================================
// Parallel Model Exploration
// =============================================================================

// StartParallelExploration starts exploring multiple models in parallel
func (s *BranchService) StartParallelExploration(ctx context.Context, conversationID uint, sourceMsgID uint, prompt string, modelList []string) (*models.ParallelExploration, error) {
	modelsJSON, _ := json.Marshal(modelList)

	exploration := &models.ParallelExploration{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		SourceMsgID:    sourceMsgID,
		Prompt:         prompt,
		Models:         modelsJSON,
		Status:         models.ParallelStatusRunning,
	}

	if err := s.db.Create(exploration).Error; err != nil {
		return nil, err
	}

	// Run parallel exploration in background
	go s.runParallelExploration(ctx, exploration, modelList)

	return exploration, nil
}

func (s *BranchService) runParallelExploration(ctx context.Context, exploration *models.ParallelExploration, modelList []string) {
	var results []models.ParallelResult
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Get conversation history for context
	var history []models.Message
	s.db.Where("conversation_id = ?", exploration.ConversationID).
		Order("created_at asc").
		Limit(10).
		Find(&history)

	for _, model := range modelList {
		wg.Add(1)
		go func(m string) {
			defer wg.Done()

			startTime := time.Now()
			result := models.ParallelResult{
				Model: m,
			}

			// Create a branch for this model
			branchName := fmt.Sprintf("Explore: %s", m)
			branch, err := s.CreateBranch(exploration.ConversationID, &exploration.SourceMsgID, branchName, fmt.Sprintf("Parallel exploration using %s", m))
			if err != nil {
				result.Error = err.Error()
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}
			result.BranchID = branch.ID

			// Build messages for AI request
			messages := make([]models.Message, len(history))
			copy(messages, history)
			messages = append(messages, models.Message{
				Role:    "user",
				Content: exploration.Prompt,
			})

			// Call AI service
			req := &ai.ChatRequest{
				Messages: messages,
				Model:    m,
				Stream:   true,
			}

			ch, err := s.aiService.ChatStream(ctx, req)
			if err != nil {
				result.Error = err.Error()
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}

			// Collect response
			var responseBuilder strings.Builder
			for chunk := range ch {
				if chunk.Done {
					break
				}
				responseBuilder.WriteString(chunk.Content)
			}

			fullResponse := responseBuilder.String()
			result.FullResponse = fullResponse
			result.Latency = time.Since(startTime).Milliseconds()
			result.TokenCount = len(fullResponse) / 4 // Rough estimate

			// Create preview (first 200 chars)
			if len(fullResponse) > 200 {
				result.ResponsePreview = fullResponse[:200] + "..."
			} else {
				result.ResponsePreview = fullResponse
			}

			// Save the assistant message to the branch
			assistantMsg := &models.Message{
				ConversationID: exploration.ConversationID,
				Role:           "assistant",
				Content:        fullResponse,
				Model:          m,
				Status:         "success",
				BranchID:       &branch.ID,
			}
			s.db.Create(assistantMsg)

			// Update branch message count
			s.db.Model(&models.Branch{}).Where("id = ?", branch.ID).
				UpdateColumn("message_count", gorm.Expr("message_count + 1"))

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(model)
	}

	wg.Wait()

	// Score the results (simple heuristic - can be enhanced with AI scoring)
	bestScore := 0.0
	var bestBranchID string
	for i := range results {
		if results[i].Error == "" {
			// Simple scoring based on response length and latency
			score := float64(len(results[i].FullResponse)) / float64(results[i].Latency+1)
			results[i].Score = &score
			if score > bestScore {
				bestScore = score
				bestBranchID = results[i].BranchID
			}
		}
	}

	// Update exploration with results
	resultsJSON, _ := json.Marshal(results)
	now := time.Now()

	s.db.Model(exploration).Updates(map[string]interface{}{
		"status":         models.ParallelStatusCompleted,
		"results":        resultsJSON,
		"best_branch_id": bestBranchID,
		"completed_at":   &now,
	})
}

// GetParallelExploration returns a parallel exploration session
func (s *BranchService) GetParallelExploration(explorationID string) (*models.ParallelExploration, error) {
	var exploration models.ParallelExploration
	err := s.db.First(&exploration, "id = ?", explorationID).Error
	if err != nil {
		return nil, err
	}
	return &exploration, nil
}

// GetParallelExplorations returns all explorations for a conversation
func (s *BranchService) GetParallelExplorations(conversationID uint) ([]models.ParallelExploration, error) {
	var explorations []models.ParallelExploration
	err := s.db.Where("conversation_id = ?", conversationID).
		Order("created_at desc").
		Find(&explorations).Error
	return explorations, err
}

// SelectExplorationBranch selects a branch from parallel exploration as the active branch
func (s *BranchService) SelectExplorationBranch(conversationID uint, branchID string) error {
	return s.SwitchBranch(conversationID, branchID)
}

// =============================================================================
// Message Tree Operations
// =============================================================================

// GetMessageTree returns the message tree structure for a conversation/branch
func (s *BranchService) GetMessageTree(conversationID uint, branchID *string) (*models.MessageNode, error) {
	var messages []models.Message
	query := s.db.Where("conversation_id = ?", conversationID)

	if branchID != nil {
		query = query.Where("branch_id = ? OR branch_id IS NULL", *branchID)
	}

	if err := query.Order("created_at asc").Find(&messages).Error; err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		return nil, nil
	}

	// Build tree
	nodeMap := make(map[uint]*models.MessageNode)
	var root *models.MessageNode

	for i := range messages {
		msg := &messages[i]
		nodeMap[msg.ID] = &models.MessageNode{
			Message:  msg,
			Children: []*models.MessageNode{},
		}
	}

	// Link children to parents
	for _, node := range nodeMap {
		if node.Message.ParentID != nil {
			if parent, ok := nodeMap[*node.Message.ParentID]; ok {
				parent.Children = append(parent.Children, node)
			}
		} else if root == nil {
			root = node
		}
	}

	return root, nil
}

// GetBranchMessages returns messages for a specific branch
func (s *BranchService) GetBranchMessages(branchID string) ([]models.Message, error) {
	var messages []models.Message
	err := s.db.Where("branch_id = ?", branchID).
		Order("created_at asc").
		Find(&messages).Error
	return messages, err
}

// RegenerateResponse regenerates the AI response for a message
func (s *BranchService) RegenerateResponse(ctx context.Context, msgID uint, model string) (*models.Message, error) {
	// Get the user message
	var userMsg models.Message
	if err := s.db.First(&userMsg, msgID).Error; err != nil {
		return nil, fmt.Errorf("message not found: %w", err)
	}

	if userMsg.Role != "user" {
		return nil, fmt.Errorf("can only regenerate response for user messages")
	}

	// Get conversation history up to this message
	var history []models.Message
	s.db.Where("conversation_id = ? AND created_at < ?", userMsg.ConversationID, userMsg.CreatedAt).
		Order("created_at asc").
		Limit(10).
		Find(&history)

	// Add the user message
	history = append(history, userMsg)

	// Call AI
	req := &ai.ChatRequest{
		Messages: history,
		Model:    model,
		Stream:   true,
	}

	ch, err := s.aiService.ChatStream(ctx, req)
	if err != nil {
		return nil, err
	}

	// Collect response
	var responseBuilder strings.Builder
	for chunk := range ch {
		if chunk.Done {
			break
		}
		responseBuilder.WriteString(chunk.Content)
	}

	// Create new assistant message
	assistantMsg := &models.Message{
		ConversationID: userMsg.ConversationID,
		Role:           "assistant",
		Content:        responseBuilder.String(),
		Model:          model,
		Status:         "success",
		ParentID:       &userMsg.ID,
		BranchID:       userMsg.BranchID,
	}

	if err := s.db.Create(assistantMsg).Error; err != nil {
		return nil, err
	}

	return assistantMsg, nil
}

// RegenerateResponseStream regenerates the AI response with streaming output
func (s *BranchService) RegenerateResponseStream(ctx context.Context, msgID uint, model string) (<-chan ai.ChatChunk, *models.Message, error) {
	// Get the user message
	var userMsg models.Message
	if err := s.db.First(&userMsg, msgID).Error; err != nil {
		return nil, nil, fmt.Errorf("message not found: %w", err)
	}

	if userMsg.Role != "user" {
		return nil, nil, fmt.Errorf("can only regenerate response for user messages")
	}

	// Get conversation history up to this message
	var history []models.Message
	s.db.Where("conversation_id = ? AND created_at < ?", userMsg.ConversationID, userMsg.CreatedAt).
		Order("created_at asc").
		Limit(10).
		Find(&history)

	// Add the user message
	history = append(history, userMsg)

	// Call AI
	req := &ai.ChatRequest{
		Messages: history,
		Model:    model,
		Stream:   true,
	}

	ch, err := s.aiService.ChatStream(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	// Create output channel that wraps the AI channel
	outCh := make(chan ai.ChatChunk)

	go func() {
		defer close(outCh)
		var responseBuilder strings.Builder

		for chunk := range ch {
			if !chunk.Done {
				responseBuilder.WriteString(chunk.Content)
			}
			outCh <- chunk

			if chunk.Done {
				// Save assistant message when done
				assistantMsg := &models.Message{
					ConversationID: userMsg.ConversationID,
					Role:           "assistant",
					Content:        responseBuilder.String(),
					Model:          model,
					Status:         "success",
					ParentID:       &userMsg.ID,
					BranchID:       userMsg.BranchID,
				}
				s.db.Create(assistantMsg)
			}
		}
	}()

	return outCh, &userMsg, nil
}

// MultiRegenerate generates multiple alternative responses
func (s *BranchService) MultiRegenerate(ctx context.Context, msgID uint, model string, count int) ([]models.Message, error) {
	var responses []models.Message
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg, err := s.RegenerateResponse(ctx, msgID, model)
			if err == nil {
				mu.Lock()
				responses = append(responses, *msg)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return responses, nil
}
