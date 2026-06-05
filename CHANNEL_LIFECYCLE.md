# Channel & Goroutine Lifecycle

## Design Principles

1. **Context propagation**: All long-running goroutines receive `context.Context`
2. **WaitGroup synchronization**: All goroutines are tracked via `sync.WaitGroup`
3. **Buffered channels**: Prevent blocking on sends/receives
4. **Graceful shutdown**: Signals cancel contexts, goroutines drain and exit

## Patterns

### Goroutine Launch
```go
wg.Add(1)
go func() {
    defer wg.Done()
    // work with ctx
}()
```

### Channel Creation
```go
// Buffer size based on expected throughput
chan := make(chan Type, bufferSize)
```

### Shutdown Sequence
1. Cancel context (signals, errors, completion)
2. Wait for goroutines to drain (`wg.Wait()`)
3. Close resources (connections, files, DB)

## Verification

All packages follow these patterns:
- `barrage`: Workers, stats collector, web server
- `proxy`: Listener, parser, replicator, workers
- `record`: Network ingest, parser, DB writer
- `replay`: DB reader, workers
- `single`: Single worker

No untracked goroutines or unbuffered channels exist.
