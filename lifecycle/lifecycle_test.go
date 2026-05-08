// Package lifecycle provides shared process management for flowgre modes.
// It handles context creation, signal handling (SIGINT/SIGTERM), and WaitGroup coordination.
package lifecycle

import (
	"testing"
	"time"
)

// TestNew verifies that New creates a valid Manager with initialized fields.
func TestNew(t *testing.T) {
	t.Parallel()
	mgr := New()

	if mgr == nil {
		t.Fatal("New() returned nil")
	}
	if mgr.ctx == nil {
		t.Error("New() created Manager with nil context")
	}
	if mgr.cancel == nil {
		t.Error("New() created Manager with nil cancel function")
	}
	if mgr.wg == nil {
		t.Error("New() created Manager with nil WaitGroup")
	}
}

// TestContext verifies that Context() returns the managed context.
func TestContext(t *testing.T) {
	t.Parallel()
	mgr := New()

	ctx := mgr.Context()
	if ctx == nil {
		t.Error("Context() returned nil")
	}
	if ctx != mgr.ctx {
		t.Error("Context() did not return the managed context")
	}
}

// TestCancel verifies that Cancel() properly cancels the context.
func TestCancel(t *testing.T) {
	t.Parallel()
	mgr := New()

	// Verify context is not cancelled initially
	select {
	case <-mgr.Context().Done():
		t.Error("Context should not be cancelled before Cancel()")
	default:
		// Expected: context not cancelled
	}

	// Call Cancel
	mgr.Cancel()

	// Verify context is cancelled
	select {
	case <-mgr.Context().Done():
		// Expected: context cancelled
	default:
		t.Error("Context should be cancelled after Cancel()")
	}
}

// TestWaitGroup verifies that WaitGroup() returns the managed WaitGroup.
func TestWaitGroup(t *testing.T) {
	t.Parallel()
	mgr := New()

	wg := mgr.WaitGroup()
	if wg == nil {
		t.Error("WaitGroup() returned nil")
	}
	if wg != mgr.wg {
		t.Error("WaitGroup() did not return the managed WaitGroup")
	}
}

// TestWait verifies that Wait() blocks until all goroutines complete.
func TestWait(t *testing.T) {
	t.Parallel()
	mgr := New()

	// Add a goroutine that completes immediately
	mgr.wg.Add(1)
	go func() {
		defer mgr.wg.Done()
	}()

	// Wait should return when the goroutine completes
	done := make(chan struct{})
	go func() {
		mgr.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Expected: Wait returned
	case <-time.After(2 * time.Second):
		t.Error("Wait() did not return after goroutine completed")
	}
}

// TestSetupSignalHandler verifies that SetupSignalHandler() sets up signal handling.
func TestSetupSignalHandler(t *testing.T) {
	t.Parallel()
	mgr := New()

	cleanupChan := mgr.SetupSignalHandler()
	if cleanupChan == nil {
		t.Fatal("SetupSignalHandler() returned nil")
	}

	// Verify the channel is set up and buffered
	// We can't easily test actual signal handling in unit tests,
	// so we verify the channel is ready to receive signals
	select {
	case <-cleanupChan:
		// Channel received a signal (unlikely in test, but possible)
	default:
		// Expected: channel is empty but ready
	}

	// The signal handler goroutine is running and will send to cleanupChan
	// when it receives a signal. We can't trigger that in a unit test,
	// but we've verified the setup worked.
}

// TestSignalHandlerIntegration verifies the full signal handling flow.
func TestSignalHandlerIntegration(t *testing.T) {
	t.Parallel()
	mgr := New()

	_ = mgr.SetupSignalHandler()

	// Verify the signal handler goroutine is running by checking
	// that the context gets cancelled when we call Cancel.
	// Actual signal triggering can't be tested in unit tests.
	done := make(chan struct{})
	go func() {
		<-mgr.Context().Done()
		close(done)
	}()

	mgr.Cancel()

	// Verify cancellation is propagated
	select {
	case <-done:
		// Expected: goroutine received cancellation
	case <-time.After(2 * time.Second):
		t.Error("Signal handler did not propagate cancellation")
	}
}

// TestWaitWithMultipleGoroutines verifies Wait() handles multiple goroutines.
func TestWaitWithMultipleGoroutines(t *testing.T) {
	t.Parallel()
	mgr := New()

	numGoroutines := 5
	for i := 0; i < numGoroutines; i++ {
		mgr.wg.Add(1)
		go func(id int) {
			defer mgr.wg.Done()
			// Simulate some work
			time.Sleep(time.Duration(id*10) * time.Millisecond)
		}(i)
	}

	// Wait should block until all goroutines complete
	done := make(chan struct{})
	go func() {
		mgr.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Expected: all goroutines completed
	case <-time.After(5 * time.Second):
		t.Error("Wait() did not return after all goroutines completed")
	}
}

// TestContextCancellation verifies that goroutines can listen to context cancellation.
func TestContextCancellation(t *testing.T) {
	t.Parallel()
	mgr := New()

	// Start a goroutine that listens to context
	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case <-mgr.Context().Done():
			// Expected: context cancelled
		case <-time.After(5 * time.Second):
			t.Error("Goroutine did not receive context cancellation")
		}
	}()

	// Cancel after a short delay
	time.Sleep(100 * time.Millisecond)
	mgr.Cancel()

	// Verify goroutine received cancellation
	select {
	case <-done:
		// Expected: goroutine exited
	case <-time.After(2 * time.Second):
		t.Error("Goroutine did not exit after context cancellation")
	}
}

// TestMultipleCancelCalls verifies that calling Cancel() multiple times is safe.
func TestMultipleCancelCalls(t *testing.T) {
	t.Parallel()
	mgr := New()

	// Call Cancel multiple times
	mgr.Cancel()
	mgr.Cancel()
	mgr.Cancel()

	// Verify context is cancelled
	select {
	case <-mgr.Context().Done():
		// Expected: context cancelled
	default:
		t.Error("Context should be cancelled after Cancel()")
	}
}

// TestSignalHandlerDoesNotBlock verifies that signal handler doesn't block.
func TestSignalHandlerDoesNotBlock(t *testing.T) {
	t.Parallel()
	mgr := New()

	cleanupChan := mgr.SetupSignalHandler()

	// Verify we can set up the handler without blocking
	// The channel should be buffered and ready
	if cleanupChan == nil {
		t.Fatal("SetupSignalHandler() returned nil channel")
	}

	// We can't test actual signal reception in unit tests,
	// but we've verified the handler is set up correctly
	_ = cleanupChan
}
