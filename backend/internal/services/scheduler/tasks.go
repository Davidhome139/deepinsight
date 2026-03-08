package scheduler

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"
)

// MaintenanceTasks contains all maintenance task definitions
type MaintenanceTasks struct {
	db        *gorm.DB
	logger    *log.Logger
	dataDir   string
	logDir    string
	tempDir   string
}

// NewMaintenanceTasks creates a new MaintenanceTasks instance
func NewMaintenanceTasks(db *gorm.DB, dataDir string) *MaintenanceTasks {
	return &MaintenanceTasks{
		db:      db,
		logger:  log.Default(),
		dataDir: dataDir,
		logDir:  filepath.Join(dataDir, "logs"),
		tempDir: filepath.Join(dataDir, "temp"),
	}
}

// RegisterAll registers all maintenance tasks with the scheduler
func (m *MaintenanceTasks) RegisterAll(s *Scheduler) error {
	tasks := []*Task{
		// Daily tasks - run at 2:00 AM
		{
			ID:          "daily_log_rotation",
			Name:        "Log Rotation",
			Description: "Rotate and compress old log files",
			Schedule:    "0 2 * * *", // 2:00 AM daily
			Func:        m.LogRotation,
			Enabled:     true,
		},
		{
			ID:          "daily_temp_cleanup",
			Name:        "Temp File Cleanup",
			Description: "Clean up temporary files older than 24 hours",
			Schedule:    "0 3 * * *", // 3:00 AM daily
			Func:        m.TempFileCleanup,
			Enabled:     true,
		},
		{
			ID:          "daily_session_cleanup",
			Name:        "Session Cleanup",
			Description: "Remove expired sessions from database",
			Schedule:    "0 4 * * *", // 4:00 AM daily
			Func:        m.SessionCleanup,
			Enabled:     true,
		},

		// Weekly tasks - run on Sunday
		{
			ID:          "weekly_db_optimize",
			Name:        "Database Optimization",
			Description: "Run VACUUM ANALYZE on PostgreSQL tables",
			Schedule:    "0 3 * * 0", // 3:00 AM Sunday
			Func:        m.DatabaseOptimization,
			Enabled:     true,
		},
		{
			ID:          "weekly_analytics_aggregate",
			Name:        "Analytics Aggregation",
			Description: "Aggregate daily analytics into weekly summaries",
			Schedule:    "0 4 * * 0", // 4:00 AM Sunday
			Func:        m.WeeklyAnalyticsAggregation,
			Enabled:     true,
		},

		// Monthly tasks - run on 1st of month
		{
			ID:          "monthly_stats_aggregate",
			Name:        "Monthly Statistics",
			Description: "Aggregate usage statistics for the month",
			Schedule:    "0 5 1 * *", // 5:00 AM on 1st of month
			Func:        m.MonthlyStatsAggregation,
			Enabled:     true,
		},
		{
			ID:          "monthly_old_data_archive",
			Name:        "Data Archive",
			Description: "Archive old data to cold storage",
			Schedule:    "0 6 1 * *", // 6:00 AM on 1st of month
			Func:        m.ArchiveOldData,
			Enabled:     true,
		},

		// Hourly health checks
		{
			ID:          "hourly_health_check",
			Name:        "System Health Check",
			Description: "Check system health and report anomalies",
			Schedule:    "0 * * * *", // Every hour
			Func:        m.HealthCheck,
			Enabled:     true,
		},
	}

	for _, task := range tasks {
		if err := s.RegisterTask(task); err != nil {
			return fmt.Errorf("failed to register task %s: %w", task.ID, err)
		}
	}

	return nil
}

// LogRotation rotates log files
func (m *MaintenanceTasks) LogRotation(ctx context.Context) error {
	m.logger.Println("[LogRotation] Starting log rotation")

	if _, err := os.Stat(m.logDir); os.IsNotExist(err) {
		m.logger.Println("[LogRotation] Log directory does not exist, skipping")
		return nil
	}

	cutoff := time.Now().AddDate(0, 0, -7) // Keep 7 days of logs
	rotatedCount := 0

	err := filepath.Walk(m.logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if info.IsDir() {
			return nil
		}

		// Rotate files older than cutoff
		if info.ModTime().Before(cutoff) {
			archivePath := path + ".archived"
			if err := os.Rename(path, archivePath); err != nil {
				m.logger.Printf("[LogRotation] Failed to archive %s: %v", path, err)
				return nil
			}
			rotatedCount++
		}

		return nil
	})

	m.logger.Printf("[LogRotation] Rotated %d log files", rotatedCount)
	return err
}

// TempFileCleanup removes old temporary files
func (m *MaintenanceTasks) TempFileCleanup(ctx context.Context) error {
	m.logger.Println("[TempCleanup] Starting temp file cleanup")

	if _, err := os.Stat(m.tempDir); os.IsNotExist(err) {
		m.logger.Println("[TempCleanup] Temp directory does not exist, skipping")
		return nil
	}

	cutoff := time.Now().Add(-24 * time.Hour) // Remove files older than 24 hours
	removedCount := 0
	var removedSize int64

	err := filepath.Walk(m.tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if info.IsDir() {
			return nil
		}

		if info.ModTime().Before(cutoff) {
			size := info.Size()
			if err := os.Remove(path); err != nil {
				m.logger.Printf("[TempCleanup] Failed to remove %s: %v", path, err)
				return nil
			}
			removedCount++
			removedSize += size
		}

		return nil
	})

	m.logger.Printf("[TempCleanup] Removed %d files (%.2f MB)", removedCount, float64(removedSize)/1024/1024)
	return err
}

// SessionCleanup removes expired sessions
func (m *MaintenanceTasks) SessionCleanup(ctx context.Context) error {
	m.logger.Println("[SessionCleanup] Starting session cleanup")

	if m.db == nil {
		m.logger.Println("[SessionCleanup] Database not available, skipping")
		return nil
	}

	// Delete sessions older than 30 days
	cutoff := time.Now().AddDate(0, 0, -30)

	result := m.db.WithContext(ctx).
		Exec("DELETE FROM sessions WHERE updated_at < ?", cutoff)

	if result.Error != nil {
		return fmt.Errorf("failed to cleanup sessions: %w", result.Error)
	}

	m.logger.Printf("[SessionCleanup] Removed %d expired sessions", result.RowsAffected)
	return nil
}

// DatabaseOptimization runs VACUUM ANALYZE on the database
func (m *MaintenanceTasks) DatabaseOptimization(ctx context.Context) error {
	m.logger.Println("[DBOptimize] Starting database optimization")

	if m.db == nil {
		m.logger.Println("[DBOptimize] Database not available, skipping")
		return nil
	}

	// Get list of tables to optimize
	var tables []string
	result := m.db.WithContext(ctx).Raw(`
		SELECT tablename FROM pg_tables 
		WHERE schemaname = 'public'
	`).Scan(&tables)

	if result.Error != nil {
		return fmt.Errorf("failed to get tables: %w", result.Error)
	}

	for _, table := range tables {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		m.logger.Printf("[DBOptimize] Running VACUUM ANALYZE on %s", table)
		if err := m.db.WithContext(ctx).Exec(fmt.Sprintf("VACUUM ANALYZE %s", table)).Error; err != nil {
			m.logger.Printf("[DBOptimize] Warning: VACUUM failed for %s: %v", table, err)
		}
	}

	m.logger.Printf("[DBOptimize] Optimized %d tables", len(tables))
	return nil
}

// WeeklyAnalyticsAggregation aggregates daily analytics
func (m *MaintenanceTasks) WeeklyAnalyticsAggregation(ctx context.Context) error {
	m.logger.Println("[WeeklyAnalytics] Starting weekly aggregation")

	if m.db == nil {
		m.logger.Println("[WeeklyAnalytics] Database not available, skipping")
		return nil
	}

	// Aggregate last week's daily data into weekly summary
	weekStart := time.Now().AddDate(0, 0, -7).Truncate(24 * time.Hour)
	weekEnd := time.Now().Truncate(24 * time.Hour)

	result := m.db.WithContext(ctx).Exec(`
		INSERT INTO analytics_weekly (week_start, week_end, total_requests, total_tokens, unique_users, created_at)
		SELECT 
			? as week_start,
			? as week_end,
			COALESCE(SUM(request_count), 0) as total_requests,
			COALESCE(SUM(token_count), 0) as total_tokens,
			COALESCE(COUNT(DISTINCT user_id), 0) as unique_users,
			NOW() as created_at
		FROM analytics_daily
		WHERE date >= ? AND date < ?
		ON CONFLICT (week_start) DO UPDATE SET
			total_requests = EXCLUDED.total_requests,
			total_tokens = EXCLUDED.total_tokens,
			unique_users = EXCLUDED.unique_users
	`, weekStart, weekEnd, weekStart, weekEnd)

	if result.Error != nil {
		// Table might not exist yet
		m.logger.Printf("[WeeklyAnalytics] Warning: %v", result.Error)
		return nil
	}

	m.logger.Printf("[WeeklyAnalytics] Aggregation complete")
	return nil
}

// MonthlyStatsAggregation aggregates monthly statistics
func (m *MaintenanceTasks) MonthlyStatsAggregation(ctx context.Context) error {
	m.logger.Println("[MonthlyStats] Starting monthly aggregation")

	if m.db == nil {
		m.logger.Println("[MonthlyStats] Database not available, skipping")
		return nil
	}

	// Get last month's start and end
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
	monthEnd := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	result := m.db.WithContext(ctx).Exec(`
		INSERT INTO analytics_monthly (month_start, month_end, total_requests, total_tokens, unique_users, avg_response_time, created_at)
		SELECT 
			? as month_start,
			? as month_end,
			COALESCE(SUM(total_requests), 0) as total_requests,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(unique_users), 0) as unique_users,
			COALESCE(AVG(total_requests), 0) as avg_response_time,
			NOW() as created_at
		FROM analytics_weekly
		WHERE week_start >= ? AND week_start < ?
		ON CONFLICT (month_start) DO UPDATE SET
			total_requests = EXCLUDED.total_requests,
			total_tokens = EXCLUDED.total_tokens,
			unique_users = EXCLUDED.unique_users
	`, monthStart, monthEnd, monthStart, monthEnd)

	if result.Error != nil {
		m.logger.Printf("[MonthlyStats] Warning: %v", result.Error)
		return nil
	}

	m.logger.Printf("[MonthlyStats] Monthly aggregation complete")
	return nil
}

// ArchiveOldData archives data older than retention period
func (m *MaintenanceTasks) ArchiveOldData(ctx context.Context) error {
	m.logger.Println("[Archive] Starting old data archive")

	if m.db == nil {
		m.logger.Println("[Archive] Database not available, skipping")
		return nil
	}

	// Archive chat history older than 90 days
	cutoff := time.Now().AddDate(0, 0, -90)

	// First, export to archive table
	result := m.db.WithContext(ctx).Exec(`
		INSERT INTO chat_history_archive (id, conversation_id, role, content, created_at, archived_at)
		SELECT id, conversation_id, role, content, created_at, NOW()
		FROM chat_history
		WHERE created_at < ?
		ON CONFLICT (id) DO NOTHING
	`, cutoff)

	if result.Error != nil {
		m.logger.Printf("[Archive] Warning during archive: %v", result.Error)
	}

	// Then delete from main table
	result = m.db.WithContext(ctx).Exec(`
		DELETE FROM chat_history WHERE created_at < ?
	`, cutoff)

	if result.Error != nil {
		m.logger.Printf("[Archive] Warning during cleanup: %v", result.Error)
		return nil
	}

	m.logger.Printf("[Archive] Archived %d records", result.RowsAffected)
	return nil
}

// HealthCheck performs system health verification
func (m *MaintenanceTasks) HealthCheck(ctx context.Context) error {
	m.logger.Println("[HealthCheck] Starting health check")

	var issues []string

	// Check database connection
	if m.db != nil {
		sqlDB, err := m.db.DB()
		if err != nil {
			issues = append(issues, fmt.Sprintf("DB error: %v", err))
		} else if err := sqlDB.PingContext(ctx); err != nil {
			issues = append(issues, fmt.Sprintf("DB ping failed: %v", err))
		}
	}

	// Check disk space (data directory)
	if m.dataDir != "" {
		// Get disk usage - simplified check
		var stat struct{}
		if _, err := os.Stat(m.dataDir); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("Data dir not accessible: %v", err))
		} else {
			_ = stat // Placeholder for actual disk usage check
		}
	}

	if len(issues) > 0 {
		m.logger.Printf("[HealthCheck] Issues found: %v", issues)
		return fmt.Errorf("health check found %d issues: %v", len(issues), issues)
	}

	m.logger.Println("[HealthCheck] All checks passed")
	return nil
}
