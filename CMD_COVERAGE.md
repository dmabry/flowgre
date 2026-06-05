# CMD Coverage

## Current State (44.8%)

### Covered (100%)
- All `ParseFlags` methods: flag parsing is deterministic and testable
- `targetFlags` type: String/Set methods for proxy targets

### Uncovered (0%)
- `Execute` methods: Call blocking `Run` functions that start network listeners
- `Run*` wrapper functions: Thin wrappers around ParseFlags + Execute

## Why Coverage Is Low

The `cmd` package is a CLI entry point layer. Its `Execute` methods:
1. Parse flags (tested)
2. Start network listeners (blocking, untestable in unit tests)
3. Wait for signals (blocking, untestable in unit tests)

## Improvement Options

### Option 1: Integration Tests
Add integration tests that start actual network listeners and verify behavior. Tradeoff: slower, more fragile.

### Option 2: Mock Dependencies
Extract interfaces for network operations and mock them. Tradeoff: more indirection, less realistic.

### Option 3: Accept Current Coverage
The cmd layer is thin — most logic is in tested packages (barrage, proxy, etc.). The uncovered code is boilerplate flag parsing + blocking calls.

## Recommendation

Accept current coverage. The cmd layer delegates to well-tested packages. Adding mocks or integration tests would add complexity without proportional benefit.
