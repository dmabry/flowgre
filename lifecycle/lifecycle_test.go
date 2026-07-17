// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package lifecycle

import (
	"context"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	m := New()

	if m.ctx == nil {
		t.Fatal("expected non-nil context")
	}
	if m.cancel == nil {
		t.Fatal("expected non-nil cancel func")
	}
	if m.wg == nil {
		t.Fatal("expected non-nil WaitGroup")
	}
}

func TestContext(t *testing.T) {
	m := New()
	ctx := m.Context()

	if ctx == nil {
		t.Fatal("expected non-nil context")
	}

	// Context should not be cancelled initially
	select {
	case <-ctx.Done():
		t.Fatal("context should not be cancelled initially")
	default:
		// expected
	}
}

func TestCancel(t *testing.T) {
	m := New()

	// Context should not be cancelled
	select {
	case <-m.ctx.Done():
		t.Fatal("context should not be cancelled before Cancel()")
	default:
		// expected
	}

	m.Cancel()

	// Context should now be cancelled
	select {
	case <-m.ctx.Done():
		// expected
	case <-time.After(time.Second):
		t.Fatal("context should be cancelled after Cancel()")
	}
}

func TestCancelPropagatesToGoroutines(t *testing.T) {
	m := New()
	ctx := m.Context()

	done := make(chan struct{})
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		<-ctx.Done()
		close(done)
	}()

	// Give goroutine time to start
	time.Sleep(50 * time.Millisecond)

	m.Cancel()

	select {
	case <-done:
		// expected — goroutine received cancellation
	case <-time.After(time.Second):
		t.Fatal("goroutine did not receive cancellation signal")
	}
}

func TestWaitGroup(t *testing.T) {
	m := New()
	wg := m.WaitGroup()

	if wg == nil {
		t.Fatal("expected non-nil WaitGroup")
	}

	// WaitGroup should be the same instance on repeated calls
	wg2 := m.WaitGroup()
	if wg != wg2 {
		t.Fatal("WaitGroup should return the same instance")
	}
}

func TestWaitBlocksUntilDone(t *testing.T) {
	m := New()

	m.wg.Add(1)
	go func() {
		time.Sleep(100 * time.Millisecond)
		m.wg.Done()
	}()

	// Wait should return after goroutine completes
	done := make(chan struct{})
	go func() {
		m.Wait()
		close(done)
	}()

	select {
	case <-done:
		// expected
	case <-time.After(time.Second):
		t.Fatal("Wait did not return after goroutine completed")
	}
}

func TestWaitWithCancelledContext(t *testing.T) {
	m := New()

	m.wg.Add(1)
	go func() {
		<-m.ctx.Done()
		m.wg.Done()
	}()

	// Give goroutine time to start
	time.Sleep(50 * time.Millisecond)

	// Cancel should trigger the goroutine to finish, then Wait returns
	m.Cancel()

	done := make(chan struct{})
	go func() {
		m.Wait()
		close(done)
	}()

	select {
	case <-done:
		// expected
	case <-time.After(time.Second):
		t.Fatal("Wait did not return after cancellation")
	}
}

func TestMultipleGoroutines(t *testing.T) {
	m := New()
	ctx := m.Context()

	const goroutines = 5
	results := make(chan int, goroutines)

	for i := 0; i < goroutines; i++ {
		m.wg.Add(1)
		go func(id int) {
			defer m.wg.Done()
			<-ctx.Done()
			results <- id
		}(i)
	}

	// Give goroutines time to start
	time.Sleep(50 * time.Millisecond)

	m.Cancel()

	// Wait for all goroutines to finish
	m.Wait()
	close(results)

	// Verify all goroutines completed
	count := 0
	for range results {
		count++
	}
	if count != goroutines {
		t.Fatalf("expected %d goroutines to complete, got %d", goroutines, count)
	}
}

func TestContextDeadlinePropagation(t *testing.T) {
	m := New()

	// Derive a context with deadline from the manager's context
	derivedCtx, cancel := context.WithTimeout(m.Context(), 100*time.Millisecond)
	defer cancel()

	select {
	case <-derivedCtx.Done():
		if derivedCtx.Err() == context.DeadlineExceeded {
			// expected
		} else {
			t.Fatalf("unexpected error: %v", derivedCtx.Err())
		}
	case <-time.After(time.Second):
		t.Fatal("derived context deadline did not fire")
	}
}

func TestDoubleCancel(t *testing.T) {
	m := New()

	// Calling Cancel() multiple times should not panic
	m.Cancel()
	m.Cancel()
	m.Cancel()
}

func TestWaitGroupConcurrentAccess(t *testing.T) {
	m := New()

	const goroutines = 100
	m.wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			m.wg.Done()
		}()
	}

	done := make(chan struct{})
	go func() {
		m.Wait()
		close(done)
	}()

	select {
	case <-done:
		// expected
	case <-time.After(time.Second):
		t.Fatal("Wait did not return after all goroutines completed")
	}
}

func TestSetupSignalHandlerCreatesChannels(t *testing.T) {
	m := New()
	defer m.Cancel()

	cleanupDone := m.SetupSignalHandler()

	if cleanupDone == nil {
		t.Fatal("expected non-nil cleanup channel")
	}

	// Channel should not be closed or have a value yet
	select {
	case <-cleanupDone:
		t.Fatal("cleanup channel should not have a value yet")
	default:
		// expected
	}
}

func TestSetupSignalHandlerReceivesSignal(t *testing.T) {
	// We can't easily send OS signals in tests, so we verify the
	// SetupSignalHandler creates the expected structure and the
	// goroutine is running by checking that Cancel() still works.
	m := New()

	cleanupDone := m.SetupSignalHandler()

	// Cancel should still work independently of signal handler
	m.Cancel()

	select {
	case <-m.ctx.Done():
		// expected — context was cancelled
	case <-time.After(time.Second):
		t.Fatal("context should be cancelled")
	}

	// cleanupDone may or may not have a value (depends on whether
	// a signal was received), but it should exist
	_ = cleanupDone
}

func TestSetupSignalHandlerStopsAfterCancel(t *testing.T) {
	m := New()
	cleanupDone := m.SetupSignalHandler()
	m.Cancel()

	select {
	case _, ok := <-cleanupDone:
		if ok {
			t.Fatal("cleanup channel should close without a signal value")
		}
	case <-time.After(time.Second):
		t.Fatal("signal handler did not stop after cancellation")
	}
}

func TestContextValuePropagation(t *testing.T) {
	m := New()

	// Verify the context supports the standard context.Context interface
	var _ context.Context = m.Context()

	// Context should have no error initially
	if m.Context().Err() != nil {
		t.Fatal("context should have no error initially")
	}

	// Value should return nil for unknown keys
	if val := m.Context().Value("nonexistent"); val != nil {
		t.Fatal("expected nil value for unknown key")
	}

	// Deadline should return zero time and false
	deadline, ok := m.Context().Deadline()
	if ok {
		t.Fatal("expected no deadline")
	}
	if !deadline.IsZero() {
		t.Fatal("expected zero deadline")
	}
}
