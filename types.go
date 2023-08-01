package circuitbreaker

import (
	"fmt"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return fmt.Sprintf("unknown state: %d", s)
	}
}

type counts struct {
	requests             uint32
	totalSuccesses       uint32
	totalFailures        uint32
	consecutiveSuccesses uint32
	consecutiveFailures  uint32
}

func (c *counts) onRequest() {
	c.requests++
}

func (c *counts) onSuccess() {
	c.totalSuccesses++
	c.consecutiveSuccesses++
	c.consecutiveFailures = 0
}

func (c *counts) onFailure() {
	c.totalFailures++
	c.consecutiveFailures++
	c.consecutiveSuccesses = 0
}

func (c *counts) reset() {
	c.requests = 0
	c.totalSuccesses = 0
	c.totalFailures = 0
	c.consecutiveSuccesses = 0
	c.consecutiveFailures = 0
}

type settings struct {
	maxHalfOpenRequests uint32
	closedResetInterval time.Duration
	openTimeOut         time.Duration
	readyToTrip         func(counts counts) bool
	onStateChange       func(name string, from State, to State)
	isSuccessful        func(err error) bool
}

func (s settings) validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(s.closedResetInterval, validation.Min(time.Duration(1))),
		validation.Field(s.openTimeOut, validation.Min(time.Duration(1))),
	)
}

type SettingsOption func(*settings)

func (o SettingsOption) apply(s *settings) {
	o(s)
}

// WithMaxHalfOpenRequests sets the maximum number of requests allowed to pass
// through the circuit breaker when it is in the half-open state. If the number
// is 0, the circuit breaker will allow only one request.
func WithMaxHalfOpenRequests(max uint32) SettingsOption {
	return SettingsOption(func(s *settings) {
		s.maxHalfOpenRequests = max
	})
}

// WithOpenTimeOut sets the duration of peroid to wait before reseting the counts
// when the circuit breaker is in the closed state.
func WithClosedResetInterval(interval time.Duration) SettingsOption {
	return SettingsOption(func(s *settings) {
		s.closedResetInterval = interval
	})
}

// WithOpenTimeOut sets the duration of peroid to open state
func WithOpenTimeOut(timeout time.Duration) SettingsOption {
	return SettingsOption(func(s *settings) {
		s.openTimeOut = timeout
	})
}

// WithReadyToTrip sets the function to determine whether the circuit breaker should
// transition from the closed state to the open state.
func WithReadyToTrip(fn func(counts counts) bool) SettingsOption {
	return SettingsOption(func(s *settings) {
		s.readyToTrip = fn
	})
}

// WithOnStateChane sets the callback function to be called when the state changes
func WithOnStateChange(fn func(name string, from State, to State)) SettingsOption {
	return SettingsOption(func(s *settings) {
		s.onStateChange = fn
	})
}

// WithIsSuccessful sets the function to determine whether the request is successful
func WithIsSuccessful(fn func(err error) bool) SettingsOption {
	return SettingsOption(func(s *settings) {
		s.isSuccessful = fn
	})
}

func defaultReadyToTrip(counts counts) bool {
	return counts.consecutiveFailures > 5
}

func defaultIsSuccessful(err error) bool {
	return err == nil
}

var defaultSettings = settings{
	maxHalfOpenRequests: 10,
	closedResetInterval: 5 * time.Second,
	openTimeOut:         10 * time.Second,
	readyToTrip:         defaultReadyToTrip,
	onStateChange:       nil,
	isSuccessful:        defaultIsSuccessful,
}
