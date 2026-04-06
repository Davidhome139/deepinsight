package agentsystem

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
	"time"

	"backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PermissionService manages tool access control and auditing
type PermissionService struct {
	db         *gorm.DB
	cache      map[string]*permissionCache
	cacheMu    sync.RWMutex
	rateLimits map[string]*rateLimitTracker
	rateMu     sync.RWMutex
}

type permissionCache struct {
	permissions []models.ToolPermission
	expireAt    time.Time
}

type rateLimitTracker struct {
	counts    map[string]int // toolPattern -> count
	resetAt   time.Time
	windowMin int
}

// NewPermissionService creates a new permission service
func NewPermissionService(db *gorm.DB) *PermissionService {
	return &PermissionService{
		db:         db,
		cache:      make(map[string]*permissionCache),
		rateLimits: make(map[string]*rateLimitTracker),
	}
}

// CreatePermission creates a new tool permission rule
func (s *PermissionService) CreatePermission(ctx context.Context, userID uint, perm *models.ToolPermission) (*models.ToolPermission, error) {
	perm.ID = uuid.New().String()
	perm.UserID = userID

	if err := s.db.WithContext(ctx).Create(perm).Error; err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	// Invalidate cache
	s.invalidateUserCache(userID)

	return perm, nil
}

// GetPermission retrieves a permission by ID
func (s *PermissionService) GetPermission(ctx context.Context, id string, userID uint) (*models.ToolPermission, error) {
	var perm models.ToolPermission
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&perm).Error; err != nil {
		return nil, fmt.Errorf("permission not found: %w", err)
	}
	return &perm, nil
}

// ListPermissions lists all permissions for a user
func (s *PermissionService) ListPermissions(ctx context.Context, userID uint) ([]models.ToolPermission, error) {
	var perms []models.ToolPermission
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&perms).Error; err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}
	return perms, nil
}

// UpdatePermission updates an existing permission
func (s *PermissionService) UpdatePermission(ctx context.Context, id string, userID uint, updates map[string]interface{}) (*models.ToolPermission, error) {
	var perm models.ToolPermission
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&perm).Error; err != nil {
		return nil, fmt.Errorf("permission not found: %w", err)
	}

	if err := s.db.WithContext(ctx).Model(&perm).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update permission: %w", err)
	}

	s.invalidateUserCache(userID)
	return &perm, nil
}

// DeletePermission deletes a permission
func (s *PermissionService) DeletePermission(ctx context.Context, id string, userID uint) error {
	result := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&models.ToolPermission{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete permission: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("permission not found")
	}

	s.invalidateUserCache(userID)
	return nil
}

// CheckPermission checks if a tool invocation is allowed
func (s *PermissionService) CheckPermission(ctx context.Context, userID uint, toolName string, args map[string]interface{}, agentID, workflowID *string) (*PermissionResult, error) {
	perms := s.getUserPermissions(ctx, userID)

	result := &PermissionResult{
		Allowed:   true,
		ToolName:  toolName,
		CheckedAt: time.Now(),
	}

	// Find matching permissions
	var matchingPerms []models.ToolPermission
	for _, perm := range perms {
		if s.matchesPattern(toolName, perm.ToolPattern) {
			// Check scope
			if perm.Scope == "agent" && agentID != nil && perm.ScopeID != nil && *perm.ScopeID != *agentID {
				continue
			}
			if perm.Scope == "workflow" && workflowID != nil && perm.ScopeID != nil && *perm.ScopeID != *workflowID {
				continue
			}
			matchingPerms = append(matchingPerms, perm)
		}
	}

	if len(matchingPerms) == 0 {
		// No explicit permission - allow by default (can be configured)
		return result, nil
	}

	// Check each matching permission
	for _, perm := range matchingPerms {
		if !perm.IsEnabled {
			continue
		}

		// Check expiration
		if perm.ExpiresAt != nil && perm.ExpiresAt.Before(time.Now()) {
			continue
		}

		// Check rate limits
		if rateLimited, reason := s.checkRateLimit(userID, perm); rateLimited {
			result.Allowed = false
			result.DenialReason = reason
			result.PermissionID = &perm.ID
			break
		}

		// Check blocked arguments
		if blocked, reason := s.checkBlockedArgs(perm, args); blocked {
			result.Allowed = false
			result.DenialReason = reason
			result.PermissionID = &perm.ID
			break
		}

		// Check if requires approval
		if perm.RequiresApproval {
			result.RequiresApproval = true
			result.PermissionID = &perm.ID
		}

		result.PermissionID = &perm.ID
		result.AuditLevel = perm.AuditLevel
	}

	// Log the invocation
	go s.logInvocation(ctx, userID, toolName, args, result, agentID, workflowID)

	return result, nil
}

// PermissionResult represents the result of a permission check
type PermissionResult struct {
	Allowed          bool      `json:"allowed"`
	RequiresApproval bool      `json:"requires_approval"`
	ToolName         string    `json:"tool_name"`
	DenialReason     string    `json:"denial_reason,omitempty"`
	PermissionID     *string   `json:"permission_id,omitempty"`
	AuditLevel       string    `json:"audit_level,omitempty"`
	CheckedAt        time.Time `json:"checked_at"`
}

func (s *PermissionService) matchesPattern(toolName, pattern string) bool {
	// Convert glob pattern to regex
	regexPattern := "^" + regexp.QuoteMeta(pattern) + "$"
	regexPattern = regexp.MustCompile(`\\\*`).ReplaceAllString(regexPattern, ".*")
	regexPattern = regexp.MustCompile(`\\\?`).ReplaceAllString(regexPattern, ".")

	matched, err := regexp.MatchString(regexPattern, toolName)
	if err != nil {
		return false
	}
	return matched
}

func (s *PermissionService) checkRateLimit(userID uint, perm models.ToolPermission) (bool, string) {
	var rateLimit map[string]interface{}
	if err := json.Unmarshal(perm.RateLimit, &rateLimit); err != nil {
		return false, ""
	}

	maxPerMinute, _ := rateLimit["maxPerMinute"].(float64)
	if maxPerMinute <= 0 {
		return false, ""
	}

	key := fmt.Sprintf("%d:%s", userID, perm.ID)

	s.rateMu.Lock()
	defer s.rateMu.Unlock()

	tracker, exists := s.rateLimits[key]
	if !exists || time.Now().After(tracker.resetAt) {
		s.rateLimits[key] = &rateLimitTracker{
			counts:    make(map[string]int),
			resetAt:   time.Now().Add(time.Minute),
			windowMin: 1,
		}
		tracker = s.rateLimits[key]
	}

	tracker.counts[perm.ToolPattern]++
	if float64(tracker.counts[perm.ToolPattern]) > maxPerMinute {
		return true, fmt.Sprintf("rate limit exceeded: max %d per minute", int(maxPerMinute))
	}

	return false, ""
}

func (s *PermissionService) checkBlockedArgs(perm models.ToolPermission, args map[string]interface{}) (bool, string) {
	var blockedPatterns []string
	if err := json.Unmarshal(perm.BlockedArgs, &blockedPatterns); err != nil {
		return false, ""
	}

	argsJSON, _ := json.Marshal(args)
	argsStr := string(argsJSON)

	for _, pattern := range blockedPatterns {
		if matched, _ := regexp.MatchString(pattern, argsStr); matched {
			return true, fmt.Sprintf("blocked argument pattern: %s", pattern)
		}
	}

	return false, ""
}

func (s *PermissionService) getUserPermissions(ctx context.Context, userID uint) []models.ToolPermission {
	s.cacheMu.RLock()
	cacheKey := fmt.Sprintf("user:%d", userID)
	cached, exists := s.cache[cacheKey]
	s.cacheMu.RUnlock()

	if exists && time.Now().Before(cached.expireAt) {
		return cached.permissions
	}

	var perms []models.ToolPermission
	s.db.WithContext(ctx).Where("user_id = ? AND is_enabled = true", userID).Find(&perms)

	s.cacheMu.Lock()
	s.cache[cacheKey] = &permissionCache{
		permissions: perms,
		expireAt:    time.Now().Add(5 * time.Minute),
	}
	s.cacheMu.Unlock()

	return perms
}

func (s *PermissionService) invalidateUserCache(userID uint) {
	s.cacheMu.Lock()
	delete(s.cache, fmt.Sprintf("user:%d", userID))
	s.cacheMu.Unlock()
}

func (s *PermissionService) logInvocation(ctx context.Context, userID uint, toolName string, args map[string]interface{}, result *PermissionResult, agentID, workflowID *string) {
	argsJSON, _ := json.Marshal(args)

	status := "allowed"
	if !result.Allowed {
		status = "denied"
	}

	log := &models.ToolInvocationLog{
		ID:           uuid.New().String(),
		UserID:       userID,
		AgentID:      agentID,
		WorkflowID:   workflowID,
		ToolName:     toolName,
		ToolArgs:     argsJSON,
		Status:       status,
		PermissionID: result.PermissionID,
		DenialReason: result.DenialReason,
		CreatedAt:    time.Now(),
	}

	s.db.Create(log)
}

// GetInvocationLogs retrieves tool invocation logs
func (s *PermissionService) GetInvocationLogs(ctx context.Context, userID uint, filter InvocationLogFilter) ([]models.ToolInvocationLog, error) {
	query := s.db.WithContext(ctx).Where("user_id = ?", userID)

	if filter.AgentID != "" {
		query = query.Where("agent_id = ?", filter.AgentID)
	}
	if filter.WorkflowID != "" {
		query = query.Where("workflow_id = ?", filter.WorkflowID)
	}
	if filter.ToolName != "" {
		query = query.Where("tool_name LIKE ?", "%"+filter.ToolName+"%")
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if !filter.StartDate.IsZero() {
		query = query.Where("created_at >= ?", filter.StartDate)
	}
	if !filter.EndDate.IsZero() {
		query = query.Where("created_at <= ?", filter.EndDate)
	}

	var logs []models.ToolInvocationLog
	if err := query.Order("created_at DESC").Limit(filter.Limit).Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}
	return logs, nil
}

// InvocationLogFilter defines filters for log queries
type InvocationLogFilter struct {
	AgentID    string
	WorkflowID string
	ToolName   string
	Status     string
	StartDate  time.Time
	EndDate    time.Time
	Limit      int
}

// GetUsageStats returns tool usage statistics
func (s *PermissionService) GetUsageStats(ctx context.Context, userID uint, days int) (*UsageStats, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	var stats UsageStats

	// Total invocations
	s.db.WithContext(ctx).Model(&models.ToolInvocationLog{}).
		Where("user_id = ? AND created_at >= ?", userID, startDate).
		Count(&stats.TotalInvocations)

	// Allowed vs denied
	s.db.WithContext(ctx).Model(&models.ToolInvocationLog{}).
		Where("user_id = ? AND created_at >= ? AND status = ?", userID, startDate, "allowed").
		Count(&stats.AllowedCount)

	s.db.WithContext(ctx).Model(&models.ToolInvocationLog{}).
		Where("user_id = ? AND created_at >= ? AND status = ?", userID, startDate, "denied").
		Count(&stats.DeniedCount)

	// Top tools
	var topTools []struct {
		ToolName string
		Count    int64
	}
	s.db.WithContext(ctx).Model(&models.ToolInvocationLog{}).
		Select("tool_name, COUNT(*) as count").
		Where("user_id = ? AND created_at >= ?", userID, startDate).
		Group("tool_name").
		Order("count DESC").
		Limit(10).
		Scan(&topTools)

	stats.TopTools = make(map[string]int64)
	for _, t := range topTools {
		stats.TopTools[t.ToolName] = t.Count
	}

	// Daily breakdown
	var dailyStats []struct {
		Date  string
		Count int64
	}
	s.db.WithContext(ctx).Model(&models.ToolInvocationLog{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("user_id = ? AND created_at >= ?", userID, startDate).
		Group("DATE(created_at)").
		Order("date").
		Scan(&dailyStats)

	stats.DailyBreakdown = make(map[string]int64)
	for _, d := range dailyStats {
		stats.DailyBreakdown[d.Date] = d.Count
	}

	return &stats, nil
}

// UsageStats represents tool usage statistics
type UsageStats struct {
	TotalInvocations int64            `json:"total_invocations"`
	AllowedCount     int64            `json:"allowed_count"`
	DeniedCount      int64            `json:"denied_count"`
	TopTools         map[string]int64 `json:"top_tools"`
	DailyBreakdown   map[string]int64 `json:"daily_breakdown"`
}

// CreateDefaultPermissions creates default permission rules for a user
func (s *PermissionService) CreateDefaultPermissions(ctx context.Context, userID uint) error {
	defaults := []models.ToolPermission{
		{
			Name:        "Allow Read Operations",
			Description: "Allow all read-only file operations",
			ToolPattern: "file/read*",
			Actions:     mustMarshal([]string{"read"}),
			Scope:       "global",
			IsEnabled:   true,
			AuditLevel:  "summary",
		},
		{
			Name:        "Restricted Write Operations",
			Description: "Rate-limited write operations",
			ToolPattern: "file/write*",
			Actions:     mustMarshal([]string{"write"}),
			Scope:       "global",
			RateLimit:   mustMarshal(map[string]int{"maxPerMinute": 10, "maxPerHour": 100}),
			IsEnabled:   true,
			AuditLevel:  "detailed",
		},
		{
			Name:        "Block Dangerous Commands",
			Description: "Block potentially dangerous shell commands",
			ToolPattern: "shell/*",
			Actions:     mustMarshal([]string{"execute"}),
			Scope:       "global",
			BlockedArgs: mustMarshal([]string{"rm -rf", "sudo", "chmod 777", "mkfs", "> /dev"}),
			IsEnabled:   true,
			AuditLevel:  "detailed",
		},
		{
			Name:        "MCP External Tools",
			Description: "Allow MCP tools with approval for sensitive operations",
			ToolPattern: "mcp://*/*",
			Actions:     mustMarshal([]string{"read", "write", "execute"}),
			Scope:       "global",
			RateLimit:   mustMarshal(map[string]int{"maxPerMinute": 30}),
			IsEnabled:   true,
			AuditLevel:  "summary",
		},
	}

	for _, perm := range defaults {
		perm.ID = uuid.New().String()
		perm.UserID = userID
		if err := s.db.WithContext(ctx).Create(&perm).Error; err != nil {
			return fmt.Errorf("failed to create default permission: %w", err)
		}
	}

	return nil
}

func mustMarshal(v interface{}) models.JSON {
	data, _ := json.Marshal(v)
	return data
}

// ApprovalRequest represents a pending approval for tool execution
type ApprovalRequest struct {
	ID           string                 `json:"id"`
	UserID       uint                   `json:"user_id"`
	ToolName     string                 `json:"tool_name"`
	Args         map[string]interface{} `json:"args"`
	AgentID      *string                `json:"agent_id"`
	WorkflowID   *string                `json:"workflow_id"`
	PermissionID string                 `json:"permission_id"`
	Status       string                 `json:"status"` // pending, approved, denied
	RequestedAt  time.Time              `json:"requested_at"`
	ResolvedAt   *time.Time             `json:"resolved_at"`
	ResolvedBy   *uint                  `json:"resolved_by"`
}

// RequestApproval creates a new approval request
func (s *PermissionService) RequestApproval(ctx context.Context, userID uint, toolName string, args map[string]interface{}, permissionID string, agentID, workflowID *string) (*ApprovalRequest, error) {
	// In a full implementation, this would store the request and notify the user
	// For now, we return the request structure
	request := &ApprovalRequest{
		ID:           uuid.New().String(),
		UserID:       userID,
		ToolName:     toolName,
		Args:         args,
		AgentID:      agentID,
		WorkflowID:   workflowID,
		PermissionID: permissionID,
		Status:       "pending",
		RequestedAt:  time.Now(),
	}

	return request, nil
}
