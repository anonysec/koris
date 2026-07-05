package tui

import (
	"sync"
	"testing"
	"time"
)

func TestRingBuffer_PushAndAll(t *testing.T) {
	rb := newRingBuffer[int](5)

	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	got := rb.All()
	want := []int{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("All() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("All()[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	rb := newRingBuffer[int](3)

	rb.Push(1)
	rb.Push(2)
	rb.Push(3)
	rb.Push(4) // overwrites 1
	rb.Push(5) // overwrites 2

	got := rb.All()
	want := []int{3, 4, 5}
	if len(got) != len(want) {
		t.Fatalf("All() after overflow len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("All()[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestRingBuffer_LenAndCap(t *testing.T) {
	rb := newRingBuffer[int](5)

	if rb.Cap() != 5 {
		t.Errorf("Cap() = %d, want 5", rb.Cap())
	}
	if rb.Len() != 0 {
		t.Errorf("Len() = %d, want 0", rb.Len())
	}

	rb.Push(1)
	rb.Push(2)
	if rb.Len() != 2 {
		t.Errorf("Len() = %d, want 2", rb.Len())
	}

	// Fill and overflow
	rb.Push(3)
	rb.Push(4)
	rb.Push(5)
	rb.Push(6) // overflow
	if rb.Len() != 5 {
		t.Errorf("Len() after overflow = %d, want 5", rb.Len())
	}
}

func TestRingBuffer_Last(t *testing.T) {
	rb := newRingBuffer[int](5)
	rb.Push(10)
	rb.Push(20)
	rb.Push(30)
	rb.Push(40)
	rb.Push(50)

	tests := []struct {
		name string
		n    int
		want []int
	}{
		{"last 3", 3, []int{30, 40, 50}},
		{"last 1", 1, []int{50}},
		{"last all", 5, []int{10, 20, 30, 40, 50}},
		{"last more than count", 10, []int{10, 20, 30, 40, 50}},
		{"last 0", 0, nil},
		{"last negative", -1, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rb.Last(tt.n)
			if len(got) != len(tt.want) {
				t.Fatalf("Last(%d) len = %d, want %d", tt.n, len(got), len(tt.want))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("Last(%d)[%d] = %d, want %d", tt.n, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRingBuffer_Last_AfterOverflow(t *testing.T) {
	rb := newRingBuffer[int](3)
	rb.Push(1)
	rb.Push(2)
	rb.Push(3)
	rb.Push(4) // overwrites 1
	rb.Push(5) // overwrites 2

	got := rb.Last(2)
	want := []int{4, 5}
	if len(got) != len(want) {
		t.Fatalf("Last(2) len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Last(2)[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestRingBuffer_Latest(t *testing.T) {
	rb := newRingBuffer[int](5)

	// Empty buffer
	_, ok := rb.Latest()
	if ok {
		t.Error("Latest() on empty buffer should return false")
	}

	rb.Push(42)
	val, ok := rb.Latest()
	if !ok || val != 42 {
		t.Errorf("Latest() = (%d, %v), want (42, true)", val, ok)
	}

	rb.Push(99)
	val, ok = rb.Latest()
	if !ok || val != 99 {
		t.Errorf("Latest() = (%d, %v), want (99, true)", val, ok)
	}
}

func TestRingBuffer_Latest_AfterOverflow(t *testing.T) {
	rb := newRingBuffer[int](2)
	rb.Push(1)
	rb.Push(2)
	rb.Push(3) // overwrites 1

	val, ok := rb.Latest()
	if !ok || val != 3 {
		t.Errorf("Latest() after overflow = (%d, %v), want (3, true)", val, ok)
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := newRingBuffer[int](5)
	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	rb.Clear()

	if rb.Len() != 0 {
		t.Errorf("Len() after Clear() = %d, want 0", rb.Len())
	}
	if rb.Cap() != 5 {
		t.Errorf("Cap() after Clear() = %d, want 5", rb.Cap())
	}

	got := rb.All()
	if len(got) != 0 {
		t.Errorf("All() after Clear() len = %d, want 0", len(got))
	}

	// Verify buffer is reusable after clear
	rb.Push(10)
	rb.Push(20)
	if rb.Len() != 2 {
		t.Errorf("Len() after re-push = %d, want 2", rb.Len())
	}
	val, ok := rb.Latest()
	if !ok || val != 20 {
		t.Errorf("Latest() after re-push = (%d, %v), want (20, true)", val, ok)
	}
}

func TestRingBuffer_DefaultCapacity(t *testing.T) {
	rb := newRingBuffer[int](0)
	if rb.Cap() != defaultBufferSize {
		t.Errorf("Cap() with 0 capacity = %d, want %d", rb.Cap(), defaultBufferSize)
	}

	rb2 := newRingBuffer[int](-5)
	if rb2.Cap() != defaultBufferSize {
		t.Errorf("Cap() with negative capacity = %d, want %d", rb2.Cap(), defaultBufferSize)
	}
}

func TestRingBuffer_ConcurrentAccess(t *testing.T) {
	rb := newRingBuffer[int](100)
	var wg sync.WaitGroup
	const writers = 10
	const readsPerWriter = 100

	// Concurrent writers
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for i := 0; i < readsPerWriter; i++ {
				rb.Push(base + i)
			}
		}(w * readsPerWriter)
	}

	// Concurrent readers
	for r := 0; r < 5; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				_ = rb.All()
				_ = rb.Len()
				_ = rb.Last(10)
				_, _ = rb.Latest()
			}
		}()
	}

	wg.Wait()

	// After all pushes: 10 writers * 100 items = 1000 items pushed into cap=100
	if rb.Len() != 100 {
		t.Errorf("Len() after concurrent writes = %d, want 100", rb.Len())
	}
}

func TestRingBuffer_ExactlyAtCapacity(t *testing.T) {
	rb := newRingBuffer[int](5)

	// Fill exactly to capacity — should not trigger overflow
	for i := 1; i <= 5; i++ {
		rb.Push(i)
	}

	if rb.Len() != 5 {
		t.Errorf("Len() at exact capacity = %d, want 5", rb.Len())
	}

	got := rb.All()
	want := []int{1, 2, 3, 4, 5}
	if len(got) != len(want) {
		t.Fatalf("All() at capacity len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("All()[%d] = %d, want %d", i, got[i], want[i])
		}
	}

	// Latest should be the last pushed
	val, ok := rb.Latest()
	if !ok || val != 5 {
		t.Errorf("Latest() at capacity = (%d, %v), want (5, true)", val, ok)
	}
}

func TestRingBuffer_OrderingAfterMultipleWraps(t *testing.T) {
	rb := newRingBuffer[int](3)

	// First wrap: push 6 items (wraps twice) into a capacity-3 buffer
	for i := 1; i <= 6; i++ {
		rb.Push(i)
	}

	got := rb.All()
	want := []int{4, 5, 6}
	if len(got) != len(want) {
		t.Fatalf("All() after 2 wraps len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("All()[%d] = %d, want %d", i, got[i], want[i])
		}
	}

	// Push more to wrap again
	rb.Push(7)
	rb.Push(8)
	rb.Push(9)

	got = rb.All()
	want = []int{7, 8, 9}
	if len(got) != len(want) {
		t.Fatalf("All() after 3 wraps len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("All()[%d] = %d, want %d", i, got[i], want[i])
		}
	}

	// Last(2) should give the 2 most recent in order
	last2 := rb.Last(2)
	wantLast := []int{8, 9}
	if len(last2) != 2 {
		t.Fatalf("Last(2) len = %d, want 2", len(last2))
	}
	for i := range wantLast {
		if last2[i] != wantLast[i] {
			t.Errorf("Last(2)[%d] = %d, want %d", i, last2[i], wantLast[i])
		}
	}
}

func TestRingBuffer_ConcurrentReadWrite(t *testing.T) {
	rb := newRingBuffer[int](50)
	var wg sync.WaitGroup
	const iterations = 500

	// Writer goroutine pushes sequential values
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			rb.Push(i)
		}
	}()

	// Reader goroutine checks ordering invariant
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			items := rb.All()
			// Items should be in ascending order (since single writer)
			for j := 1; j < len(items); j++ {
				if items[j] < items[j-1] {
					t.Errorf("ordering violated: items[%d]=%d < items[%d]=%d",
						j, items[j], j-1, items[j-1])
					return
				}
			}
		}
	}()

	wg.Wait()
}

func TestRingBuffer_WithLogEntry(t *testing.T) {
	rb := newRingBuffer[LogEntry](3)

	entries := []LogEntry{
		{Time: time.Now().UTC(), Level: LevelInfo, Component: "api", Message: "started"},
		{Time: time.Now().UTC(), Level: LevelWarn, Component: "db", Message: "slow query"},
		{Time: time.Now().UTC(), Level: LevelError, Component: "auth", Message: "failed"},
	}

	for _, e := range entries {
		rb.Push(e)
	}

	got := rb.All()
	if len(got) != 3 {
		t.Fatalf("All() len = %d, want 3", len(got))
	}
	for i, e := range got {
		if e.Component != entries[i].Component || e.Message != entries[i].Message {
			t.Errorf("All()[%d] component=%s msg=%s, want component=%s msg=%s",
				i, e.Component, e.Message, entries[i].Component, entries[i].Message)
		}
	}

	latest, ok := rb.Latest()
	if !ok || latest.Component != "auth" {
		t.Errorf("Latest() = (%v, %v), want auth entry", latest, ok)
	}

	last2 := rb.Last(2)
	if len(last2) != 2 || last2[0].Component != "db" || last2[1].Component != "auth" {
		t.Errorf("Last(2) = %v, want [db, auth] entries", last2)
	}
}
