package logger

import (
	"sync"
	"testing"
)

func TestNewInMemoryLogCollector(t *testing.T) {
	collector := NewInMemoryLogCollector(10)
	if collector == nil {
		t.Fatal("NewInMemoryLogCollector returned nil")
	}
	if collector.capacity != 10 {
		t.Errorf("Expected capacity 10, got %d", collector.capacity)
	}
	if len(collector.messages) != 0 {
		t.Errorf("Expected initial messages to be empty, but got %d messages", len(collector.messages))
	}
}

func TestAddAndGet(t *testing.T) {
	collector := NewInMemoryLogCollector(3)
	collector.Add("message 1")
	collector.Add("message 2")

	messages := collector.Get()
	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(messages))
	}
	if messages[0] != "message 2" || messages[1] != "message 1" {
		t.Errorf("Messages not in correct order: %v", messages)
	}
}

func TestAddCapacity(t *testing.T) {
	collector := NewInMemoryLogCollector(2)
	collector.Add("message 1")
	collector.Add("message 2")
	collector.Add("message 3")

	messages := collector.Get()
	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages after exceeding capacity, got %d", len(messages))
	}
	if messages[0] != "message 3" || messages[1] != "message 2" {
		t.Errorf("Messages not trimmed correctly: %v", messages)
	}
}

func TestGetEmpty(t *testing.T) {
	collector := NewInMemoryLogCollector(5)
	messages := collector.Get()
	if len(messages) != 0 {
		t.Errorf("Expected empty slice for new collector, got %d messages", len(messages))
	}
}

func TestGetIsCopy(t *testing.T) {
	collector := NewInMemoryLogCollector(5)
	collector.Add("message 1")
	messages := collector.Get()
	messages[0] = "modified"

	originalMessages := collector.Get()
	if originalMessages[0] == "modified" {
		t.Error("Get() should return a copy of the messages, but the original was modified.")
	}
}

func TestThreadSafety(t *testing.T) {
	collector := NewInMemoryLogCollector(100)
	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			collector.Add("some message")
		}()
	}

	wg.Wait()

	messages := collector.Get()
	if len(messages) != numGoroutines {
		t.Errorf("Expected %d messages after concurrent adds, got %d", numGoroutines, len(messages))
	}
}
