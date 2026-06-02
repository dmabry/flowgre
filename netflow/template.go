// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import (
	"encoding/binary"
	"fmt"
	"time"
)

// Header NetflowHeader v9
type Header struct {
	Version      uint16
	FlowCount    uint16
	SysUptime    uint32
	UnixSec      uint32
	FlowSequence uint32
	SourceID     uint32
}

// Get the size of the Header in bytes
func (h *Header) size() int {
	size := binary.Size(h.Version)
	size += binary.Size(h.FlowCount)
	size += binary.Size(h.SysUptime)
	size += binary.Size(h.UnixSec)
	size += binary.Size(h.FlowSequence)
	size += binary.Size(h.SourceID)
	return size
}

// Get the Header in String
func (h *Header) String() string {
	return fmt.Sprintf("Version: %d Count: %d SysUptime: %d UnixSec: %d FlowSequence: %d SourceID: %d || ",
		h.Version, h.FlowCount, h.SysUptime, h.UnixSec, h.FlowSequence, h.SourceID)
}

// Generate a Header accounting for the given flowCount.  Flowcount should match the expected number of flows in the
// Netflow packet that the Header will be used for.
func (h *Header) Generate(flowSetCount int, sourceID int, session *Session) Header {
	now := time.Now().UnixNano()
	secs := now / int64(time.Second)
	startTime := session.StartTime()
	sysUptime := uint32((now-startTime)/int64(time.Millisecond)) + 1000

	header := new(Header)
	header.Version = 9
	header.SysUptime = sysUptime
	header.UnixSec = uint32(secs)
	header.FlowCount = uint16(flowSetCount)
	header.FlowSequence = session.NextSeq()
	header.SourceID = uint32(sourceID)

	return *header
}

// Field for Template struct
type Field struct {
	Type   uint16
	Length uint16
}

// Get the Field in String
func (f *Field) String() string {
	return fmt.Sprintf("Type: %d Length: %d", f.Type, f.Length)
}

// Template for TemplateFlowSet
type Template struct {
	TemplateID uint16 // 0-255
	FieldCount uint16
	Fields     []Field
}

// Get the size of the Template in bytes
func (t *Template) size() int {
	size := binary.Size(t.TemplateID)
	size += binary.Size(t.FieldCount)
	for _, field := range t.Fields {
		size += binary.Size(field)
	}
	return size
}

// Get the size of the Fields in a given Template in bytes
func (t *Template) sizeOfFields() int {
	var size int
	for _, field := range t.Fields {
		size += int(field.Length)
	}
	return size
}

// TemplateFlowSet for Netflow
type TemplateFlowSet struct {
	FlowSetID uint16 // seems to always be 0???
	Length    uint16
	Templates []Template
	Padding   int
}

// Generate a TemplateFlowSet.
// Per Netflow v9 spec, FlowSetID is *always* 0 for a TemplateFlow.
// Hardcoded TemplateID to 256, but could be variable as long as it is greater than 255.
// If profile is nil, defaults to GenericProfile for backward compatibility.
func (t *TemplateFlowSet) Generate(session *Session, profile ...FlowProfile) TemplateFlowSet {
	p := FlowProfile(&GenericProfile{}) // default
	if len(profile) > 0 && profile[0] != nil {
		p = profile[0]
	}

	templateFlowSet := new(TemplateFlowSet)
	templateFlowSet.FlowSetID = 0
	var templates []Template
	// template
	template := new(Template)
	fields := p.TemplateFields()
	template.TemplateID = 256
	template.FieldCount = uint16(len(fields))
	// add fields to the template
	template.Fields = fields
	templates = append(templates, *template)
	templateFlowSet.Templates = templates
	// Calculate raw size and add 32-bit padding per NetFlow v9 spec
	rawSize := templateFlowSet.rawSize()
	remainder := rawSize % 4
	if remainder > 0 {
		templateFlowSet.Padding = 4 - remainder
	}
	templateFlowSet.Length = uint16(rawSize + templateFlowSet.Padding)
	return *templateFlowSet
}

// rawSize returns the size of the TemplateFlowSet in bytes before padding.
func (t *TemplateFlowSet) rawSize() int {
	size := binary.Size(t.FlowSetID)
	size += binary.Size(t.Length)
	for _, i := range t.Templates {
		size += binary.Size(i.TemplateID)
		size += binary.Size(i.FieldCount)
		for _, f := range i.Fields {
			size += binary.Size(f.Type)
			size += binary.Size(f.Length)
		}
	}
	return size
}

// Get the size of the TemplateFlowSet in bytes
func (t *TemplateFlowSet) size() int {
	size := binary.Size(t.FlowSetID)
	size += binary.Size(t.Length)
	for _, i := range t.Templates {
		size += binary.Size(i.TemplateID)
		size += binary.Size(i.FieldCount)
		for _, f := range i.Fields {
			size += binary.Size(f.Type)
			size += binary.Size(f.Length)
		}
	}
	return size
}
