// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import (
	"net"
	"testing"
)

func TestMinimalProfile_Name(t *testing.T) {
	t.Parallel()

	p := &MinimalProfile{}
	if p.Name() != "minimal" {
		t.Errorf("expected name 'minimal', got %q", p.Name())
	}
}

func TestMinimalProfile_TemplateFields(t *testing.T) {
	t.Parallel()

	p := &MinimalProfile{}
	fields := p.TemplateFields()

	if len(fields) != 7 {
		t.Fatalf("expected 7 fields, got %d", len(fields))
	}

	expectedTypes := []uint16{
		IN_BYTES, IN_PKTS,
		IPV4_SRC_ADDR, IPV4_DST_ADDR,
		L4_SRC_PORT, L4_DST_PORT,
		PROTOCOL,
	}
	for i, expected := range expectedTypes {
		if fields[i].Type != expected {
			t.Errorf("field[%d] type mismatch: got %d want %d", i, fields[i].Type, expected)
		}
	}

	expectedLengths := []uint16{4, 4, 4, 4, 2, 2, 1}
	for i, expected := range expectedLengths {
		if fields[i].Length != expected {
			t.Errorf("field[%d] length mismatch: got %d want %d", i, fields[i].Length, expected)
		}
	}
}

func TestMinimalProfile_TemplateFlowSet(t *testing.T) {
	t.Parallel()

	session := NewSession()
	tfs := new(TemplateFlowSet).Generate(session, &MinimalProfile{})

	if len(tfs.Templates) != 1 {
		t.Fatalf("expected 1 template, got %d", len(tfs.Templates))
	}

	template := tfs.Templates[0]
	if template.FieldCount != 7 {
		t.Errorf("expected FieldCount 7, got %d", template.FieldCount)
	}
}

func TestExtendedProfile_Name(t *testing.T) {
	t.Parallel()

	p := &ExtendedProfile{}
	if p.Name() != "extended" {
		t.Errorf("expected name 'extended', got %q", p.Name())
	}
}

func TestExtendedProfile_TemplateFields(t *testing.T) {
	t.Parallel()

	p := &ExtendedProfile{}
	fields := p.TemplateFields()

	if len(fields) != 15 {
		t.Fatalf("expected 15 fields, got %d", len(fields))
	}

	expectedTypes := []uint16{
		IN_BYTES, IN_PKTS,
		IPV4_SRC_ADDR, IPV4_DST_ADDR,
		L4_SRC_PORT, L4_DST_PORT,
		PROTOCOL,
		IN_SRC_MAC, OUT_DST_MAC,
		SRC_VLAN, DST_VLAN,
		MIN_TTL, MAX_TTL,
		FIRST_SWITCHED, LAST_SWITCHED,
	}
	for i, expected := range expectedTypes {
		if fields[i].Type != expected {
			t.Errorf("field[%d] type mismatch: got %d want %d", i, fields[i].Type, expected)
		}
	}

	expectedLengths := []uint16{4, 4, 4, 4, 2, 2, 1, 6, 6, 2, 2, 1, 1, 4, 4}
	for i, expected := range expectedLengths {
		if fields[i].Length != expected {
			t.Errorf("field[%d] length mismatch: got %d want %d", i, fields[i].Length, expected)
		}
	}
}

func TestExtendedProfile_TemplateFlowSet(t *testing.T) {
	t.Parallel()

	session := NewSession()
	tfs := new(TemplateFlowSet).Generate(session, &ExtendedProfile{})

	if len(tfs.Templates) != 1 {
		t.Fatalf("expected 1 template, got %d", len(tfs.Templates))
	}

	template := tfs.Templates[0]
	if template.FieldCount != 15 {
		t.Errorf("expected FieldCount 15, got %d", template.FieldCount)
	}
}

func TestProfileFieldDataAlignment(t *testing.T) {
	t.Parallel()

	profiles := []FlowProfile{
		&GenericProfile{},
		&MinimalProfile{},
		&ExtendedProfile{},
	}

	for _, p := range profiles {
		t.Run(p.Name(), func(t *testing.T) {
			t.Parallel()
			fields := p.TemplateFields()
			if len(fields) == 0 {
				t.Errorf("profile %q has no fields", p.Name())
			}

			// Verify all fields have valid lengths (> 0)
			for i, f := range fields {
				if f.Length == 0 {
					t.Errorf("field[%d] has zero length (type %d)", i, f.Type)
				}
			}

			// Verify no duplicate field types
			seen := make(map[uint16]bool)
			for i, f := range fields {
				if seen[f.Type] {
					t.Errorf("field[%d] has duplicate type %d", i, f.Type)
				}
				seen[f.Type] = true
			}
		})
	}
}

func TestMinimalFlow_Generate(t *testing.T) {
	t.Parallel()

	session := NewSession()
	srcIP := net.ParseIP("10.0.0.1")
	dstIP := net.ParseIP("10.0.0.2")

	mf := new(MinimalFlow).Generate(srcIP, dstIP, httpsPort, session)

	if mf.SrcAddr == 0 {
		t.Error("expected non-zero src addr")
	}
	if mf.DstAddr == 0 {
		t.Error("expected non-zero dst addr")
	}
	if mf.DstPort != uint16(httpsPort) {
		t.Errorf("expected dst port %d, got %d", httpsPort, mf.DstPort)
	}
	if mf.Protocol != tcpProto {
		t.Errorf("expected protocol %d, got %d", tcpProto, mf.Protocol)
	}
}

func TestExtendedFlow_Generate(t *testing.T) {
	t.Parallel()

	session := NewSession()
	srcIP := net.ParseIP("10.0.0.1")
	dstIP := net.ParseIP("10.0.0.2")

	ef := new(ExtendedFlow).Generate(srcIP, dstIP, httpsPort, session)

	if ef.SrcAddr == 0 {
		t.Error("expected non-zero src addr")
	}
	if ef.DstAddr == 0 {
		t.Error("expected non-zero dst addr")
	}
	if ef.DstPort != uint16(httpsPort) {
		t.Errorf("expected dst port %d, got %d", httpsPort, ef.DstPort)
	}
	if ef.Protocol != tcpProto {
		t.Errorf("expected protocol %d, got %d", tcpProto, ef.Protocol)
	}
	if ef.SrcVlan == 0 || ef.SrcVlan > 4094 {
		t.Errorf("expected valid src VLAN, got %d", ef.SrcVlan)
	}
	if ef.DstVlan == 0 || ef.DstVlan > 4094 {
		t.Errorf("expected valid dst VLAN, got %d", ef.DstVlan)
	}
}
