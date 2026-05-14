// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

// FlowProfile defines a NetFlow flow type with its template fields and data records.
// Each profile specifies what fields are included in the template and provides
// a factory for creating corresponding data records.
type FlowProfile interface {
	// TemplateFields returns the field definitions for the template.
	// Field order must match the data record struct field order exactly.
	TemplateFields() []Field

	// Name returns a human-readable name for logging and CLI display.
	Name() string
}

// GenericProfile implements FlowProfile for the default 18-field flow.
// This profile maintains backward compatibility with existing GenericFlow records.
type GenericProfile struct{}

// Name returns the profile name.
func (p *GenericProfile) Name() string {
	return "generic"
}

// TemplateFields returns the 18-field template that matches GenericFlow struct layout.
// Field order must match GenericFlow struct field order exactly.
func (p *GenericProfile) TemplateFields() []Field {
	return []Field{
		{Type: IN_BYTES, Length: 4},
		{Type: OUT_BYTES, Length: 4},
		{Type: IN_PKTS, Length: 4},
		{Type: OUT_PKTS, Length: 4},
		{Type: IPV4_SRC_ADDR, Length: 4},
		{Type: IPV4_DST_ADDR, Length: 4},
		{Type: IPV6_SRC_ADDR, Length: 16},
		{Type: IPV6_DST_ADDR, Length: 16},
		{Type: IPV6_SRC_MASK, Length: 1},
		{Type: IPV6_DST_MASK, Length: 1},
		{Type: L4_SRC_PORT, Length: 2},
		{Type: L4_DST_PORT, Length: 2},
		{Type: PROTOCOL, Length: 1},
		{Type: TCP_FLAGS, Length: 1},
		{Type: FIRST_SWITCHED, Length: 4},
		{Type: LAST_SWITCHED, Length: 4},
		{Type: ENGINE_TYPE, Length: 1},
		{Type: ENGINE_ID, Length: 1},
	}
}
