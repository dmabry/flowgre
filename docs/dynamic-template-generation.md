# Dynamic NetFlow Template Generation

**Status:** Complete  ✨
**Priority:** High
**Effort:** Half-day
**Author:** Sparky
**Created:** 2026-05-14
**Completed:** 2026-05-14

---

## Problem Statement

NetFlow v9 and IPFIX templates are currently hardcoded to a single flow type. The `GenericFlow` struct defines 18 fields (NetFlow) or 19 fields (IPFIX) that are baked into the template at generation time. This means:

1. **Every flow uses the same template** — there's no way to generate different flow profiles (e.g., a minimal DNS-only flow vs. a full TCP flow with MAC addresses and VLAN tags).
2. **Template generation is coupled to `GenericFlow`** — `TemplateFlowSet.Generate()` calls `new(GenericFlow).GetTemplateFields()` directly, so the template is always the same 18-field layout.
3. **Custom collector testing is limited** — some collectors expect specific field combinations, and we can't test edge cases like unusual field orders, variable-length fields, or exotic field types (MPLS labels, interface names, etc.).

The TODO in `netflow/template.go:106` captures this:
```go
// TODO: Hardcoded FieldCount and Fields for HTTPS Flow. Need to work on Generating different flows
```

---

## Current Architecture

### Template Generation Flow

```
TemplateFlowSet.Generate(session)
    └─> new(GenericFlow).GetTemplateFields()  // Always returns 18 fixed fields
         └─> Template{TemplateID: 256, Fields: [...]18Field}
              └─> TemplateFlowSet{Templates: [template]}
```

### Data Generation Flow

```
DataFlowSet.Generate(flowCount, srcRange, dstRange, flowSrcPort, session)
    └─> for each flow:
         └─> new(GenericFlow).Generate(srcIP, dstIP, flowPort, session)
              └─> GenericFlow struct with 18 fields populated
```

### Key Files

| File | Role | LOC |
|------|------|-----|
| `netflow/template.go` | Template struct + `TemplateFlowSet.Generate()` | 158 |
| `netflow/flow.go` | `GenericFlow` struct, `GetTemplateFields()`, `Generate()` | 271 |
| `netflow/dataflowset.go` | `DataFlowSet.Generate()` — creates flow records | 74 |
| `netflow/packet.go` | `Netflow.ToBytes()` — serializes to wire format | 115 |
| `barrage/generator.go` | `FlowGenerator` interface, `netflowGenerator`, `ipfixGenerator` | 74 |
| `ipfix/ipfix.go` | IPFIX equivalents (same pattern, IANA field types) | 733 |

### Field Type Constants

`netflow/flow.go` already defines **90+ NetFlow v9 field type constants** (lines 44-138) — `IN_BYTES`, `SRC_TOS`, `MPLS_TOP_LABEL_TYPE`, `IF_NAME`, etc. — but only 18 are used in `GetTemplateFields()`. The constants are there; they're just not wired up for dynamic use.

---

## Proposed Solution

### Design Goals

1. **Backward compatible** — existing code that uses `GenericFlow` should work unchanged.
2. **Extensible** — new flow profiles can be added without modifying existing code.
3. **Type-safe** — field definitions and data records should stay in sync.
4. **Testable** — each flow profile should be independently testable.

### Architecture

Introduce a `FlowProfile` interface that abstracts the relationship between template fields and data records:

```go
// FlowProfile defines a NetFlow/IPFIX flow type.
type FlowProfile interface {
    // TemplateFields returns the field definitions for the template.
    // Field order must match the data record struct field order exactly.
    TemplateFields() []Field

    // NewDataRecord creates a new data record to be populated by Generate().
    NewDataRecord() any

    // Name returns a human-readable name for logging.
    Name() string
}
```

### Implementation Plan

#### Phase 1: Extract `GenericFlow` into a Profile (1 hour)

**File: `netflow/profile_generic.go`** (new)

```go
package netflow

// GenericProfile implements FlowProfile for the default 18-field flow.
type GenericProfile struct{}

func (p *GenericProfile) Name() string {
    return "generic"
}

func (p *GenericProfile) TemplateFields() []Field {
    // Move GetTemplateFields() logic here
    // ...
}

func (p *GenericProfile) NewDataRecord() any {
    return &GenericFlow{}
}
```

**File: `netflow/flow.go`** (modify)

- Remove `GetTemplateFields()` from `GenericFlow` (move to `GenericProfile`)
- Keep `GenericFlow.Generate()` as-is (it populates the struct)

**File: `netflow/template.go`** (modify)

- `TemplateFlowSet.Generate()` accepts an optional `FlowProfile` parameter
- Falls back to `GenericProfile` if none provided (backward compatible)

```go
func (t *TemplateFlowSet) Generate(session *Session, profile ...FlowProfile) TemplateFlowSet {
    p := &GenericProfile{} // default
    if len(profile) > 0 {
        p = profile[0]
    }
    template := Template{
        TemplateID: 256,
        FieldCount: uint16(len(p.TemplateFields())),
        Fields:     p.TemplateFields(),
    }
    // ...
}
```

**Tests:**
- `TestGenericProfile_TemplateFields` — verify 18 fields match current output
- `TestTemplateFlowSet_Generate_Default` — verify default profile produces identical bytes to current code
- `TestTemplateFlowSet_Generate_Custom` — verify custom profile produces different template

#### Phase 2: Build-in Flow Profiles (1 hour)

Create pre-defined profiles for common testing scenarios:

**File: `netflow/profile_minimal.go`** (new)

```go
// MinimalProfile generates a minimal flow with only essential fields:
// src IP, dst IP, src port, dst port, protocol, bytes, packets.
type MinimalProfile struct{}

func (p *MinimalProfile) Name() string { return "minimal" }
func (p *MinimalProfile) TemplateFields() []Field {
    return []Field{
        {Type: IN_BYTES, Length: 4},
        {Type: IN_PKTS, Length: 4},
        {Type: IPV4_SRC_ADDR, Length: 4},
        {Type: IPV4_DST_ADDR, Length: 4},
        {Type: L4_SRC_PORT, Length: 2},
        {Type: L4_DST_PORT, Length: 2},
        {Type: PROTOCOL, Length: 1},
    }
}
func (p *MinimalProfile) NewDataRecord() any {
    return &MinimalFlow{} // new struct matching the 7 fields
}
```

**File: `netflow/profile_extended.go`** (new)

```go
// ExtendedProfile generates a flow with MAC addresses, VLANs, TTL, and interface info.
type ExtendedProfile struct{}

func (p *ExtendedProfile) Name() string { return "extended" }
func (p *ExtendedProfile) TemplateFields() []Field {
    return []Field{
        {Type: IN_BYTES, Length: 4},
        {Type: IN_PKTS, Length: 4},
        {Type: IPV4_SRC_ADDR, Length: 4},
        {Type: IPV4_DST_ADDR, Length: 4},
        {Type: L4_SRC_PORT, Length: 2},
        {Type: L4_DST_PORT, Length: 2},
        {Type: PROTOCOL, Length: 1},
        {Type: IN_SRC_MAC, Length: 6},
        {Type: OUT_DST_MAC, Length: 6},
        {Type: SRC_VLAN, Length: 2},
        {Type: DST_VLAN, Length: 2},
        {Type: MIN_TTL, Length: 1},
        {Type: MAX_TTL, Length: 1},
        {Type: FIRST_SWITCHED, Length: 4},
        {Type: LAST_SWITCHED, Length: 4},
    }
}
func (p *ExtendedProfile) NewDataRecord() any {
    return &ExtendedFlow{} // new struct matching the 14 fields
}
```

**Tests:**
- `TestMinimalProfile_TemplateFields` — verify 7 fields
- `TestExtendedProfile_TemplateFields` — verify 14 fields
- `TestProfile_DataRecord_Matches_Template` — generic test that verifies `NewDataRecord()` struct fields align with `TemplateFields()` count

#### Phase 3: Wire into `FlowGenerator` (1 hour)

**File: `barrage/generator.go`** (modify)

Add profile parameter to `FlowGenerator`:

```go
type FlowGenerator interface {
    Label() string
    GenerateTemplate(sourceID int, session *Session) []byte
    GenerateOptionsData(sourceID int, session *Session) []byte
    GenerateData(flowCount int, sourceID int, srcRange string, dstRange string, session *Session) []byte
}

// WithProfile is a functional option for FlowGenerator.
type WithProfile func() netflow.FlowProfile

func (g netflowGenerator) GenerateTemplate(sourceID int, session *Session, opts ...WithProfile) []byte {
    profile := &netflow.GenericProfile{}
    for _, opt := range opts {
        if p := opt(); p != nil {
            profile = p
        }
    }
    // Use profile to generate template
    tfs := new(netflow.TemplateFlowSet).Generate(session, profile)
    // ...
}
```

**File: `cmd/barrage.go`** (modify)

Add `-profile` flag:

```go
c.profile = fs.String("profile", "generic", "Flow profile: generic, minimal, extended")
```

Map string to profile:

```go
var profile netflow.FlowProfile
switch *c.profile {
case "minimal":
    profile = &netflow.MinimalProfile{}
case "extended":
    profile = &netflow.ExtendedProfile{}
default:
    profile = &netflow.GenericProfile{}
}
```

**Tests:**
- `TestBarrage_WithMinimalProfile` — verify minimal profile produces smaller packets
- `TestBarrage_WithExtendedProfile` — verify extended profile produces larger packets
- `TestBarrage_ProfileRoundTrip` — verify template + data flow round-trips correctly

#### Phase 4: IPFIX Parity (30 minutes)

Mirror the profile system in `ipfix/` package:

- `ipfix/profile_generic.go` — move existing `GenericFlow.GetTemplateFields()` logic
- `ipfix/profile_minimal.go` — minimal IPFIX profile (IANA field types)
- `ipfix/profile_extended.go` — extended IPFIX profile

The IPFIX package already has its own `GenericFlow` struct with IANA field types, so the pattern is identical but with different field type constants.

#### Phase 5: Tests and Documentation (30 minutes)

- Update `netflow/netflow_test.go` with profile-based tests
- Add `TestTemplateFieldDataAlignment` — generic test that verifies any profile's template fields match its data record struct
- Update README with new `-profile` flag
- Update AGENTS.md with the new architecture

---

## File Changes Summary

| File | Action | Description |
|------|--------|-------------|
| `netflow/profile_generic.go` | **New** | `GenericProfile` — default 18-field profile |
| `netflow/profile_minimal.go` | **New** | `MinimalProfile` — 7-field minimal profile |
| `netflow/profile_extended.go` | **New** | `ExtendedProfile` — 14-field extended profile |
| `netflow/flow.go` | **Modify** | Remove `GetTemplateFields()`, keep `Generate()` |
| `netflow/template.go` | **Modify** | Accept optional `FlowProfile` in `Generate()` |
| `netflow/dataflowset.go` | **Modify** | Accept optional `FlowProfile` in `Generate()` |
| `barrage/generator.go` | **Modify** | Add profile support to `FlowGenerator` |
| `cmd/barrage.go` | **Modify** | Add `-profile` flag |
| `ipfix/profile_generic.go` | **New** | IPFIX `GenericProfile` |
| `ipfix/profile_minimal.go` | **New** | IPFIX `MinimalProfile` |
| `ipfix/profile_extended.go` | **New** | IPFIX `ExtendedProfile` |
| `ipfix/ipfix.go` | **Modify** | Accept optional profile in template/data generation |
| `README.md` | **Modify** | Document `-profile` flag |
| `AGENTS.md` | **Modify** | Document new profile architecture |

---

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Template/data field mismatch | Silent corruption — collector rejects packets | `TestTemplateFieldDataAlignment` verifies alignment at test time |
| Backward compatibility break | Existing users see different packets | Default profile is `GenericProfile` which produces identical output to current code |
| IPFIX parity lag | IPFIX lags behind NetFlow profiles | Mirror profiles in same PR; automated tests catch drift |

---

## Acceptance Criteria

- [ ] Default profile produces byte-identical output to current code
- [ ] Minimal profile generates valid NetFlow v9 packets with 7 fields
- [ ] Extended profile generates valid NetFlow v9 packets with 14+ fields
- [ ] IPFIX profiles mirror NetFlow profiles with IANA field types
- [ ] `-profile` flag works in barrage mode
- [ ] All tests pass with `-race` detector
- [ ] `netflow` coverage remains ≥ 58%
- [ ] README documents the new `-profile` flag

---

## Future Enhancements (Out of Scope)

- **Custom profile via YAML** — users define their own field lists in config files
- **Per-worker profiles** — different workers use different profiles for mixed testing
- **Variable-length fields** — support `IF_NAME`, `APPLICATION_DESCRIPTION`, etc.
- **Template ID rotation** — test collector behavior with changing template IDs
- **Multi-template packets** — multiple templates in a single NetFlow packet
