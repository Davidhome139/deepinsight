package agent

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestErrorTypes(t *testing.T) {
	assert.Equal(t, "connection not found", ErrConnectionNotFound.Error())
	assert.Equal(t, "connection timeout", ErrConnectionTimeout.Error())
	assert.Equal(t, "resource exhausted", ErrResourceExhausted.Error())
	assert.Equal(t, "server not ready", ErrServerNotReady.Error())
	assert.Equal(t, "tool not found", ErrToolNotFound.Error())
	assert.Equal(t, "invalid input", ErrInvalidInput.Error())
	assert.Equal(t, "circuit breaker is open", ErrCircuitBreakerOpen.Error())
}

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second)
	assert.NotNil(t, cb)
	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, 0, cb.GetFailureCount())
}

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second)
	
	called := false
	err := cb.Execute(func() error {
		called = true
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, 0, cb.GetFailureCount())
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond) // 短超时便于测试
	
	testErr := errors.New("test error")
	
	// 第一次失败
	err := cb.Execute(func() error {
		return testErr
	})
	assert.Equal(t, testErr, err)
	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, 1, cb.GetFailureCount())

	// 第二次失败，触发熔断
	err = cb.Execute(func() error {
		return testErr
	})
	assert.Equal(t, testErr, err)
	assert.Equal(t, StateOpen, cb.GetState())
	assert.Equal(t, 2, cb.GetFailureCount())

	// 第三次尝试，熔断器打开
	err = cb.Execute(func() error {
		return nil // 这个函数不会被调用
	})
	assert.Equal(t, ErrCircuitBreakerOpen, err)
	assert.Equal(t, StateOpen, cb.GetState())
}

func TestCircuitBreaker_Execute_Recovery(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond) // 短超时便于测试
	
	// 触发熔断
	cb.Execute(func() error { return errors.New("error1") })
	cb.Execute(func() error { return errors.New("error2") })
	
	assert.Equal(t, StateOpen, cb.GetState())

	// 等待超时
	time.Sleep(150 * time.Millisecond)

	// 现在应该进入半开状态
	called := false
	err := cb.Execute(func() error {
		called = true
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, 0, cb.GetFailureCount())
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(3, 30*time.Second)
	
	// 模拟一些失败
	cb.Execute(func() error { return errors.New("error") })
	cb.Execute(func() error { return errors.New("error") })
	
	assert.Equal(t, 2, cb.GetFailureCount())

	// 重置
	cb.Reset()
	
	assert.Equal(t, StateClosed, cb.GetState())
	assert.Equal(t, 0, cb.GetFailureCount())
}

func TestRetryWithBackoff_Success(t *testing.T) {
	policy := RetryPolicy{
		MaxAttempts: 3,
		Backoff:     10 * time.Millisecond,
		MaxBackoff:  100 * time.Millisecond,
	}

	attempts := 0
	err := RetryWithBackoff(policy, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary error")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, attempts)
}

func TestRetryWithBackoff_Failure(t *testing.T) {
	policy := RetryPolicy{
		MaxAttempts: 3,
		Backoff:     10 * time.Millisecond,
		MaxBackoff:  100 * time.Millisecond,
	}

	testErr := errors.New("persistent error")
	attempts := 0
	err := RetryWithBackoff(policy, func() error {
		attempts++
		return testErr
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 3 attempts")
	assert.Equal(t, 3, attempts)
}

func TestRetryWithBackoff_NoRetry(t *testing.T) {
	policy := RetryPolicy{
		MaxAttempts: 3,
		Backoff:     10 * time.Millisecond,
		MaxBackoff:  100 * time.Millisecond,
	}

	attempts := 0
	err := RetryWithBackoff(policy, func() error {
		attempts++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, attempts)
}

func TestIsRetryableError(t *testing.T) {
	// 可重试的错误
	assert.True(t, IsRetryableError(ErrConnectionNotFound))
	assert.True(t, IsRetryableError(ErrConnectionTimeout))
	assert.True(t, IsRetryableError(ErrServerNotReady))
	
	// 模拟网络超时错误
	assert.True(t, IsRetryableError(errors.New("context deadline exceeded")))
	assert.True(t, IsRetryableError(errors.New("i/o timeout")))
	
	// 不可重试的错误
	assert.False(t, IsRetryableError(ErrInvalidInput))
	assert.False(t, IsRetryableError(ErrToolNotFound))
	assert.False(t, IsRetryableError(nil))
	assert.False(t, IsRetryableError(errors.New("other error")))
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := WrapError(originalErr, "context")

	assert.Error(t, wrappedErr)
	assert.Contains(t, wrappedErr.Error(), "context")
	assert.Contains(t, wrappedErr.Error(), "original error")
}

func TestWrapError_Nil(t *testing.T) {
	err := WrapError(nil, "context")
	assert.NoError(t, err)
}

func TestErrorCollector(t *testing.T) {
	ec := NewErrorCollector()
	
	assert.False(t, ec.HasErrors())
	assert.Empty(t, ec.GetErrors())

	// 添加错误
	err1 := errors.New("error1")
	err2 := errors.New("error2")
	
	ec.Add(err1)
	ec.Add(err2)
	ec.Add(nil) // nil应该被忽略
	
	assert.True(t, ec.HasErrors())
	assert.Equal(t, 2, len(ec.GetErrors()))
	assert.Equal(t, err1, ec.GetErrors()[0])
	assert.Equal(t, err2, ec.GetErrors()[1])

	// 转换为单个错误
	combinedErr := ec.ToError()
	assert.Error(t, combinedErr)
	assert.Contains(t, combinedErr.Error(), "multiple errors")
	assert.Contains(t, combinedErr.Error(), "error1")
	assert.Contains(t, combinedErr.Error(), "error2")

	// 清除错误
	ec.Clear()
	assert.False(t, ec.HasErrors())
	assert.Empty(t, ec.GetErrors())
}

func TestErrorCollector_SingleError(t *testing.T) {
	ec := NewErrorCollector()
	
	err := errors.New("single error")
	ec.Add(err)
	
	combinedErr := ec.ToError()
	assert.Equal(t, err, combinedErr)
}

func TestErrorCollector_NoErrors(t *testing.T) {
	ec := NewErrorCollector()
	
	assert.NoError(t, ec.ToError())
}

func TestDefaultRetryPolicy(t *testing.T) {
	assert.Equal(t, 3, DefaultRetryPolicy.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, DefaultRetryPolicy.Backoff)
	assert.Equal(t, 5*time.Second, DefaultRetryPolicy.MaxBackoff)
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	assert.Equal(t, 5, DefaultCircuitBreakerConfig.FailureThreshold)
	assert.Equal(t, 30*time.Second, DefaultCircuitBreakerConfig.ResetTimeout)
}