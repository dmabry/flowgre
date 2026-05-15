// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestDataFlowSet_Generate_MinimalProfile(t *testing.T) {
	t.Parallel()

	session := NewSession()
	dfs := new(DataFlowSet).Generate(5, "10.0.0.0/8", "10.0.0.0/8", httpsPort, session, &MinimalProfile{})

	if len(dfs.Items) != 5 {
		t.Fatalf("expected 5 items, got %d", len(dfs.Items))
	}

	for i, item := range dfs.Items {
		mf, ok := item.(MinimalFlow)
		if !ok {
			t.Errorf("item[%d]: expected MinimalFlow, got %T", i, item)
			continue
		}
		if mf.SrcAddr == 0 {
			t.Errorf("item[%d]: expected non-zero src addr", i)
		}
		if mf.DstPort != uint16(httpsPort) {
			t.Errorf("item[%d]: expected dst port %d, got %d", i, httpsPort, mf.DstPort)
		}
	}
}

func TestDataFlowSet_Generate_ExtendedProfile(t *testing.T) {
	t.Parallel()

	session := NewSession()
	dfs := new(DataFlowSet).Generate(5, "10.0.0.0/8", "10.0.0.0/8", httpsPort, session, &ExtendedProfile{})

	if len(dfs.Items) != 5 {
		t.Fatalf("expected 5 items, got %d", len(dfs.Items))
	}

	for i, item := range dfs.Items {
		ef, ok := item.(ExtendedFlow)
		if !ok {
			t.Errorf("item[%d]: expected ExtendedFlow, got %T", i, item)
			continue
		}
		if ef.SrcAddr == 0 {
			t.Errorf("item[%d]: expected non-zero src addr", i)
		}
		if ef.DstPort != uint16(httpsPort) {
			t.Errorf("item[%d]: expected dst port %d, got %d", i, httpsPort, ef.DstPort)
		}
		if ef.SrcVlan == 0 {
			t.Errorf("item[%d]: expected non-zero src VLAN", i)
		}
	}
}

func TestDataFlowSet_Generate_DefaultProfile(t *testing.T) {
	t.Parallel()

	session := NewSession()
	dfs := new(DataFlowSet).Generate(5, "10.0.0.0/8", "10.0.0.0/8", httpsPort, session)

	if len(dfs.Items) != 5 {
		t.Fatalf("expected 5 items, got %d", len(dfs.Items))
	}

	for i, item := range dfs.Items {
		gf, ok := item.(GenericFlow)
		if !ok {
			t.Errorf("item[%d]: expected GenericFlow, got %T", i, item)
			continue
		}
		if gf.Ipv4SrcAddr == 0 {
			t.Errorf("item[%d]: expected non-zero IPv4 src addr", i)
		}
	}
}

func TestMinimalProfile_RoundTrip(t *testing.T) {
	t.Parallel()

	session := NewSession()
	sourceID := 618
	flowCount := 5

	// Generate template + data with minimal profile
	tFlow := GenerateTemplateNetflow(sourceID, session, &MinimalProfile{})
	dFlow := GenerateDataNetflow(flowCount, sourceID, "10.0.0.0/8", "10.0.0.0/8", httpsPort, session, &MinimalProfile{})

	// Serialize
	tBuf := tFlow.ToBytes()
	dBuf := dFlow.ToBytes()

	// Verify template
	tRead := make([]byte, tBuf.Len())
	tBuf.Read(tRead)

	tReader := bytes.NewReader(tRead)
	var tHeader Header
	binary.Read(tReader, binary.BigEndian, &tHeader)

	if tHeader.Version != 9 {
		t.Fatalf("expected version 9, got %d", tHeader.Version)
	}

	// Parse template fields
	var fsID, fsLen uint16
	binary.Read(tReader, binary.BigEndian, &fsID)
	binary.Read(tReader, binary.BigEndian, &fsLen)

	var templateID, fieldCount uint16
	binary.Read(tReader, binary.BigEndian, &templateID)
	binary.Read(tReader, binary.BigEndian, &fieldCount)

	if fieldCount != 7 {
		t.Fatalf("expected 7 fields in minimal template, got %d", fieldCount)
	}

	// Verify data
	dRead := make([]byte, dBuf.Len())
	dBuf.Read(dRead)

	dReader := bytes.NewReader(dRead)
	var dHeader Header
	binary.Read(dReader, binary.BigEndian, &dHeader)

	if dHeader.Version != 9 {
		t.Fatalf("expected version 9, got %d", dHeader.Version)
	}

	// Parse data flow set header
	var dFsID, dFsLen uint16
	binary.Read(dReader, binary.BigEndian, &dFsID)
	binary.Read(dReader, binary.BigEndian, &dFsLen)

	if dFsID != 256 {
		t.Fatalf("expected FlowSetID 256, got %d", dFsID)
	}

	// Read each MinimalFlow record
	for i := range flowCount {
		var mf MinimalFlow
		err := binary.Read(dReader, binary.BigEndian, &mf)
		if err != nil {
			t.Fatalf("failed to read MinimalFlow[%d]: %v", i, err)
		}
		if mf.SrcAddr == 0 {
			t.Errorf("MinimalFlow[%d]: expected non-zero src addr after round trip", i)
		}
	}
}

func TestExtendedProfile_RoundTrip(t *testing.T) {
	t.Parallel()

	session := NewSession()
	sourceID := 618
	flowCount := 5

	// Generate template + data with extended profile
	tFlow := GenerateTemplateNetflow(sourceID, session, &ExtendedProfile{})
	dFlow := GenerateDataNetflow(flowCount, sourceID, "10.0.0.0/8", "10.0.0.0/8", httpsPort, session, &ExtendedProfile{})

	// Serialize
	tBuf := tFlow.ToBytes()
	dBuf := dFlow.ToBytes()

	// Verify template
	tRead := make([]byte, tBuf.Len())
	tBuf.Read(tRead)

	tReader := bytes.NewReader(tRead)
	var tHeader Header
	binary.Read(tReader, binary.BigEndian, &tHeader)

	// Parse template fields
	var fsID, fsLen uint16
	binary.Read(tReader, binary.BigEndian, &fsID)
	binary.Read(tReader, binary.BigEndian, &fsLen)

	var templateID, fieldCount uint16
	binary.Read(tReader, binary.BigEndian, &templateID)
	binary.Read(tReader, binary.BigEndian, &fieldCount)

	if fieldCount != 15 {
		t.Fatalf("expected 15 fields in extended template, got %d", fieldCount)
	}

	// Verify data
	dRead := make([]byte, dBuf.Len())
	dBuf.Read(dRead)

	dReader := bytes.NewReader(dRead)
	var dHeader Header
	binary.Read(dReader, binary.BigEndian, &dHeader)

	// Parse data flow set header
	var dFsID, dFsLen uint16
	binary.Read(dReader, binary.BigEndian, &dFsID)
	binary.Read(dReader, binary.BigEndian, &dFsLen)

	// Read each ExtendedFlow record
	for i := range flowCount {
		var ef ExtendedFlow
		err := binary.Read(dReader, binary.BigEndian, &ef)
		if err != nil {
			t.Fatalf("failed to read ExtendedFlow[%d]: %v", i, err)
		}
		if ef.SrcAddr == 0 {
			t.Errorf("ExtendedFlow[%d]: expected non-zero src addr after round trip", i)
		}
		if ef.SrcVlan == 0 {
			t.Errorf("ExtendedFlow[%d]: expected non-zero src VLAN after round trip", i)
		}
	}
}

func TestGenerateNetflow_WithProfile(t *testing.T) {
	profiles := []struct {
		name     string
		profile  FlowProfile
		fieldCnt uint16
	}{
		{"generic", &GenericProfile{}, 18},
		{"minimal", &MinimalProfile{}, 7},
		{"extended", &ExtendedProfile{}, 15},
	}

	for _, tc := range profiles {
		t.Run(tc.name, func(t *testing.T) {
			session := NewSession()

			nf := GenerateNetflow(5, 1, "10.0.0.0/8", "10.0.0.0/8", session, tc.profile)

			if len(nf.TemplateFlowSets) != 1 {
				t.Fatal("expected 1 template flow set")
			}

			tmpl := nf.TemplateFlowSets[0].Templates[0]
			if tmpl.FieldCount != tc.fieldCnt {
				t.Errorf("expected FieldCount %d, got %d", tc.fieldCnt, tmpl.FieldCount)
			}

			if len(nf.DataFlowSets) != 1 {
				t.Fatal("expected 1 data flow set")
			}

			if len(nf.DataFlowSets[0].Items) != 5 {
				t.Errorf("expected 5 items, got %d", len(nf.DataFlowSets[0].Items))
			}
		})
	}
}
