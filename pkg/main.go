package pkg

import (
	"net/http"
	"sync/atomic"
	"time"
)

const (
	open int32 = iota
	close
	halfOpen
)

type CircuitBreaker struct {
	state        int32
	ErrorCount   atomic.Uint64
	requestCount atomic.Uint64
	config       *ConfigCircuitBreaker
}

type ConfigCircuitBreaker struct {
	FailureThreshold int32
	ResetTimeout     time.Duration
	SuccessThreshold int32
}

type AtomicCircuit interface {
	NewCircuitBreaker(FailureThreshold int32, SuccessThreshold int32, ResetTimeout time.Duration) *CircuitBreaker
}

func NewCircuitBreaker(FailureThreshold int32, SuccessThreshold int32, ResetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		config: &ConfigCircuitBreaker{
			FailureThreshold: FailureThreshold,
			SuccessThreshold: SuccessThreshold,
			ResetTimeout:     ResetTimeout,
		},
		ErrorCount:   atomic.Uint64{},
		requestCount: atomic.Uint64{},
		state:        1,
	}
}

func (cb *CircuitBreaker) Execute(url string) error {
	return cb.stateMachineHandler(url)
}

func (cb *CircuitBreaker) stateMachineHandler(url string) error {
	div5 := cb.requestCount.Load() % 3
	if div5 == 0 && cb.ErrorCount.Load() > 5 {
		cb.state = 3
	} else if cb.ErrorCount.Load() > 5 {
		cb.state = 0
	} else {
		cb.state = 1
	}

	switch cb.state {
	case open:
		cb.requestCount.Add(1)
		return nil
	case close:
		err := MiddlayercallGrpc(url)
		if err != nil {
			cb.ErrorCount.Add(1)
			cb.requestCount.Add(1)
			return err
		}
		cb.requestCount.Add(1)
		return nil
	case halfOpen:
		err := MiddlayercallGrpc(url)
		if err != nil {
			cb.ErrorCount.Add(1)
			cb.requestCount.Add(1)
			return err
		}
		AtomicDecrementUint64(&cb.ErrorCount)
		cb.requestCount.Add(1)
	}
	return nil
}

func AtomicDecrementUint64(v *atomic.Uint64) {
	for {
		old := v.Load()
		if old == 0 {
			return
		}
		if v.CompareAndSwap(old, old-1) {
			return
		}
	}
}

func MiddlayercallGrpc(url string) error {
	// after implements a interface now only get fuck it
	_, err := http.Get(url)
	if err != nil {
		return err
	}
	return nil
}
