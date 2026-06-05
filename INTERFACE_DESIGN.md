# Interface Design

## FlowGenerator (`barrage/generator.go`)

Abstracts protocol-specific packet generation so the barrage worker loop is shared between NetFlow and IPFIX.

```go
type FlowGenerator interface {
    Label() string
    GenerateTemplate(sourceID int, session *netflow.Session) []byte
    GenerateOptionsData(sourceID int, session *netflow.Session) []byte
    GenerateData(flowCount int, sourceID int, srcRange, dstRange string, session *netflow.Session) []byte
}
```

**Design rationale:**
- 4 methods are tightly coupled to the packet generation lifecycle
- Splitting would create coordination overhead without benefit
- Single responsibility: "generate packets for a protocol"

## FlowProfile (`netflow/profile_generic.go`)

Defines flow field layouts for different profiles (generic, minimal, extended).

```go
type FlowProfile interface {
    Fields() []FieldDefinition
    Name() string
}
```

**Design rationale:**
- Small interface with clear responsibilities
- Used only by netflow package for template/data generation
- Easy to extend with new profiles

## IPFIXFlowProfile (`ipfix/profile_generic.go`)

Similar to FlowProfile but for IPFIX-specific field definitions.

```go
type IPFIXFlowProfile interface {
    TemplateFields() []Field
    Name() string
}
```

**Design rationale:**
- Mirrors FlowProfile but with IPFIX-specific types
- Could theoretically unify with FlowProfile, but keeps protocols isolated

## Interface Segregation Assessment

All interfaces follow ISP:
- Consumers depend only on methods they need
- No "god interfaces" forcing unused methods
- New profiles can be added without modifying existing code (OCP)
