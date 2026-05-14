// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import (
	"testing"
)

func TestGenericProfile_Name(t *testing.T) {
	t.Parallel()

	p := &GenericProfile{}
	if p.Name() != "generic" {
		t.Errorf("expected name 'generic', got %q", p.Name())
	}
}

func TestGenericProfile_TemplateFields(t *testing.T) {
	t.Parallel()

	p := &GenericProfile{}
	fields := p.TemplateFields()

	// Should have 18 fields
	if len(fields) != 18 {
		t.Errorf("expected 18 fields, got %d", len(fields))
	}

	// Verify field types match the original GenericFlow.GetTemplateFields() order
	expectedTypes := []uint16{
		IN_BYTES, OUT_BYTES, IN_PKTS, OUT_PKTS,
		IPV4_SRC_ADDR, IPV4_DST_ADDR,
		IPV6_SRC_ADDR, IPV6_DST_ADDR, IPV6_SRC_MASK, IPV6_DST_MASK,
		L4_SRC_PORT, L4_DST_PORT,
		PROTOCOL, TCP_FLAGS,
		FIRST_SWITCHED, LAST_SWITCHED,
		ENGINE_TYPE, ENGINE_ID,
	}
	for i, expected := range expectedTypes {
		if fields[i].Type != expected {
			t.Errorf("field[%d] type mismatch: got %d want %d", i, fields[i].Type, expected)
		}
	}

	// Verify field lengths
	expectedLengths := []uint16{4, 4, 4, 4, 4, 4, 16, 16, 1, 1, 2, 2, 1, 1, 4, 4, 1, 1}
	for i, expected := range expectedLengths {
		if fields[i].Length != expected {
			t.Errorf("field[%d] length mismatch: got %d want %d", i, fields[i].Length, expected)
		}
	}
}

func TestTemplateFlowSet_Generate_DefaultProfile(t *testing.T) {
	t.Parallel()

	session := NewSession()
	tfs := new(TemplateFlowSet).Generate(session)

	// Should produce the same result as before (backward compatible)
	if len(tfs.Templates) != 1 {
		t.Fatalf("expected 1 template, got %d", len(tfs.Templates))
	}

	template := tfs.Templates[0]
	if template.TemplateID != 256 {
		t.Errorf("expected TemplateID 256, got %d", template.TemplateID)
	}
	if template.FieldCount != 18 {
		t.Errorf("expected FieldCount 18, got %d", template.FieldCount)
	}
	if len(template.Fields) != 18 {
		t.Errorf("expected 18 fields, got %d", len(template.Fields))
	}
	// Original template length was 80 bytes
	if tfs.Length != 80 {
		t.Errorf("expected Length 80, got %d", tfs.Length)
	}
}

func TestTemplateFlowSet_Generate_WithProfile(t *testing.T) {
	t.Parallel()

	session := NewSession()

	// Generate with explicit GenericProfile
	tfs := new(TemplateFlowSet).Generate(session, &GenericProfile{})

	if len(tfs.Templates) != 1 {
		t.Fatalf("expected 1 template, got %d", len(tfs.Templates))
	}

	template := tfs.Templates[0]
	if template.FieldCount != 18 {
		t.Errorf("expected FieldCount 18, got %d", template.FieldCount)
	}
}

func TestTemplateFlowSet_Generate_ProfileProducesIdenticalOutput(t *testing.T) {
	t.Parallel()

	session := NewSession()

	// Generate without profile (default)
	tfsDefault := new(TemplateFlowSet).Generate(session)
	// Generate with explicit GenericProfile
	tfsProfile := new(TemplateFlowSet).Generate(session, &GenericProfile{})

	// Field counts should match
	if tfsDefault.Templates[0].FieldCount != tfsProfile.Templates[0].FieldCount {
		t.Errorf("field count mismatch: default %d vs profile %d",
			tfsDefault.Templates[0].FieldCount, tfsProfile.Templates[0].FieldCount)
	}

	// Fields should match
	for i := range tfsDefault.Templates[0].Fields {
		df := tfsDefault.Templates[0].Fields[i]
		pf := tfsProfile.Templates[0].Fields[i]
		if df.Type != pf.Type || df.Length != pf.Length {
			t.Errorf("field[%d] mismatch: default {Type:%d,Len:%d} vs profile {Type:%d,Len:%d}",
				i, df.Type, df.Length, pf.Type, pf.Length)
		}
	}
}
