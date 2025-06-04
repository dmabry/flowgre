package ipfix

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"time"
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

func TestIPFIXHeaderValidation(t *testing.T) {
	// Test valid header
	validHeader := &Header{
		VersionNumber:     10,
		Length:            16,
		ExportTime:        uint32(time.Now().Unix()),
		SequenceNumber:    1,
		ObservationDomainID: 12345,
	}
	assert.True(t, validHeader.IsValid())

	// Test invalid version number
	invalidVersionHeader := &Header{
		VersionNumber:     9,
		Length:            16,
		ExportTime:        uint32(time.Now().Unix()),
		SequenceNumber:    1,
		ObservationDomainID: 12345,
	}
	assert.False(t, invalidVersionHeader.IsValid())

	// Test invalid length
	shortHeader := &Header{
		VersionNumber:     10,
		Length:            15, // Too short
		ExportTime:        uint32(time.Now().Unix()),
		SequenceNumber:    1,
		ObservationDomainID: 12345,
	}
	assert.False(t, shortHeader.IsValid())
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