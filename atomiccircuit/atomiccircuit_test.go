package atomiccircuit

import (
	"errors"
	"testing"
	"time"
)

var errMock = errors.New("mock error")

func createTestCircuitBreaker() *CircuitBreaker {
	return NewCircuitBreaker(3, 2, 100*time.Millisecond)
}

func TestNewCircuitBreaker(t *testing.T) {
	cb := createTestCircuitBreaker()

	if cb.State.Load() != close {
		t.Errorf("Estado inicial deveria ser close, mas foi %v", cb.State.Load())
	}

	if cb.SuccessCount.Load() != 0 {
		t.Errorf("SuccessCount inicial deveria ser 0, mas foi %v", cb.SuccessCount.Load())
	}

	if cb.ErrorCount.Load() != 0 {
		t.Errorf("ErrorCount inicial deveria ser 0, mas foi %v", cb.ErrorCount.Load())
	}

	if cb.RequestCount.Load() != 0 {
		t.Errorf("RequestCount inicial deveria ser 0, mas foi %v", cb.RequestCount.Load())
	}

	if cb.config.FailureThreshold != 3 {
		t.Errorf("FailureThreshold deveria ser 3, mas foi %v", cb.config.FailureThreshold)
	}

	if cb.config.SuccessThreshold != 2 {
		t.Errorf("SuccessThreshold deveria ser 2, mas foi %v", cb.config.SuccessThreshold)
	}

	if cb.config.ResetTimeout != 100*time.Millisecond {
		t.Errorf("ResetTimeout deveria ser 100ms, mas foi %v", cb.config.ResetTimeout)
	}
}

func TestExecuteSuccess(t *testing.T) {
	cb := createTestCircuitBreaker()

	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Deveria retornar sem erro, mas retornou: %v", err)
	}

	if cb.SuccessCount.Load() != 1 {
		t.Errorf("SuccessCount deveria ser 1, mas foi %v", cb.SuccessCount.Load())
	}

	if cb.ErrorCount.Load() != 0 {
		t.Errorf("ErrorCount deveria ser 0, mas foi %v", cb.ErrorCount.Load())
	}

	if cb.RequestCount.Load() != 1 {
		t.Errorf("RequestCount deveria ser 1, mas foi %v", cb.RequestCount.Load())
	}
}

func TestExecuteError(t *testing.T) {
	cb := createTestCircuitBreaker()

	err := cb.Execute(func() error {
		return errMock
	})

	if err != errMock {
		t.Errorf("Deveria retornar errMock, mas retornou: %v", err)
	}

	if cb.SuccessCount.Load() != 0 {
		t.Errorf("SuccessCount deveria ser 0, mas foi %v", cb.SuccessCount.Load())
	}

	if cb.ErrorCount.Load() != 1 {
		t.Errorf("ErrorCount deveria ser 1, mas foi %v", cb.ErrorCount.Load())
	}

	if cb.RequestCount.Load() != 1 {
		t.Errorf("RequestCount deveria ser 1, mas foi %v", cb.RequestCount.Load())
	}
}

func TestTransitionToOpen(t *testing.T) {
	cb := createTestCircuitBreaker()

	for i := 0; i < 3; i++ {
		err := cb.Execute(func() error {
			return errMock
		})

		if err != errMock {
			t.Errorf("Deveria retornar errMock, mas retornou: %v", err)
		}
	}

	if cb.State.Load() != open {
		t.Errorf("Estado deveria ser open, mas foi %v", cb.State.Load())
	}

	err := cb.Execute(func() error {
		return nil
	})

	if err != cb.ErrBreakerOpen {
		t.Errorf("Deveria retornar ErrBreakerOpen, mas retornou: %v", err)
	}
}

func TestTransitionToHalfOpen(t *testing.T) {
	cb := createTestCircuitBreaker()

	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return errMock
		})
	}

	time.Sleep(cb.config.ResetTimeout + 10*time.Millisecond)

	var executado bool
	err := cb.Execute(func() error {
		executado = true
		return nil
	})

	if !executado {
		t.Error("A função deveria ter sido executada no estado half-open")
	}

	if err != nil {
		t.Errorf("Execução em half-open deveria ter sucesso, mas retornou: %v", err)
	}

	if cb.State.Load() != halfOpen {
		t.Errorf("Estado deveria ser halfOpen, mas foi %v", cb.State.Load())
	}
}

func TestTransitionFromHalfOpenToClose(t *testing.T) {
	cb := createTestCircuitBreaker()

	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return errMock
		})
	}

	time.Sleep(cb.config.ResetTimeout + 10*time.Millisecond)

	for i := 0; i < int(cb.config.SuccessThreshold); i++ {
		cb.Execute(func() error {
			return nil
		})
	}

	if cb.State.Load() != close {
		t.Errorf("Estado deveria ser close, mas foi %v", cb.State.Load())
	}

	if cb.SuccessCount.Load() != 0 {
		t.Errorf("SuccessCount deveria ser resetado para 0, mas foi %v", cb.SuccessCount.Load())
	}

	if cb.ErrorCount.Load() != 0 {
		t.Errorf("ErrorCount deveria ser resetado para 0, mas foi %v", cb.ErrorCount.Load())
	}
}

func TestTransitionFromHalfOpenToOpen(t *testing.T) {
	cb := createTestCircuitBreaker()

	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return errMock
		})
	}

	time.Sleep(cb.config.ResetTimeout + 10*time.Millisecond)

	cb.Execute(func() error {
		return errMock
	})

	if cb.State.Load() != open {
		t.Errorf("Estado deveria ser open, mas foi %v", cb.State.Load())
	}
}

func TestConcurrentRequests(t *testing.T) {
	cb := createTestCircuitBreaker()

	for i := 0; i < int(cb.config.FailureThreshold); i++ {
		cb.Execute(func() error {
			return errMock
		})
	}

	if state := cb.State.Load(); state != open {
		t.Errorf("Estado deveria ser open(%d), mas foi %d", open, state)
	}

	initialRequestCount := cb.RequestCount.Load()
	for i := 0; i < 5; i++ {
		err := cb.Execute(func() error {
			t.Error("Esta função não deveria ser executada quando o circuito está aberto")
			return nil
		})

		if err != cb.ErrBreakerOpen {
			t.Errorf("Deveria retornar ErrBreakerOpen, mas retornou: %v", err)
		}
	}

	if cb.RequestCount.Load() != initialRequestCount+5 {
		t.Errorf("RequestCount deveria ser %d, mas foi %v", initialRequestCount+5, cb.RequestCount.Load())
	}
}
