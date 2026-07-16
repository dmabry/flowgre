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

	size := tfs.rawSize()
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
	dfs, err := new(DataFlowSet).Generate(10, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, session)
	if err != nil {
		t.Fatal(err)
	}

	size := dfs.size()
	if size <= 0 {
		t.Errorf("DataFlowSet size should be positive, got %d", size)
	}
}

func TestDataFlowSet_Generate_ZeroFlowCount(t *testing.T) {
	t.Parallel()
	session := NewSession()
	dfs, err := new(DataFlowSet).Generate(0, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, session)
	if err != nil {
		t.Fatal(err)
	}

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
			dfs, err := new(DataFlowSet).Generate(1, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, session, tc.profile)
			if err != nil {
				t.Fatal(err)
			}
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
	nf, err := GenerateNetflow(flowcount, sourceID, "10.0.0.0/8", "10.0.0.0/8", session)
	if err != nil {
		t.Fatal(err)
	}

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
	nf, err := GenerateNetflow(flowcount, sourceID, "10.0.0.0/8", "10.0.0.0/8", session)
	if err != nil {
		t.Fatal(err)
	}

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
	nf, err := GenerateDataNetflow(flowcount, sourceID, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, session)
	if err != nil {
		t.Fatal(err)
	}

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

// ---------------------------------------------------------------------------
// Options Template FlowSet validation (RFC 3954 Section 6.1)
// ---------------------------------------------------------------------------

// buildOptionsTemplatePacket builds a minimal NetFlow v9 packet containing
// a Template FlowSet (ID=0) and an Options Template FlowSet (ID=1).
func buildOptionsTemplatePacket() []byte {
	var buf bytes.Buffer

	// Header: version=9, flowCount=2 (1 template + 1 options template)
	binary.Write(&buf, binary.BigEndian, uint16(9))
	binary.Write(&buf, binary.BigEndian, uint16(2))
	binary.Write(&buf, binary.BigEndian, uint32(1000))    // SysUptime
	binary.Write(&buf, binary.BigEndian, uint32(1000000)) // UnixSec
	binary.Write(&buf, binary.BigEndian, uint32(1))       // FlowSequence
	binary.Write(&buf, binary.BigEndian, uint32(618))     // SourceID

	// Template FlowSet (ID=0)
	// One template: ID=256, 2 fields (each 4 bytes = Type+Length)
	// Fields: (type=1, len=4), (type=2, len=4)
	// Total: FlowSet header(4) + tmplID(2) + fieldCount(2) + 2*fieldSpec(4) = 16
	tmplLen := uint16(4 + 2 + 2 + 2*4)                // = 16
	binary.Write(&buf, binary.BigEndian, uint16(0))   // FlowSetID
	binary.Write(&buf, binary.BigEndian, tmplLen)     // Length
	binary.Write(&buf, binary.BigEndian, uint16(256)) // TemplateID
	binary.Write(&buf, binary.BigEndian, uint16(2))   // FieldCount
	binary.Write(&buf, binary.BigEndian, uint16(1))   // Field 1 Type
	binary.Write(&buf, binary.BigEndian, uint16(4))   // Field 1 Length
	binary.Write(&buf, binary.BigEndian, uint16(2))   // Field 2 Type
	binary.Write(&buf, binary.BigEndian, uint16(4))   // Field 2 Length

	// Options Template FlowSet (ID=1) - RFC 3954 Section 6.1
	// One options template: ID=257, scopeLen=8 (2 scope fields), optLen=8 (2 option fields)
	// 6-byte header + 8 bytes scope field specifiers + 8 bytes option field specifiers
	// Total: FlowSet header(4) + header(6) + scope(8) + options(8) = 26
	optsLen := uint16(4 + 6 + 8 + 8)                  // = 26
	binary.Write(&buf, binary.BigEndian, uint16(1))   // FlowSetID
	binary.Write(&buf, binary.BigEndian, optsLen)     // Length
	binary.Write(&buf, binary.BigEndian, uint16(257)) // TemplateID
	binary.Write(&buf, binary.BigEndian, uint16(8))   // Option Scope Length (bytes)
	binary.Write(&buf, binary.BigEndian, uint16(8))   // Option Length (bytes)
	// Scope field specifiers (2 fields, each 4 bytes)
	binary.Write(&buf, binary.BigEndian, uint16(1)) // Scope field 1 Type
	binary.Write(&buf, binary.BigEndian, uint16(4)) // Scope field 1 Length
	binary.Write(&buf, binary.BigEndian, uint16(2)) // Scope field 2 Type
	binary.Write(&buf, binary.BigEndian, uint16(4)) // Scope field 2 Length
	// Option field specifiers (2 fields, each 4 bytes)
	binary.Write(&buf, binary.BigEndian, uint16(3)) // Option field 1 Type
	binary.Write(&buf, binary.BigEndian, uint16(4)) // Option field 1 Length
	binary.Write(&buf, binary.BigEndian, uint16(4)) // Option field 2 Type
	binary.Write(&buf, binary.BigEndian, uint16(4)) // Option field 2 Length

	return buf.Bytes()
}

func TestIsValidNetFlow_OptionsTemplate_Valid(t *testing.T) {
	t.Parallel()
	payload := buildOptionsTemplatePacket()

	valid, err := IsValidNetFlow(payload, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("valid Options Template packet should pass validation")
	}
}

func TestIsValidNetFlow_OptionsTemplate_BadTemplateID(t *testing.T) {
	t.Parallel()
	payload := buildOptionsTemplatePacket()
	// Template FlowSet is 16 bytes, so Options Template FlowSet starts at offset 36
	// Options Template TemplateID is at offset 36 + 4 = 40
	// Change TemplateID from 257 to 255 (below 256)
	payload[40] = 0
	payload[41] = 255

	valid, err := IsValidNetFlow(payload, 9)
	if err == nil {
		t.Error("expected error for Options Template ID below 256")
	}
	if valid {
		t.Error("Options Template with ID below 256 should not be valid")
	}
}

func TestIsValidNetFlow_OptionsTemplate_BadScopeLength(t *testing.T) {
	t.Parallel()
	payload := buildOptionsTemplatePacket()
	// Option Scope Length is at offset 42 (after TemplateID at 40-41)
	// Change from 8 to 6 (not multiple of 4)
	binary.BigEndian.PutUint16(payload[42:44], 6)

	valid, err := IsValidNetFlow(payload, 9)
	if err == nil {
		t.Error("expected error for Option Scope Length not multiple of 4")
	}
	if valid {
		t.Error("Options Template with bad Scope Length should not be valid")
	}
}

func TestIsValidNetFlow_OptionsTemplate_BadOptionLength(t *testing.T) {
	t.Parallel()
	payload := buildOptionsTemplatePacket()
	// Option Length is at offset 44
	// Change from 8 to 7 (not multiple of 4)
	binary.BigEndian.PutUint16(payload[44:46], 7)

	valid, err := IsValidNetFlow(payload, 9)
	if err == nil {
		t.Error("expected error for Option Length not multiple of 4")
	}
	if valid {
		t.Error("Options Template with bad Option Length should not be valid")
	}
}

func TestIsValidNetFlow_OptionsTemplate_ZeroFieldLength(t *testing.T) {
	t.Parallel()
	payload := buildOptionsTemplatePacket()
	// First scope field length is at offset 48 (6-byte header ends at 46, first field type at 46-47, length at 48-49)
	// Change from 4 to 0
	binary.BigEndian.PutUint16(payload[48:50], 0)

	valid, err := IsValidNetFlow(payload, 9)
	if err == nil {
		t.Error("expected error for zero-length scope field")
	}
	if valid {
		t.Error("Options Template with zero-length field should not be valid")
	}
}

// ---------------------------------------------------------------------------
// Multi-record FlowCount validation
// ---------------------------------------------------------------------------

func TestIsValidNetFlow_MultiRecord_FlowCount(t *testing.T) {
	t.Parallel()
	// Generate packet with 10 data flows + 1 template = 11 records
	session := NewSession()
	nf, err := GenerateNetflow(10, 618, "10.0.0.0/8", "10.0.0.0/8", session)
	if err != nil {
		t.Fatal(err)
	}

	buf := nf.ToBytes()
	payload := buf.Bytes()

	valid, err := IsValidNetFlow(payload, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("valid multi-record NetFlow should pass validation")
	}

	// Verify header FlowCount is correct
	var header Header
	binary.Read(bytes.NewReader(payload), binary.BigEndian, &header)
	if header.FlowCount != 11 {
		t.Errorf("expected FlowCount 11 (1 template + 10 data), got %d", header.FlowCount)
	}
}

func TestIsValidNetFlow_DataOnly_Accepted(t *testing.T) {
	t.Parallel()
	// GenerateDataNetflow produces data-only packets (no template)
	session := NewSession()
	nf, err := GenerateDataNetflow(10, 618, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, session)
	if err != nil {
		t.Fatal(err)
	}

	buf := nf.ToBytes()
	payload := buf.Bytes()

	valid, err := IsValidNetFlow(payload, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("data-only NetFlow packet should pass validation (template sent in prior packet)")
	}
}

func TestIsValidNetFlow_FlowCount_TooLow(t *testing.T) {
	t.Parallel()
	session := NewSession()
	nf, err := GenerateNetflow(10, 618, "10.0.0.0/8", "10.0.0.0/8", session)
	if err != nil {
		t.Fatal(err)
	}

	buf := nf.ToBytes()
	payload := buf.Bytes()

	// Set FlowCount to 1 (too low for 1 template + 10 data records)
	binary.BigEndian.PutUint16(payload[2:4], 1)

	valid, err := IsValidNetFlow(payload, 9)
	if err == nil {
		t.Error("expected error for FlowCount less than actual records")
	}
	if valid {
		t.Error("packet with FlowCount less than counted records should not be valid")
	}
}

func TestIsValidNetFlow_FlowCount_Zero(t *testing.T) {
	t.Parallel()
	session := NewSession()
	nf, err := GenerateNetflow(10, 618, "10.0.0.0/8", "10.0.0.0/8", session)
	if err != nil {
		t.Fatal(err)
	}

	buf := nf.ToBytes()
	payload := buf.Bytes()

	// Set FlowCount to 0
	binary.BigEndian.PutUint16(payload[2:4], 0)

	valid, err := IsValidNetFlow(payload, 9)
	if err == nil {
		t.Error("expected error for zero FlowCount")
	}
	if valid {
		t.Error("packet with zero FlowCount should not be valid")
	}
}

// ---------------------------------------------------------------------------
// countDataRecordsRange helper
// ---------------------------------------------------------------------------

func TestCountDataRecordsRange(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		dataLen int
		recSize int
		wantMin int
		wantMax int
	}{
		{"exact_fit", 100, 10, 10, 10},
		{"with_padding_1", 101, 10, 10, 10},
		{"with_padding_2", 102, 10, 10, 10},
		{"with_padding_3", 103, 10, 10, 10},
		{"single_record", 10, 10, 1, 1},
		{"single_record_padded", 12, 10, 1, 1},
		{"no_valid_padding", 105, 10, -1, -1},
		// 1-byte record: 4 data bytes can be 1 record + 3 padding or 4 records
		{"short_record_ambiguous", 4, 1, 1, 4},
		// 1-byte record: 3 data bytes can be 1 record + 2 padding or 3 records
		{"short_record_ambiguous_3", 3, 1, 1, 3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			min, max := countDataRecordsRange(tc.dataLen, tc.recSize)
			if min != tc.wantMin || max != tc.wantMax {
				t.Errorf("countDataRecordsRange(%d, %d) = (%d, %d), want (%d, %d)",
					tc.dataLen, tc.recSize, min, max, tc.wantMin, tc.wantMax)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// countDataRecords helper (unambiguous only)
// ---------------------------------------------------------------------------

func TestCountDataRecords(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		dataLen   int
		recSize   int
		wantCount int
	}{
		{"exact_fit", 100, 10, 10},
		{"with_padding_1", 101, 10, 10},
		{"with_padding_2", 102, 10, 10},
		{"with_padding_3", 103, 10, 10},
		{"single_record", 10, 10, 1},
		{"single_record_padded", 12, 10, 1},
		{"no_valid_padding", 105, 10, -1},
		// Ambiguous case: returns -1 (multiple valid interpretations)
		{"short_record_ambiguous", 4, 1, -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := countDataRecords(tc.dataLen, tc.recSize)
			if got != tc.wantCount {
				t.Errorf("countDataRecords(%d, %d) = %d, want %d", tc.dataLen, tc.recSize, got, tc.wantCount)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Inflated FlowCount rejection (all templates known)
// ---------------------------------------------------------------------------

func TestIsValidNetFlow_FlowCount_Inflated_Rejected(t *testing.T) {
	t.Parallel()
	// Generate packet with 10 data flows + 1 template = 11 records
	session := NewSession()
	nf, err := GenerateNetflow(10, 618, "10.0.0.0/8", "10.0.0.0/8", session)
	if err != nil {
		t.Fatal(err)
	}

	buf := nf.ToBytes()
	payload := buf.Bytes()

	// Inflate FlowCount from 11 to 12
	binary.BigEndian.PutUint16(payload[2:4], 12)

	valid, err := IsValidNetFlow(payload, 9)
	if err == nil {
		t.Error("expected error for inflated FlowCount when all templates are known")
	}
	if valid {
		t.Error("packet with FlowCount 12 but only 11 records should not be valid")
	}
}

// ---------------------------------------------------------------------------
// Options Template + Options Data Records: exact count
// ---------------------------------------------------------------------------

// buildOptionsDataPacket builds a NetFlow v9 packet with:
// - Template FlowSet (ID=0): 1 template (ID=256, 2 fields, record size=8)
// - Options Template FlowSet (ID=1): 1 options template (ID=257, 2 scope + 2 option fields, record size=16)
// - Data FlowSet (ID=256): 3 data records (each 8 bytes)
// - Options Data FlowSet (ID=257): 2 options data records (each 16 bytes)
// Total records: 1 (template) + 1 (options template) + 3 (data) + 2 (options data) = 7
func buildOptionsDataPacket() []byte {
	var buf bytes.Buffer

	// Header: version=9, flowCount=7
	binary.Write(&buf, binary.BigEndian, uint16(9))
	binary.Write(&buf, binary.BigEndian, uint16(7))
	binary.Write(&buf, binary.BigEndian, uint32(1000))
	binary.Write(&buf, binary.BigEndian, uint32(1000000))
	binary.Write(&buf, binary.BigEndian, uint32(1))
	binary.Write(&buf, binary.BigEndian, uint32(618))

	// Template FlowSet (ID=0): 1 template, 2 fields (type=1,len=4), (type=2,len=4)
	// Record size = 4+4 = 8
	tmplLen := uint16(4 + 2 + 2 + 2*4) // = 16
	binary.Write(&buf, binary.BigEndian, uint16(0))
	binary.Write(&buf, binary.BigEndian, tmplLen)
	binary.Write(&buf, binary.BigEndian, uint16(256)) // TemplateID
	binary.Write(&buf, binary.BigEndian, uint16(2))   // FieldCount
	binary.Write(&buf, binary.BigEndian, uint16(1))
	binary.Write(&buf, binary.BigEndian, uint16(4))
	binary.Write(&buf, binary.BigEndian, uint16(2))
	binary.Write(&buf, binary.BigEndian, uint16(4))

	// Options Template FlowSet (ID=1): 1 options template
	// 2 scope fields (type=1,len=8), (type=2,len=8) -> record data size = 16
	// 2 option fields (type=3,len=4), (type=4,len=4) -> included in record data size
	// Record size = 8+8+4+4 = 24
	scopeLen := uint16(8)                                  // 2 scope field specifiers
	optLen := uint16(8)                                    // 2 option field specifiers
	optsLen := uint16(4 + 6 + int(scopeLen) + int(optLen)) // = 26
	binary.Write(&buf, binary.BigEndian, uint16(1))
	binary.Write(&buf, binary.BigEndian, optsLen)
	binary.Write(&buf, binary.BigEndian, uint16(257)) // Options TemplateID
	binary.Write(&buf, binary.BigEndian, scopeLen)    // Option Scope Length
	binary.Write(&buf, binary.BigEndian, optLen)      // Option Length
	// Scope field specifiers
	binary.Write(&buf, binary.BigEndian, uint16(1))
	binary.Write(&buf, binary.BigEndian, uint16(8))
	binary.Write(&buf, binary.BigEndian, uint16(2))
	binary.Write(&buf, binary.BigEndian, uint16(8))
	// Option field specifiers
	binary.Write(&buf, binary.BigEndian, uint16(3))
	binary.Write(&buf, binary.BigEndian, uint16(4))
	binary.Write(&buf, binary.BigEndian, uint16(4))
	binary.Write(&buf, binary.BigEndian, uint16(4))

	// Data FlowSet (ID=256): 3 records, each 8 bytes, + padding
	// 3 * 8 = 24 bytes, no padding needed (24 % 4 == 0)
	dataLen := uint16(4 + 3*8) // = 28
	binary.Write(&buf, binary.BigEndian, uint16(256))
	binary.Write(&buf, binary.BigEndian, dataLen)
	for range 3 {
		binary.Write(&buf, binary.BigEndian, uint32(0x11223344))
		binary.Write(&buf, binary.BigEndian, uint32(0x55667788))
	}

	// Options Data FlowSet (ID=257): 2 records, each 24 bytes, + padding
	// 2 * 24 = 48 bytes, no padding needed (48 % 4 == 0)
	optsDataLen := uint16(4 + 2*24) // = 52
	binary.Write(&buf, binary.BigEndian, uint16(257))
	binary.Write(&buf, binary.BigEndian, optsDataLen)
	for range 2 {
		// Each options data record: 8+8+4+4 = 24 bytes
		binary.Write(&buf, binary.BigEndian, uint64(0x0011223344556677))
		binary.Write(&buf, binary.BigEndian, uint64(0x8899AABBCCDDEEFF))
		binary.Write(&buf, binary.BigEndian, uint32(0x11223344))
		binary.Write(&buf, binary.BigEndian, uint32(0x55667788))
	}

	return buf.Bytes()
}

func TestIsValidNetFlow_OptionsData_ExactCount(t *testing.T) {
	t.Parallel()
	payload := buildOptionsDataPacket()

	valid, err := IsValidNetFlow(payload, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("valid Options Data packet with correct FlowCount should pass validation")
	}

	// Verify the header FlowCount is exactly 7
	var header Header
	binary.Read(bytes.NewReader(payload), binary.BigEndian, &header)
	if header.FlowCount != 7 {
		t.Errorf("expected FlowCount 7, got %d", header.FlowCount)
	}
}

func TestIsValidNetFlow_OptionsData_InflatedFlowCount_Rejected(t *testing.T) {
	t.Parallel()
	payload := buildOptionsDataPacket()

	// Inflate FlowCount from 7 to 10
	binary.BigEndian.PutUint16(payload[2:4], 10)

	valid, err := IsValidNetFlow(payload, 9)
	if err == nil {
		t.Error("expected error for inflated FlowCount when all templates are known")
	}
	if valid {
		t.Error("packet with FlowCount 10 but only 7 records should not be valid")
	}
}

// ---------------------------------------------------------------------------
// Mixed known/unknown Data FlowSets: unverifiable count accepted
// ---------------------------------------------------------------------------

func TestIsValidNetFlow_MixedTemplates_Accepted(t *testing.T) {
	t.Parallel()
	// Generate a packet with a template + data for template 256,
	// then manually append a Data FlowSet for an unknown template (ID=258).
	// The unknown template's records can't be counted, so FlowCount >= totalRecords
	// is the only check, and an inflated FlowCount is accepted.
	session := NewSession()
	nf, err := GenerateNetflow(5, 618, "10.0.0.0/8", "10.0.0.0/8", session)
	if err != nil {
		t.Fatal(err)
	}

	buf := nf.ToBytes()

	// Append a Data FlowSet for unknown template 258 (8 bytes of opaque data)
	// Length = 4 (FlowSet header) + 8 (data) = 12
	unknownLen := uint16(4 + 8)
	binary.Write(&buf, binary.BigEndian, uint16(258))
	binary.Write(&buf, binary.BigEndian, unknownLen)
	binary.Write(&buf, binary.BigEndian, uint64(0x1122334455667788))

	payload := buf.Bytes()

	// Set FlowCount to 10 (1 template + 5 data + unknown records).
	// Since template 258 is unknown, the validator can't count those records,
	// so FlowCount >= 6 (counted) is the only requirement.
	binary.BigEndian.PutUint16(payload[2:4], 10)

	valid, err := IsValidNetFlow(payload, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("packet with unknown template should accept FlowCount >= counted records")
	}
}

func TestIsValidNetFlow_MixedTemplates_FlowCountTooLow_Rejected(t *testing.T) {
	t.Parallel()
	session := NewSession()
	nf, err := GenerateNetflow(5, 618, "10.0.0.0/8", "10.0.0.0/8", session)
	if err != nil {
		t.Fatal(err)
	}

	buf := nf.ToBytes()

	// Append a Data FlowSet for unknown template 258
	unknownLen := uint16(4 + 8)
	binary.Write(&buf, binary.BigEndian, uint16(258))
	binary.Write(&buf, binary.BigEndian, unknownLen)
	binary.Write(&buf, binary.BigEndian, uint64(0x1122334455667788))

	payload := buf.Bytes()

	// Minimum count is 7 (1 template + 5 data + 1 unresolved FlowSet).
	// FlowCount=6 is below the minimum.
	binary.BigEndian.PutUint16(payload[2:4], 6)

	valid, err := IsValidNetFlow(payload, 9)
	if err == nil {
		t.Error("expected error for FlowCount less than counted records")
	}
	if valid {
		t.Error("packet with FlowCount 6 but 7+ records should not be valid")
	}
}

func TestIsValidNetFlow_MixedTemplates_FlowCountExactMinimum_Accepted(t *testing.T) {
	t.Parallel()
	session := NewSession()
	nf, err := GenerateNetflow(5, 618, "10.0.0.0/8", "10.0.0.0/8", session)
	if err != nil {
		t.Fatal(err)
	}

	buf := nf.ToBytes()

	// Append a Data FlowSet for unknown template 258
	unknownLen := uint16(4 + 8)
	binary.Write(&buf, binary.BigEndian, uint16(258))
	binary.Write(&buf, binary.BigEndian, unknownLen)
	binary.Write(&buf, binary.BigEndian, uint64(0x1122334455667788))

	payload := buf.Bytes()

	// Minimum count is 7 (1 template + 5 data + 1 unresolved FlowSet).
	// FlowCount=7 equals the minimum and should be accepted.
	binary.BigEndian.PutUint16(payload[2:4], 7)

	valid, err := IsValidNetFlow(payload, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("packet with FlowCount equal to minimum should be accepted")
	}
}

// ---------------------------------------------------------------------------
// Ambiguous short-record padding: header resolves count
// ---------------------------------------------------------------------------

// buildShortRecordPacket builds a NetFlow v9 packet with a 1-byte template
// record and a Data FlowSet containing 4 bytes of data. With 1-byte records,
// the 4 data bytes can be interpreted as:
// - 1 record + 3 padding bytes (FlowCount = 2: 1 template + 1 data)
// - 4 records + 0 padding bytes (FlowCount = 5: 1 template + 4 data)
// Both are valid per RFC 3954. The header FlowCount resolves the ambiguity.
func buildShortRecordPacket() []byte {
	var buf bytes.Buffer

	// Header: version=9, flowCount set below
	binary.Write(&buf, binary.BigEndian, uint16(9))
	binary.Write(&buf, binary.BigEndian, uint16(0)) // placeholder
	binary.Write(&buf, binary.BigEndian, uint32(1000))
	binary.Write(&buf, binary.BigEndian, uint32(1000000))
	binary.Write(&buf, binary.BigEndian, uint32(1))
	binary.Write(&buf, binary.BigEndian, uint32(618))

	// Template FlowSet (ID=0): 1 template with 1 field (type=1, len=1)
	// Record size = 1 byte
	tmplLen := uint16(4 + 2 + 2 + 4) // FlowSet header + tmplID + fieldCount + field
	binary.Write(&buf, binary.BigEndian, uint16(0))
	binary.Write(&buf, binary.BigEndian, tmplLen)
	binary.Write(&buf, binary.BigEndian, uint16(256)) // TemplateID
	binary.Write(&buf, binary.BigEndian, uint16(1))   // FieldCount
	binary.Write(&buf, binary.BigEndian, uint16(1))   // Field type
	binary.Write(&buf, binary.BigEndian, uint16(1))   // Field length = 1 byte

	// Data FlowSet (ID=256): 4 bytes of data
	// With 1-byte records: could be 1 record + 3 padding or 4 records + 0 padding
	dataLen := uint16(4 + 4) // FlowSet header + 4 data bytes
	binary.Write(&buf, binary.BigEndian, uint16(256))
	binary.Write(&buf, binary.BigEndian, dataLen)
	binary.Write(&buf, binary.BigEndian, uint32(0x11223344))

	return buf.Bytes()
}

func TestIsValidNetFlow_ShortRecord_Padded_FlowCount2(t *testing.T) {
	t.Parallel()
	payload := buildShortRecordPacket()
	// FlowCount = 2: 1 template + 1 data record (3 padding bytes)
	binary.BigEndian.PutUint16(payload[2:4], 2)

	valid, err := IsValidNetFlow(payload, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("packet with FlowCount=2 (1 template + 1 padded record) should be valid")
	}
}

func TestIsValidNetFlow_ShortRecord_Unpadded_FlowCount5(t *testing.T) {
	t.Parallel()
	payload := buildShortRecordPacket()
	// FlowCount = 5: 1 template + 4 data records (0 padding bytes)
	binary.BigEndian.PutUint16(payload[2:4], 5)

	valid, err := IsValidNetFlow(payload, 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("packet with FlowCount=5 (1 template + 4 unpadded records) should be valid")
	}
}

func TestIsValidNetFlow_ShortRecord_OutOfRange_FlowCount1(t *testing.T) {
	t.Parallel()
	payload := buildShortRecordPacket()
	// FlowCount = 1: below the minimum (2)
	binary.BigEndian.PutUint16(payload[2:4], 1)

	valid, err := IsValidNetFlow(payload, 9)
	if err == nil {
		t.Error("expected error for FlowCount below valid range")
	}
	if valid {
		t.Error("packet with FlowCount=1 should not be valid (min is 2)")
	}
}

func TestIsValidNetFlow_ShortRecord_OutOfRange_FlowCount6(t *testing.T) {
	t.Parallel()
	payload := buildShortRecordPacket()
	// FlowCount = 6: above the maximum (5)
	binary.BigEndian.PutUint16(payload[2:4], 6)

	valid, err := IsValidNetFlow(payload, 9)
	if err == nil {
		t.Error("expected error for FlowCount above valid range")
	}
	if valid {
		t.Error("packet with FlowCount=6 should not be valid (max is 5)")
	}
}
