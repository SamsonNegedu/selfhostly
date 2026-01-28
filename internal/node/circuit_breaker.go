package node

import (
	"fmt"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState string

const (
	StateClosed   CircuitState = "closed"   // Normal operation, requests pass through
	StateOpen     CircuitState = "open"     // Circuit is open, requests fail fast
	StateHalfOpen CircuitState = "half-open" // Testing if service recovered
)

// CircuitBreaker implements the circuit breaker pattern for node communication
type CircuitBreaker struct {
	mu                sync.RWMutex
	circuits          map[string]*Circuit
	threshold         int           // Number of failures before opening circuit
	timeout           time.Duration // How long to wait before attempting half-open
	halfOpenSuccesses int           // Number of successes needed to close circuit in half-open state
}

// Circuit represents a single circuit breaker for a node
type Circuit struct {
	state             CircuitState
	failures          int
	successes         int
	lastFailureTime   time.Time
	lastStateChange   time.Time
	halfOpenSuccesses int
}

// NewCircuitBreaker creates a new circuit breaker manager
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		circuits:          make(map[string]*Circuit),
		threshold:         5,              // Open after 5 consecutive failures
		timeout:           60 * time.Second, // Wait 60s before trying again
		halfOpenSuccesses: 2,              // Need 2 successes to close circuit
	}
}

// IsOpen checks if the circuit is open for a given node
func (cb *CircuitBreaker) IsOpen(nodeID string) bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	circuit, exists := cb.circuits[nodeID]
	if !exists {
		return false // No circuit means closed (allow)
	}

	// If circuit is open, check if we should transition to half-open
	if circuit.state == StateOpen {
		if time.Since(circuit.lastStateChange) > cb.timeout {
			// Transition to half-open
			cb.mu.RUnlock()
			cb.mu.Lock()
			circuit.state = StateHalfOpen
			circuit.halfOpenSuccesses = 0
			circuit.lastStateChange = time.Now()
			cb.mu.Unlock()
			cb.mu.RLock()
			return false // Allow request in half-open state
		}
		return true // Circuit is open, fail fast
	}

	return false // Closed or half-open, allow request
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess(nodeID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	circuit, exists := cb.circuits[nodeID]
	if !exists {
		return // No circuit to update
	}

	circuit.successes++
	circuit.failures = 0 // Reset failure count

	// Handle state transitions
	switch circuit.state {
	case StateHalfOpen:
		circuit.halfOpenSuccesses++
		if circuit.halfOpenSuccesses >= cb.halfOpenSuccesses {
			// Success! Close the circuit
			circuit.state = StateClosed
			circuit.lastStateChange = time.Now()
			circuit.halfOpenSuccesses = 0
		}
	case StateOpen:
		// Should not happen, but just in case
		circuit.state = StateClosed
		circuit.lastStateChange = time.Now()
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure(nodeID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	circuit, exists := cb.circuits[nodeID]
	if !exists {
		// Create new circuit
		circuit = &Circuit{
			state:           StateClosed,
			lastStateChange: time.Now(),
		}
		cb.circuits[nodeID] = circuit
	}

	circuit.failures++
	circuit.lastFailureTime = time.Now()

	// Handle state transitions
	switch circuit.state {
	case StateClosed:
		if circuit.failures >= cb.threshold {
			// Open the circuit
			circuit.state = StateOpen
			circuit.lastStateChange = time.Now()
		}
	case StateHalfOpen:
		// Failure in half-open state, go back to open
		circuit.state = StateOpen
		circuit.lastStateChange = time.Now()
		circuit.halfOpenSuccesses = 0
	}
}

// GetState returns the current state of a circuit
func (cb *CircuitBreaker) GetState(nodeID string) CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	circuit, exists := cb.circuits[nodeID]
	if !exists {
		return StateClosed
	}

	return circuit.state
}

// GetStats returns statistics for a circuit
func (cb *CircuitBreaker) GetStats(nodeID string) CircuitStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	circuit, exists := cb.circuits[nodeID]
	if !exists {
		return CircuitStats{
			State:    StateClosed,
			Failures: 0,
		}
	}

	return CircuitStats{
		State:           circuit.state,
		Failures:        circuit.failures,
		Successes:       circuit.successes,
		LastFailure:     circuit.lastFailureTime,
		LastStateChange: circuit.lastStateChange,
	}
}

// Reset resets a circuit for a specific node
func (cb *CircuitBreaker) Reset(nodeID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	delete(cb.circuits, nodeID)
}

// CircuitStats holds statistics about a circuit
type CircuitStats struct {
	State           CircuitState
	Failures        int
	Successes       int
	LastFailure     time.Time
	LastStateChange time.Time
}

// CircuitOpenError is returned when a circuit is open
type CircuitOpenError struct {
	NodeID string
	Stats  CircuitStats
}

func (e *CircuitOpenError) Error() string {
	return fmt.Sprintf("circuit breaker open for node %s (failures: %d, state: %s)",
		e.NodeID, e.Stats.Failures, e.Stats.State)
}
