package agent

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Error types
var (
	ErrConnectionNotFound = errors.New("connection not found")
	ErrConnectionTimeout  = errors.New("connection timeout")
	ErrResourceExhausted  = errors.New("resource exhausted")
	ErrServerNotReady     = errors.New("server not ready")
	ErrToolNotFound       = errors.New("tool not found")
	ErrInvalidInput       = errors.New("invalid input")
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
)

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxAttempts int
	Backoff     time.Duration
	MaxBackoff  time.Duration
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration
	lastFailure      time.Time
	failureCount     int
	state            CircuitState
	mu               sync.RWMutex
}

type CircuitState string

const (
	StateClosed   CircuitState = "closed"
	StateOpen     CircuitState = "open"
	StateHalfOpen CircuitState = "half-open"
)

// NewCircuitBreaker 创建新的熔断器
func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: threshold,
		resetTimeout:     timeout,
		state:            StateClosed,
	}
}

// Execute 执行操作，受熔断器保护
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.RLock()
	state := cb.state
	cb.mu.RUnlock()

	if state == StateOpen {
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.mu.Lock()
			cb.state = StateHalfOpen
			cb.mu.Unlock()
		} else {
			return ErrCircuitBreakerOpen
		}
	}

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		cb.lastFailure = time.Now()

		if cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
		} else if cb.state == StateHalfOpen {
			cb.state = StateOpen
		}
	} else {
		if cb.state == StateHalfOpen {
			cb.state = StateClosed
			cb.failureCount = 0
		} else {
			cb.failureCount = 0
		}
	}

	return err
}

// GetState 获取熔断器状态
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetFailureCount 获取失败计数
func (cb *CircuitBreaker) GetFailureCount() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failureCount
}

// Reset 重置熔断器
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failureCount = 0
	cb.lastFailure = time.Time{}
}

// RetryWithBackoff 带退避的重试
func RetryWithBackoff(policy RetryPolicy, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < policy.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if attempt < policy.MaxAttempts-1 {
			backoff := policy.Backoff * time.Duration(1<<uint(attempt))
			if backoff > policy.MaxBackoff {
				backoff = policy.MaxBackoff
			}
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", policy.MaxAttempts, lastErr)
}

// DefaultRetryPolicy 默认重试策略
var DefaultRetryPolicy = RetryPolicy{
	MaxAttempts: 3,
	Backoff:     100 * time.Millisecond,
	MaxBackoff:  5 * time.Second,
}

// DefaultCircuitBreakerConfig 默认熔断器配置
var DefaultCircuitBreakerConfig = struct {
	FailureThreshold int
	ResetTimeout     time.Duration
}{
	FailureThreshold: 5,
	ResetTimeout:     30 * time.Second,
}

// IsRetryableError 检查错误是否可重试
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// 连接相关的错误通常可重试
	if errors.Is(err, ErrConnectionNotFound) ||
		errors.Is(err, ErrConnectionTimeout) ||
		errors.Is(err, ErrServerNotReady) {
		return true
	}

	// 网络超时错误可重试
	if err.Error() == "context deadline exceeded" ||
		err.Error() == "i/o timeout" {
		return true
	}

	return false
}

// WrapError 包装错误，添加上下文信息
func WrapError(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}

// ErrorWithStack 创建带堆栈信息的错误
func ErrorWithStack(msg string) error {
	return fmt.Errorf("%s\n%s", msg, getStackTrace())
}

// getStackTrace 获取堆栈跟踪信息
func getStackTrace() string {
	// 简化版本，实际实现可以使用runtime.Caller
	return "stack trace not available in simplified version"
}

// ErrorCollector 错误收集器
type ErrorCollector struct {
	errors []error
	mu     sync.Mutex
}

// NewErrorCollector 创建新的错误收集器
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make([]error, 0),
	}
}

// Add 添加错误
func (ec *ErrorCollector) Add(err error) {
	if err == nil {
		return
	}
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.errors = append(ec.errors, err)
}

// GetErrors 获取所有错误
func (ec *ErrorCollector) GetErrors() []error {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	return ec.errors
}

// HasErrors 检查是否有错误
func (ec *ErrorCollector) HasErrors() bool {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	return len(ec.errors) > 0
}

// Clear 清除所有错误
func (ec *ErrorCollector) Clear() {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.errors = make([]error, 0)
}

// ToError 将错误收集器转换为单个错误
func (ec *ErrorCollector) ToError() error {
	if !ec.HasErrors() {
		return nil
	}

	errors := ec.GetErrors()
	if len(errors) == 1 {
		return errors[0]
	}

	var errorMsgs []string
	for _, err := range errors {
		errorMsgs = append(errorMsgs, err.Error())
	}

	return fmt.Errorf("multiple errors occurred:\n%s", strings.Join(errorMsgs, "\n"))
}
