# IPFIX Implementation Plan for Flowgre (Codebase-Aware)

## 1. Codebase Structure Analysis
Existing Flowgre structure:
```
flowgre/
├── flow/                  # Core protocol implementation
│   └── netflow/             # Netflow v9 implementation
├── cmd/                   # CLI commands
├── models/                # Data models
├── utils/                 # Utility functions
```

## 2. Implementation Steps

### 1. Create IPFIX Package
```bash
mkdir -p /workspace/flowgre/flow/ipfix
```

### 2. IPFIX Protocol Structures
**flow/ipfix/ipfix.go**:
```go
package ipfix

import (
    "encoding/binary"
    "bytes"
    "time"
    "fmt"
)

// IPFIX Header (RFC 7011)
type Header struct {
    VersionNumber    uint16 // Must be 10 for IPFIX
    Length           uint16
    ExportTime       uint32
    SequenceNumber   uint32 // Incremented per packet
    ObservationDomainID uint32
}

func (h *Header) ToBytes() ([]byte, error) {
    buf := new(bytes.Buffer)
    if err := binary.Write(buf, binary.BigEndian, h); err != nil {
        return nil, fmt.Errorf("failed to write IPFIX header: %v", err)
    }
    return buf.Bytes(), nil
}

// Template record with enterprise support
type TemplateRecord struct {
    TemplateID      uint16
    FieldCount      uint16
    ScopeFieldCount uint16 // For scope templates
    Fields          []TemplateField
}

type TemplateField struct {
    EnterpriseFlag bool
    EnterpriseID   uint32 // IANA enterprise number
    FieldID        uint16 // Per-enterprise field definition
    FieldLength    uint16 // Variable length support
}
```

### 3. Template Record Implementation
**flow/ipfix/template.go**:
```go
package ipfix

import (
    "encoding/binary"
    "bytes"
    "fmt"
)

type TemplateRecord struct {
    TemplateID      uint16
    FieldCount      uint16
    ScopeFieldCount uint16 // For scope templates
    Fields          []TemplateField
}

type TemplateField struct {
    EnterpriseFlag bool
    EnterpriseID   uint32 // IANA enterprise number
    FieldID        uint16 // Per-enterprise field definition
    FieldLength    uint16 // Variable length support
}

func (t *TemplateRecord) ToBytes() ([]byte, error) {
    buf := new(bytes.Buffer)
    if err := binary.Write(buf, binary.BigEndian, t); err != nil {
        return nil, fmt.Errorf("failed to write template record: %v", err)
    }
    return buf.Bytes(), nil
}

### 4. Data Flow Handling
**flow/ipfix/dataflow.go**:
```go
package ipfix

import (
    "encoding/binary"
    "bytes"
    "fmt"
)

type DataFlowSet struct {
    FlowSetID uint16
    Length    uint16
    Items     []DataItem
    Padding   int
}

type DataItem struct {
    // Fields specific to IPFIX data records
}

func (d *DataFlowSet) ToBytes() ([]byte, error) {
    buf := new(bytes.Buffer)
    if err := binary.Write(buf, binary.BigEndian, d); err != nil {
        return nil, fmt.Errorf("failed to write data flow set: %v", err)
    }
    return buf.Bytes(), nil
}

### 5. Collector Modifications
**cmd/flowgre.go** (or appropriate collector file):
```go
// Add IPFIX configuration flags
ipfixEnabled := flag.Bool("ipfix.enabled", false, "Enable IPFIX support")
ipfixPort := flag.Int("ipfix.port", 4739, "IPFIX listening port")

// In UDP handler initialization
if *ipfixEnabled {
    go startIPFIXCollector(*ipfixPort)
}

func startIPFIXCollector(port int) {
    // Implementation for IPFIX UDP collector
    // Should integrate with flow/ipfix package
}
```

### 6. Configuration Integration
**cmd/flowgre.go** (or appropriate config file):
```go
// Add IPFIX configuration struct
type IPFIXConfig struct {
    Enabled            bool
    Port               int
    TemplateTimeout    time.Duration
    MaxFlowsPerPacket  int
    AllowedEnterprises []uint32
}

// In main configuration setup
ipfixConfig := IPFIXConfig{
    Enabled:           *ipfixEnabled,
    Port:              *ipfixPort,
    TemplateTimeout:   30 * time.Minute,
    AllowedEnterprises: []uint32{0}, // Default to allow IANA
}

### 7. Metrics Enhancements
**cmd/flowgre.go** (or metrics package if exists):
```go
// Add IPFIX metrics
var (
    IPFIXPacketsReceived = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "flowgre_ipfix_packets_received_total",
            Help: "Total number of IPFIX packets received",
        })
    
    IPFIXInvalidEnterprise = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "flowgre_ipfix_invalid_enterprise_id_total",
            Help: "Number of packets with invalid enterprise IDs",
        })
)

// Register IPFIX metrics in main
func init() {
    prometheus.MustRegister(
        IPFIXPacketsReceived,
        IPFIXInvalidEnterprise,
    )
}
```

### 8. Testing Strategy
**flow/ipfix/ipfix_test.go**:
```go
package ipfix

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestIPFIXHeaderSerialization(t *testing.T) {
    // Test header serialization
    header := &Header{
        VersionNumber:     10,
        Length:            16,
        ExportTime:        uint32(time.Now().Unix()),
        SequenceNumber:    1,
        ObservationDomainID: 12345,
    }
    
    data, err := header.ToBytes()
    assert.NoError(t, err)
    assert.Len(t, data, 16)
    
    // Verify version number in serialized data
    assert.Equal(t, uint16(10), binary.BigEndian.Uint16(data[0:2]))
}

func TestTemplateRecordSerialization(t *testing.T) {
    // Test template record with enterprise field
    template := &TemplateRecord{
        TemplateID:      256,
        FieldCount:      1,
        ScopeFieldCount: 0,
        Fields: []TemplateField{
            {
                EnterpriseFlag: true,
                EnterpriseID:   9, // Cisco
                FieldID:        32779,
                FieldLength:    4,
            },
        },
    }
    
    data, err := template.ToBytes()
    assert.NoError(t, err)
    assert.Len(t, data, 8) // 4 bytes for header + 4 bytes for field
    
    // Verify enterprise field serialization
    assert.Equal(t, uint16(0x8009), binary.BigEndian.Uint16(data[4:6])) // Enterprise bit set
}

func TestDataFlowSetSerialization(t *testing.T) {
    // Test data flow set serialization
    flowSet := &DataFlowSet{
        FlowSetID: 256,
        Length:    4,
    }
    
    data, err := flowSet.ToBytes()
    assert.NoError(t, err)
    assert.Len(t, data, 4)
    
    // Verify flow set ID in serialized data
    assert.Equal(t, uint16(256), binary.BigEndian.Uint16(data[0:2]))
}
```

## Implementation Roadmap
1. Create IPFIX package structure under flow/ipfix
2. Implement header parsing and validation
3. Develop template record handling with enterprise support
4. Implement data flow set handling
5. Modify UDP collector to handle IPFIX packets
6. Add IPFIX configuration options to CLI
7. Implement IPFIX-specific metrics
8. Develop comprehensive test suite in flow/ipfix
9. Update documentation with IPFIX examples
10. Perform integration testing with real IPFIX sources

## Review Considerations
- Ensure enterprise ID validation prevents unauthorized sources
- Verify proper handling of variable-length fields
- Check performance of IPFIX packet processing
- Validate metrics completeness for monitoring
- Confirm backward compatibility with Netflow v9
- Review security implications of enterprise field handling
- Ensure proper error handling for malformed packets
- Verify proper memory management for template cache