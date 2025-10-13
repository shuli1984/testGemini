package logger

import (
	"sync"
)

// InMemoryLogCollector is a thread-safe, in-memory collector for log messages.
type InMemoryLogCollector struct {
	mu       sync.RWMutex
	messages []string
	capacity int
}

// NewInMemoryLogCollector creates a new log collector with a specific capacity.
func NewInMemoryLogCollector(capacity int) *InMemoryLogCollector {
	return &InMemoryLogCollector{
		messages: make([]string, 0, capacity),
		capacity: capacity,
	}
}

// Add adds a new message to the collector.
func (c *InMemoryLogCollector) Add(message string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Add new message to the front
	c.messages = append([]string{message}, c.messages...)

	// Trim if capacity is exceeded
	if len(c.messages) > c.capacity {
		c.messages = c.messages[:c.capacity]
	}
}

// Get returns all the collected messages.
func (c *InMemoryLogCollector) Get() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to avoid race conditions on the slice
	result := make([]string, len(c.messages))
	copy(result, c.messages)
	return result
}
