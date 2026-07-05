package platform

import (
	"fmt"
	"sync"
	"time"
)

type CircuitState string

const (
	StateClosed CircuitState = "closed"
	StateOpen   CircuitState = "open"
	StateHalfOpen CircuitState = "half-open"
)

type CircuitBreaker struct {
	mu              sync.RWMutex
	state           CircuitState
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	failureThreshold int
	resetTimeout    time.Duration
	halfOpenMaxTries int
}

func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		halfOpenMaxTries: 2,
	}
}

func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check if we should transition from open to half-open
	if cb.state == StateOpen {
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = StateHalfOpen
			cb.successCount = 0
			cb.failureCount = 0
		} else {
			return fmt.Errorf("circuit breaker is open: service unavailable (retry after %v)", cb.resetTimeout-time.Since(cb.lastFailureTime))
		}
	}

	// Try to execute the function
	err := fn()

	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		if cb.state == StateClosed && cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
			return fmt.Errorf("circuit breaker opened after %d failures: %w", cb.failureCount, err)
		}

		if cb.state == StateHalfOpen {
			cb.state = StateOpen
			return fmt.Errorf("circuit breaker reopened: %w", err)
		}

		return err
	}

	// Success case
	if cb.state == StateClosed {
		cb.failureCount = 0
		return nil
	}

	if cb.state == StateHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.halfOpenMaxTries {
			cb.state = StateClosed
			cb.failureCount = 0
		}
		return nil
	}

	return nil
}

func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}
