package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// TaskFunc is the function signature for scheduled tasks
type TaskFunc func(ctx context.Context) error

// Task represents a scheduled task
type Task struct {
	ID          string
	Name        string
	Description string
	Schedule    string // cron expression
	Func        TaskFunc
	Enabled     bool
	LastRun     time.Time
	LastError   error
	RunCount    int64
}

// TaskResult contains the result of a task execution
type TaskResult struct {
	TaskID    string
	StartTime time.Time
	EndTime   time.Time
	Success   bool
	Error     error
}

// Scheduler manages scheduled tasks
type Scheduler struct {
	cron      *cron.Cron
	tasks     map[string]*Task
	entryIDs  map[string]cron.EntryID
	results   []TaskResult
	maxResult int
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	logger    *log.Logger
}

// Config holds scheduler configuration
type Config struct {
	Location     *time.Location
	MaxResults   int
	Logger       *log.Logger
	RecoverPanic bool
}

// DefaultConfig returns default scheduler configuration
func DefaultConfig() *Config {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil || loc == nil {
		// Fallback to UTC if timezone data is missing (e.g., Alpine Linux)
		loc = time.UTC
	}
	return &Config{
		Location:     loc,
		MaxResults:   1000,
		Logger:       log.Default(),
		RecoverPanic: true,
	}
}

// New creates a new Scheduler instance
func New(cfg *Config) *Scheduler {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	opts := []cron.Option{
		cron.WithLocation(cfg.Location),
		cron.WithLogger(cron.VerbosePrintfLogger(cfg.Logger)),
	}

	if cfg.RecoverPanic {
		opts = append(opts, cron.WithChain(cron.Recover(cron.DefaultLogger)))
	}

	return &Scheduler{
		cron:      cron.New(opts...),
		tasks:     make(map[string]*Task),
		entryIDs:  make(map[string]cron.EntryID),
		results:   make([]TaskResult, 0),
		maxResult: cfg.MaxResults,
		ctx:       ctx,
		cancel:    cancel,
		logger:    cfg.Logger,
	}
}

// RegisterTask adds a new task to the scheduler
func (s *Scheduler) RegisterTask(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	if !task.Enabled {
		s.tasks[task.ID] = task
		s.logger.Printf("[Scheduler] Registered task %s (disabled): %s", task.ID, task.Name)
		return nil
	}

	// Wrap the task function to track execution
	wrappedFunc := s.wrapTask(task)

	entryID, err := s.cron.AddFunc(task.Schedule, wrappedFunc)
	if err != nil {
		return fmt.Errorf("failed to schedule task %s: %w", task.ID, err)
	}

	s.tasks[task.ID] = task
	s.entryIDs[task.ID] = entryID

	s.logger.Printf("[Scheduler] Registered task %s: %s (schedule: %s)", task.ID, task.Name, task.Schedule)
	return nil
}

// wrapTask creates a wrapper function for task execution tracking
func (s *Scheduler) wrapTask(task *Task) func() {
	return func() {
		startTime := time.Now()
		s.logger.Printf("[Scheduler] Starting task %s: %s", task.ID, task.Name)

		err := task.Func(s.ctx)

		endTime := time.Now()
		duration := endTime.Sub(startTime)

		result := TaskResult{
			TaskID:    task.ID,
			StartTime: startTime,
			EndTime:   endTime,
			Success:   err == nil,
			Error:     err,
		}

		s.mu.Lock()
		task.LastRun = startTime
		task.LastError = err
		task.RunCount++

		// Store result (with max limit)
		s.results = append(s.results, result)
		if len(s.results) > s.maxResult {
			s.results = s.results[1:]
		}
		s.mu.Unlock()

		if err != nil {
			s.logger.Printf("[Scheduler] Task %s failed after %v: %v", task.ID, duration, err)
		} else {
			s.logger.Printf("[Scheduler] Task %s completed in %v", task.ID, duration)
		}
	}
}

// UnregisterTask removes a task from the scheduler
func (s *Scheduler) UnregisterTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[taskID]; !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	if entryID, exists := s.entryIDs[taskID]; exists {
		s.cron.Remove(entryID)
		delete(s.entryIDs, taskID)
	}

	delete(s.tasks, taskID)
	s.logger.Printf("[Scheduler] Unregistered task %s", taskID)
	return nil
}

// EnableTask enables a previously disabled task
func (s *Scheduler) EnableTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	if task.Enabled {
		return nil
	}

	wrappedFunc := s.wrapTask(task)
	entryID, err := s.cron.AddFunc(task.Schedule, wrappedFunc)
	if err != nil {
		return fmt.Errorf("failed to enable task %s: %w", taskID, err)
	}

	task.Enabled = true
	s.entryIDs[taskID] = entryID

	s.logger.Printf("[Scheduler] Enabled task %s", taskID)
	return nil
}

// DisableTask disables a task without removing it
func (s *Scheduler) DisableTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	if !task.Enabled {
		return nil
	}

	if entryID, exists := s.entryIDs[taskID]; exists {
		s.cron.Remove(entryID)
		delete(s.entryIDs, taskID)
	}

	task.Enabled = false
	s.logger.Printf("[Scheduler] Disabled task %s", taskID)
	return nil
}

// RunTask manually triggers a task execution
func (s *Scheduler) RunTask(taskID string) error {
	s.mu.RLock()
	task, exists := s.tasks[taskID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	go s.wrapTask(task)()
	return nil
}

// GetTask returns a task by ID
func (s *Scheduler) GetTask(taskID string) (*Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[taskID]
	return task, exists
}

// ListTasks returns all registered tasks
func (s *Scheduler) ListTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// GetResults returns task execution results
func (s *Scheduler) GetResults(limit int) []TaskResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.results) {
		limit = len(s.results)
	}

	// Return most recent results
	start := len(s.results) - limit
	results := make([]TaskResult, limit)
	copy(results, s.results[start:])
	return results
}

// GetTaskResults returns results for a specific task
func (s *Scheduler) GetTaskResults(taskID string, limit int) []TaskResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []TaskResult
	for i := len(s.results) - 1; i >= 0 && len(results) < limit; i-- {
		if s.results[i].TaskID == taskID {
			results = append(results, s.results[i])
		}
	}
	return results
}

// Start begins the scheduler
func (s *Scheduler) Start() {
	s.cron.Start()
	s.logger.Printf("[Scheduler] Started with %d tasks", len(s.tasks))
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() context.Context {
	s.cancel()
	ctx := s.cron.Stop()
	s.logger.Printf("[Scheduler] Stopped")
	return ctx
}

// IsRunning returns whether the scheduler is running
func (s *Scheduler) IsRunning() bool {
	entries := s.cron.Entries()
	return len(entries) > 0
}

// Stats returns scheduler statistics
type Stats struct {
	TotalTasks    int
	EnabledTasks  int
	DisabledTasks int
	TotalRuns     int64
	SuccessRuns   int64
	FailedRuns    int64
}

// GetStats returns current scheduler statistics
func (s *Scheduler) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := Stats{
		TotalTasks: len(s.tasks),
	}

	for _, task := range s.tasks {
		if task.Enabled {
			stats.EnabledTasks++
		} else {
			stats.DisabledTasks++
		}
		stats.TotalRuns += task.RunCount
	}

	for _, result := range s.results {
		if result.Success {
			stats.SuccessRuns++
		} else {
			stats.FailedRuns++
		}
	}

	return stats
}
