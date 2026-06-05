# Allocation Patterns

## Hot Paths

### Packet Generation (`netflow`, `ipfix`)
- `GenerateDataNetflow/GenerateDataIPFIX`: Allocates flow structs per call
- `ToBytes()`: Returns `bytes.Buffer` — allocated per packet
- `SendPacket()`: Reuses connection, allocates per-send metadata

### Worker Loop (`barrage`)
- Each worker: 1 session, 1 connection, periodic packet generation
- Stats: Shared collector, per-worker stat structs
- Channels: Buffered to prevent blocking

### Current Tradeoffs
- **Correctness > Performance**: Clear ownership, easy to reason about
- **Allocation frequency**: ~100-400 packets/sec per worker
- **GC pressure**: Moderate — short-lived objects, predictable patterns

## Optimization Opportunities

### Low Hanging Fruit
- Pool `bytes.Buffer` instances for `ToBytes()`
- Pre-allocate flow structs in worker pools
- Reuse stat structs instead of allocating per-cycle

### Higher Effort
- Custom allocator for packet buffers
- Lock-free stats collection
- Zero-copy serialization

## Measurement

See `bench/` suite for baselines:
- `bench/bench_test.go`: Microbenchmarks per-operation
- `bench/sustained_test.go`: Throughput under load
- `bench/memory_test.go`: Heap growth over time

Current baseline (DGX Spark):
- 4 workers: ~40 pkt/s, 88MB heap, zero growth
- 32 workers: ~320 pkt/s, 317MB heap, zero growth
