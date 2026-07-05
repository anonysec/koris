package tui

import "sync"

// RingBuffer is a fixed-size, thread-safe circular buffer that overwrites the
// oldest entries when full. It uses sync.RWMutex to allow concurrent reads
// while serializing writes.
type RingBuffer[T any] struct {
	mu    sync.RWMutex
	items []T
	head  int
	count int
	cap   int
}

// newRingBuffer creates a new RingBuffer with the given capacity.
// If capacity is <= 0, it defaults to defaultBufferSize (1000).
func newRingBuffer[T any](capacity int) *RingBuffer[T] {
	if capacity <= 0 {
		capacity = defaultBufferSize
	}
	return &RingBuffer[T]{
		items: make([]T, capacity),
		cap:   capacity,
	}
}

// Push adds an item to the ring buffer, overwriting the oldest entry if full.
func (rb *RingBuffer[T]) Push(item T) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	idx := (rb.head + rb.count) % rb.cap
	if rb.count == rb.cap {
		// Buffer is full, overwrite oldest
		rb.items[rb.head] = item
		rb.head = (rb.head + 1) % rb.cap
	} else {
		rb.items[idx] = item
		rb.count++
	}
}

// All returns a copy of all entries in order from oldest to newest.
func (rb *RingBuffer[T]) All() []T {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	result := make([]T, rb.count)
	for i := 0; i < rb.count; i++ {
		result[i] = rb.items[(rb.head+i)%rb.cap]
	}
	return result
}

// Last returns up to the last n entries (most recent), ordered oldest to newest.
// If n > current count, all entries are returned.
// If n <= 0, an empty slice is returned.
func (rb *RingBuffer[T]) Last(n int) []T {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	if n <= 0 {
		return nil
	}
	if n > rb.count {
		n = rb.count
	}
	result := make([]T, n)
	start := rb.count - n
	for i := 0; i < n; i++ {
		result[i] = rb.items[(rb.head+start+i)%rb.cap]
	}
	return result
}

// Latest returns the most recently added item and true, or the zero value
// and false if the buffer is empty.
func (rb *RingBuffer[T]) Latest() (T, bool) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	var zero T
	if rb.count == 0 {
		return zero, false
	}
	// The most recent item is at position (head + count - 1) % cap
	idx := (rb.head + rb.count - 1) % rb.cap
	return rb.items[idx], true
}

// Clear empties the buffer, resetting it to its initial state.
// The underlying capacity is preserved.
func (rb *RingBuffer[T]) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	var zero T
	for i := range rb.items {
		rb.items[i] = zero
	}
	rb.head = 0
	rb.count = 0
}

// Len returns the current number of entries in the buffer.
func (rb *RingBuffer[T]) Len() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}

// Cap returns the maximum capacity of the buffer.
func (rb *RingBuffer[T]) Cap() int {
	return rb.cap
}
