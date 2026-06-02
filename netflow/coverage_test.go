// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/dmabry/flowgre/utils"
)

// ---------------------------------------------------------------------------
// Version constant
// ---------------------------------------------------------------------------

func TestVersion_Constant(t *testing.T) {
	t.Parallel()
	h := new(Header).Generate(1, 1, NewSession())
	if h.Version != 9 {
		t.Errorf("Header version should be 9 for NetFlow, got %d", h.Version)
	}
}

// ---------------------------------------------------------------------------
// Header edge cases
// ---------------------------------------------------------------------------

func TestHeader_Size(t *testing.T) {
	t.Parallel()
	h := new(Header).Generate(1, 618, NewSession())
	if h.size() != 20 {
		t.Errorf("Header size should be 20 bytes, got %d", h.size())
	}
}

func TestHeader_String_Output(t *testing.T) {
	t.Parallel()
	h := new(Header).Generate(5, 618, NewSession())
	s := h.String()

	if !strings.Contains(s, "Version: 9") {
		t.Error("Header String should contain version")
	}
	if !strings.Contains(s, "Count: 5") {
		t.Error("Header String should contain flow count")
	}
	if !strings.Contains(s, "SourceID: 618") {
		t.Error("Header String should contain source ID")
	}
}

// ---------------------------------------------------------------------------
// TemplateFlowSet edge cases
// ---------------------------------------------------------------------------

func TestTemplateFlowSet_Size(t *testing.T) {
	t.Parallel()
	session := NewSession()
	tfs := new(TemplateFlowSet).Generate(session)

	size := tfs.size()
	if size <= 0 {
		t.Errorf("TemplateFlowSet size should be positive, got %d", size)
	}

	// Size should match: FlowSetID(2) + Length(2) + TemplateID(2) + FieldCount(2) + fields
	expected := 4 + 4 + len(tfs.Templates[0].Fields)*4 // header + template header + fields
	if size != expected {
		t.Errorf("TemplateFlowSet size mismatch: got %d, expected %d", size, expected)
	}
}

func TestTemplateFlowSet_Padding_Alignment(t *testing.T) {
	t.Parallel()
	session := NewSession()

	cases := []struct {
		name    string
		profile FlowProfile
	}{
		{"minimal", &MinimalProfile{}},
		{"extended", &ExtendedProfile{}},
		{"generic", &GenericProfile{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tfs := new(TemplateFlowSet).Generate(session, tc.profile)
			// Total length should be aligned to 4-byte boundary
			if tfs.Length%4 != 0 {
				t.Errorf("%s: TemplateFlowSet length %d should be 4-byte aligned",
					tc.name, tfs.Length)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DataFlowSet edge cases
// ---------------------------------------------------------------------------

func TestDataFlowSet_Size(t *testing.T) {
	t.Parallel()
	session := NewSession()
	dfs := new(DataFlowSet).Generate(10, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, session)

	size := dfs.size()
	if size <= 0 {
		t.Errorf("DataFlowSet size should be positive, got %d", size)
	}
}

func TestDataFlowSet_Generate_ZeroFlowCount(t *testing.T) {
	t.Parallel()
	session := NewSession()
	dfs := new(DataFlowSet).Generate(0, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, session)

	if len(dfs.Items) != 0 {
		t.Errorf("expected 0 items with zero flow count, got %d", len(dfs.Items))
	}

	// Should still produce a valid (empty) flowset
	if dfs.FlowSetID < 256 {
		t.Errorf("FlowSetID should be >= 256, got %d", dfs.FlowSetID)
	}
}

func TestDataFlowSet_Padding_Alignment(t *testing.T) {
	t.Parallel()
	session := NewSession()

	cases := []struct {
		name    string
		profile FlowProfile
	}{
		{"minimal", &MinimalProfile{}},
		{"extended", &ExtendedProfile{}},
		{"generic", &GenericProfile{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dfs := new(DataFlowSet).Generate(1, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, session, tc.profile)
			// Total length should be aligned to 4-byte boundary
			if dfs.Length%4 != 0 {
				t.Errorf("%s: DataFlowSet length %d should be 4-byte aligned",
					tc.name, dfs.Length)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Packet size reporting
// ---------------------------------------------------------------------------

func TestGetNetFlowSizes(t *testing.T) {
	t.Parallel()
	sourceID := 618
	flowcount := 10
	session := NewSession()
	nf := GenerateNetflow(flowcount, sourceID, "10.0.0.0/8", "10.0.0.0/8", session)

	output := GetNetFlowSizes(nf)

	if !strings.Contains(output, "Header Size:") {
		t.Error("sizes output should contain header size")
	}
	if !strings.Contains(output, "Template Size:") {
		t.Error("sizes output should contain template size")
	}
	if !strings.Contains(output, "Data Size:") {
		t.Error("sizes output should contain data size")
	}
	if !strings.Contains(output, "bytes") {
		t.Error("sizes output should mention bytes")
	}
}

// ---------------------------------------------------------------------------
// IsValidNetFlow edge cases
// ---------------------------------------------------------------------------

func TestIsValidNetFlow_Valid(t *testing.T) {
	t.Parallel()
	session := NewSession()
	nf := GenerateTemplateNetflow(618, session)
	buf := nf.ToBytes()

	valid, err := IsValidNetFlow(buf.Bytes(), 9)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("valid NetFlow v9 payload should pass validation")
	}
}

func TestIsValidNetFlow_TooShort(t *testing.T) {
	t.Parallel()
	// Payload shorter than a 20-byte header
	payload := []byte{0, 9, 1} // only 3 bytes

	valid, err := IsValidNetFlow(payload, 9)
	if err == nil {
		t.Error("expected error for too-short payload")
	}
	if valid {
		t.Error("too-short payload should not be valid")
	}
}

func TestIsValidNetFlow_Empty(t *testing.T) {
	t.Parallel()
	valid, err := IsValidNetFlow(nil, 9)
	if err == nil {
		t.Error("expected error for nil payload")
	}
	if valid {
		t.Error("nil payload should not be valid")
	}
}

func TestIsValidNetFlow_RejectOtherVersions(t *testing.T) {
	t.Parallel()
	session := NewSession()
	nf := GenerateTemplateNetflow(618, session)
	buf := nf.ToBytes()

	// Valid v9 payload should fail when checking for v10
	valid, err := IsValidNetFlow(buf.Bytes(), 10)
	if err == nil {
		t.Error("expected version mismatch error")
	}
	if valid {
		t.Error("v9 payload should not validate as v10")
	}

	// Also reject v5
	valid, err = IsValidNetFlow(buf.Bytes(), 5)
	if err == nil {
		t.Error("expected version mismatch error for v5")
	}
	if valid {
		t.Error("v9 payload should not validate as v5")
	}
}

// ---------------------------------------------------------------------------
// UpdateTimeStamp edge cases
// ---------------------------------------------------------------------------

func TestUpdateTimeStamp_Valid(t *testing.T) {
	t.Parallel()
	session := NewSession()
	nf := GenerateTemplateNetflow(618, session)
	buf := nf.ToBytes()
	original := buf.Bytes()

	updated, err := UpdateTimeStamp(original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(updated) != len(original) {
		t.Errorf("updated payload length %d should match original %d", len(updated), len(original))
	}
}

func TestUpdateTimeStamp_TooShort(t *testing.T) {
	t.Parallel()
	// Less than 20 bytes (header size)
	payload := []byte{0, 9, 1}

	updated, err := UpdateTimeStamp(payload)
	if err == nil {
		t.Error("expected error for too-short payload")
	}
	if updated != nil {
		t.Error("should return nil on error")
	}
}

func TestUpdateTimeStamp_Empty(t *testing.T) {
	t.Parallel()
	updated, err := UpdateTimeStamp(nil)
	if err == nil {
		t.Error("expected error for nil payload")
	}
	if updated != nil {
		t.Error("should return nil on error")
	}
}

func TestUpdateTimeStamp_PreservesPayload(t *testing.T) {
	t.Parallel()
	session := NewSession()
	nf := GenerateTemplateNetflow(618, session)
	buf := nf.ToBytes()
	original := buf.Bytes()

	updated, err := UpdateTimeStamp(original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Compare everything except the timestamp field (bytes 8-11 in the header)
	// Header: Version(2) + FlowCount(2) + SysUptime(4) + UnixSec(4) + FlowSequence(4) + SourceID(4)
	// UnixSec is at offset 8, length 4
	for i := range len(original) {
		if i >= 8 && i < 12 {
			continue // skip timestamp bytes
		}
		if original[i] != updated[i] {
			t.Errorf("byte[%d] differs: original=0x%02x updated=0x%02x (payload corrupted outside timestamp)",
				i, original[i], updated[i])
		}
	}

	// Timestamp should have changed
	var origHeader, updHeader Header
	binary.Read(bytes.NewReader(original), binary.BigEndian, &origHeader)
	binary.Read(bytes.NewReader(updated), binary.BigEndian, &updHeader)
	if origHeader.UnixSec == updHeader.UnixSec {
		// Allow equality if both ran in the same second, but highly unlikely in tests
		t.Log("timestamps identical (ran in same second)")
	}
}

// ---------------------------------------------------------------------------
// ToBytes buffer integrity
// ---------------------------------------------------------------------------

func TestToBytes_BufferLengthMatchesFlowSetLengths(t *testing.T) {
	t.Parallel()
	sourceID := 618
	flowcount := 10
	session := NewSession()
	nf := GenerateNetflow(flowcount, sourceID, "10.0.0.0/8", "10.0.0.0/8", session)

	buf := nf.ToBytes()

	// Expected: header(20) + template flowset length + data flowset length
	expectedLen := nf.Header.size()
	for _, tf := range nf.TemplateFlowSets {
		expectedLen += int(tf.Length)
	}
	for _, df := range nf.DataFlowSets {
		expectedLen += int(df.Length)
	}

	if buf.Len() != expectedLen {
		t.Errorf("buffer length %d should match calculated total %d", buf.Len(), expectedLen)
	}
}

func TestToBytes_DataOnly_BufferLengthMatchesFlowSetLengths(t *testing.T) {
	t.Parallel()
	sourceID := 618
	flowcount := 10
	session := NewSession()
	nf := GenerateDataNetflow(flowcount, sourceID, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, session)

	buf := nf.ToBytes()

	expectedLen := nf.Header.size()
	for _, df := range nf.DataFlowSets {
		expectedLen += int(df.Length)
	}

	if buf.Len() != expectedLen {
		t.Errorf("data-only buffer length %d should match calculated total %d", buf.Len(), expectedLen)
	}
}

// ---------------------------------------------------------------------------
// GenericFlow edge cases
// ---------------------------------------------------------------------------

func TestGenericFlow_Generate_AllProtocols(t *testing.T) {
	srcIP := net.ParseIP("10.0.0.1")
	dstIP := net.ParseIP("10.0.0.2")

	cases := []struct {
		port         int
		wantPort     uint16
		wantProtocol uint8
	}{
		{utils.SSHPort, uint16(utils.SSHPort), utils.TCPProto},
		{utils.FTPPort, uint16(utils.FTPPort), utils.TCPProto},
		{utils.DNSPort, uint16(utils.DNSPort), utils.UDPProto},
		{utils.HTTPPort, uint16(utils.HTTPPort), utils.TCPProto},
		{utils.HTTPSPort, uint16(utils.HTTPSPort), utils.TCPProto},
		{utils.NTPPort, uint16(utils.NTPPort), utils.UDPProto},
		{utils.SNMPPort, uint16(utils.SNMPPort), utils.UDPProto},
		{utils.IMAPSPort, uint16(utils.IMAPSPort), utils.TCPProto},
		{utils.MySQLPort, uint16(utils.MySQLPort), utils.TCPProto},
		{utils.HTTPAltPort, uint16(utils.HTTPAltPort), utils.TCPProto},
		{utils.HTTPSAltPort, uint16(utils.HTTPSAltPort), utils.TCPProto},
		{utils.P2PPort, uint16(utils.P2PPort), utils.TCPProto},
		{utils.BTPort, uint16(utils.BTPort), utils.TCPProto},
		{99999, uint16(utils.HTTPSPort), utils.TCPProto}, // default
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("port_%d", tc.port), func(t *testing.T) {
			session := NewSession()
			gf := new(GenericFlow).Generate(srcIP, dstIP, tc.port, session)
			if gf.L4DstPort != tc.wantPort {
				t.Errorf("L4DstPort: got %d, want %d", gf.L4DstPort, tc.wantPort)
			}
			if gf.Protocol != tc.wantProtocol {
				t.Errorf("Protocol: got %d, want %d", gf.Protocol, tc.wantProtocol)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Session edge cases
// ---------------------------------------------------------------------------

func TestNewSession_StartTime(t *testing.T) {
	t.Parallel()
	s := NewSession()
	st := s.StartTime()
	if st == 0 {
		t.Error("session start time should not be zero")
	}
	// Should be a reasonable nanosecond timestamp (after year 2020)
	if st < 1577836800000000000 {
		t.Errorf("start time %d seems too old", st)
	}
}

func TestSession_NextSeq_Concurrent(t *testing.T) {
	t.Parallel()
	s := NewSession()
	const numGoroutines = 100
	var wg sync.WaitGroup
	seen := make(map[uint32]bool)
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			seq := s.NextSeq()
			mu.Lock()
			if seen[seq] {
				t.Errorf("duplicate sequence number: %d", seq)
			}
			seen[seq] = true
			mu.Unlock()
		}()
	}

	wg.Wait()

	if len(seen) != numGoroutines {
		t.Errorf("expected %d unique sequences, got %d", numGoroutines, len(seen))
	}
}

// ---------------------------------------------------------------------------
// Field.String()
// ---------------------------------------------------------------------------

func TestField_String_Output(t *testing.T) {
	t.Parallel()
	f := Field{Type: IN_BYTES, Length: 4}
	s := f.String()

	if !strings.Contains(s, "Type: 1") {
		t.Error("Field String should contain type")
	}
	if !strings.Contains(s, "Length: 4") {
		t.Error("Field String should contain length")
	}
}
