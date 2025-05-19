package atomiccircuit

import (
	"errors"
	"log/slog"
	"sync/atomic"
	"time"
)

const (
	close int32 = iota
	open
	halfOpen
)

var logger = slog.Default()

type CircuitBreaker struct {
	State           atomic.Int32
	SuccessCount    atomic.Uint64
	ErrorCount      atomic.Uint64
	RequestCount    atomic.Uint64
	LastStateChange atomic.Value
	ErrBreakerOpen  error
	config          *ConfigCircuitBreaker
}

type ConfigCircuitBreaker struct {
	FailureThreshold int64
	ResetTimeout     time.Duration
	SuccessThreshold uint64
}

func NewCircuitBreaker(FailureThreshold int64, SuccessThreshold uint64, ResetTimeout time.Duration) *CircuitBreaker {
	cb := &CircuitBreaker{
		config: &ConfigCircuitBreaker{
			FailureThreshold: FailureThreshold,
			SuccessThreshold: SuccessThreshold,
			ResetTimeout:     ResetTimeout,
		},
		ErrorCount:      atomic.Uint64{},
		SuccessCount:    atomic.Uint64{},
		RequestCount:    atomic.Uint64{},
		State:           atomic.Int32{},
		LastStateChange: atomic.Value{},
		ErrBreakerOpen:  errors.New("circuit: breaker open"),
	}
	cb.State.Add(close)
	cb.LastStateChange.Store(time.Now())
	return cb
}

func (cb *CircuitBreaker) Execute(req func() error) error {
	state := cb.State.Load()

	if state == open {
		lastChange := cb.LastStateChange.Load().(time.Time)
		timeElapsed := time.Since(lastChange)
		if timeElapsed > cb.config.ResetTimeout {
			if cb.State.CompareAndSwap(open, halfOpen) {
				cb.LastStateChange.Store(time.Now())
				logger.Debug("Transicionando para Half-Open após timeout")
				state = halfOpen
			}
		} else {
			cb.RequestCount.Add(1)
			return cb.ErrBreakerOpen
		}
	}

	var err error

	switch state {
	case close:
		err = req()
		cb.RequestCount.Add(1)

		if err != nil {
			cb.ErrorCount.Add(1)

			if cb.ErrorCount.Load() >= uint64(cb.config.FailureThreshold) {
				if cb.State.CompareAndSwap(close, open) {
					cb.LastStateChange.Store(time.Now())
					logger.Debug("Transicionando para Open devido a muitas falhas")
				}
			}
		} else {
			cb.SuccessCount.Add(1)
		}

	case halfOpen:
		err = req()
		cb.RequestCount.Add(1)

		if err != nil {
			cb.ErrorCount.Add(1)
			if cb.State.CompareAndSwap(halfOpen, open) {
				cb.LastStateChange.Store(time.Now())
				logger.Debug("Transicionando para Open devido a falha no teste")
			}
		} else {
			cb.SuccessCount.Add(1)

			if cb.SuccessCount.Load() >= cb.config.SuccessThreshold {
				if cb.State.CompareAndSwap(halfOpen, close) {
					cb.ErrorCount.Store(0)
					cb.SuccessCount.Store(0)
					cb.LastStateChange.Store(time.Now())
					logger.Debug("Transicionando para Close devido a sucesso no teste")
				}
			}
		}

	case open:
		cb.RequestCount.Add(1)
		return cb.ErrBreakerOpen
	}

	return err
}
