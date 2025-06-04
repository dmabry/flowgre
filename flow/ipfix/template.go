package ipfix

import (
	"encoding/binary"
	"bytes"
	"fmt"
)

// TemplateRecord represents an IPFIX template record with enterprise support
type TemplateRecord struct {
	TemplateID      uint16
	FieldCount      uint16
	ScopeFieldCount uint16 // For scope templates
	Fields          []TemplateField
}

// TemplateField represents a field in an IPFIX template
type TemplateField struct {
	EnterpriseFlag bool
	EnterpriseID   uint32 // IANA enterprise number
	FieldID        uint16 // Per-enterprise field definition
	FieldLength    uint16 // Variable length support
}

// ToBytes converts the template record to its byte representation
func (t *TemplateRecord) ToBytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	
	// Write the fixed header
	if err := binary.Write(buf, binary.BigEndian, t.TemplateID); err != nil {
		return nil, fmt.Errorf("failed to write template ID: %v", err)
	}
	if err := binary.Write(buf, binary.BigEndian, t.FieldCount); err != nil {
		return nil, fmt.Errorf("failed to write field count: %v", err)
	}
	
	// Write scope field count if this is a scope template
	if t.ScopeFieldCount > 0 {
		if err := binary.Write(buf, binary.BigEndian, t.ScopeFieldCount); err != nil {
			return nil, fmt.Errorf("failed to write scope field count: %v", err)
		}
	}
	
	// Write each field
	for _, field := range t.Fields {
		// Handle enterprise-specific fields
		if field.EnterpriseFlag {
			// Set the enterprise bit and write enterprise ID
			enterpriseFieldID := field.FieldID | 0x8000
			if err := binary.Write(buf, binary.BigEndian, enterpriseFieldID); err != nil {
				return nil, fmt.Errorf("failed to write enterprise field ID: %v", err)
			}
			if err := binary.Write(buf, binary.BigEndian, field.EnterpriseID); err != nil {
				return nil, fmt.Errorf("failed to write enterprise ID: %v", err)
			}
		} else {
			// Regular field
			if err := binary.Write(buf, binary.BigEndian, field.FieldID); err != nil {
				return nil, fmt.Errorf("failed to write field ID: %v", err)
			}
		}
		
		// Write field length
		if err := binary.Write(buf, binary.BigEndian, field.FieldLength); err != nil {
			return nil, fmt.Errorf("failed to write field length: %v", err)
		}
	}
	
	return buf.Bytes(), nil
}

// FromBytes populates the template record from byte data
func (t *TemplateRecord) FromBytes(data []byte) error {
	buf := bytes.NewBuffer(data)
	
	// Read the fixed header
	if err := binary.Read(buf, binary.BigEndian, &t.TemplateID); err != nil {
		return fmt.Errorf("failed to read template ID: %v", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &t.FieldCount); err != nil {
		return fmt.Errorf("failed to read field count: %v", err)
	}
	
	// Check if this is a scope template
	t.ScopeFieldCount = 0
	if t.TemplateID == 0 {
		if err := binary.Read(buf, binary.BigEndian, &t.ScopeFieldCount); err != nil {
			return fmt.Errorf("failed to read scope field count: %v", err)
		}
	}
	
	// Read fields
	t.Fields = make([]TemplateField, 0, t.FieldCount)
	for i := 0; i < int(t.FieldCount); i++ {
		var field TemplateField
		
		// Read field ID
		var fieldID uint16
		if err := binary.Read(buf, binary.BigEndian, &fieldID); err != nil {
			return fmt.Errorf("failed to read field ID: %v", err)
		}
		
		// Check for enterprise field
		if fieldID&0x8000 != 0 {
			field.EnterpriseFlag = true
			field.FieldID = fieldID &^ 0x8000
			
			// Read enterprise ID
			if err := binary.Read(buf, binary.BigEndian, &field.EnterpriseID); err != nil {
				return fmt.Errorf("failed to read enterprise ID: %v", err)
			}
		} else {
			field.EnterpriseFlag = false
			field.FieldID = fieldID
		}
		
		// Read field length
		if err := binary.Read(buf, binary.BigEndian, &field.FieldLength); err != nil {
			return fmt.Errorf("failed to read field length: %v", err)
		}
		
		t.Fields = append(t.Fields, field)
	}
	
	return nil
}