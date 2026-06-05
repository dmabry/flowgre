# Context Propagation

## Design

Flowgre uses a two-layer context model:

1. **CLI Entry Points** (`cmd/*.go`): Create contexts via `lifecycle.Manager` with signal handling
2. **Core Packages** (`barrage`, `proxy`, `record`, etc.): Accept `context.Context` via `*Ctx` functions

## Pattern

```go
// CLI layer creates context
func (c *Command) Execute() error {
    mgr := lifecycle.New()
    cleanupDone := mgr.SetupSignalHandler()
    
    go func() {
        <-cleanupDone
        mgr.Cancel()
    }()
    
    // Pass context to core package
    core.RunCtx(mgr.Context(), ...)
}

// Core package accepts context
func RunCtx(ctx context.Context, ...) {
    // All goroutines receive ctx
    go worker(ctx, ...)
}
```

## Functions Without Context

These intentionally don't accept context:
- `cmd/* Execute()`: CLI entry points that create their own contexts
- `utils/*`: Pure utility functions with no blocking I/O
- `models/*`: Data structures with no behavior

## Verification

All blocking operations accept context:
- `barrage.RunCtx/StartCtx`
- `proxy.worker/replicator/parseNetflow/proxyListener/statsPrinter`
- `record.RunCtx/netIngest/dbIngest`
- `replay.RunCtx/dbReader/worker`
- `single.RunCtx`
- `stats.Collector.Run`
- `web.RunWebServer`
