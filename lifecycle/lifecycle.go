// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package lifecycle provides shared process management for flowgre modes.
// It handles context creation, signal handling (SIGINT/SIGTERM), and WaitGroup coordination.
package lifecycle

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Manager coordinates process lifecycle for a flowgre mode.
type Manager struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup
}

// New creates a new lifecycle manager with a cancellable context and WaitGroup.
func New() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		ctx:    ctx,
		cancel: cancel,
		wg:     &sync.WaitGroup{},
	}
}

// Context returns the managed context for goroutines to listen on.
func (m *Manager) Context() context.Context {
	return m.ctx
}

// Cancel signals all goroutines to stop.
func (m *Manager) Cancel() {
	m.cancel()
}

// WaitGroup returns the shared WaitGroup for tracking goroutine completion.
func (m *Manager) WaitGroup() *sync.WaitGroup {
	return m.wg
}

// SetupSignalHandler registers a one-shot SIGINT/SIGTERM handler that calls
// Cancel. Cancelling the manager also unregisters the signal notification and
// stops the handler goroutine.
func (m *Manager) SetupSignalHandler() <-chan bool {
	sigChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		defer signal.Stop(sigChan)
		defer close(cleanupDone)
		select {
		case <-sigChan:
			log.Printf("\rReceived signal, shutting down...\n\n")
			m.Cancel()
			cleanupDone <- true
		case <-m.ctx.Done():
		}
	}()
	return cleanupDone
}

// Wait blocks until all tracked goroutines complete or context is cancelled.
func (m *Manager) Wait() {
	m.wg.Wait()
}
