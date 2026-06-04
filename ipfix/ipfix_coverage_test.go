// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package ipfix

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/utils"
)

// ---------------------------------------------------------------------------
// Options Template tests
// ---------------------------------------------------------------------------

func TestOptionsTemplateFlowSet_Generate(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()
	otfs := new(OptionsTemplateFlowSet).Generate(session)

	if otfs.FlowSetID != 0 {
		t.Errorf("FlowSetID should be 0 for template FlowSets, got %d", otfs.FlowSetID)
	}
	if otfs.Template.TemplateID != 257 {
		t.Errorf("TemplateID should be 257, got %d", otfs.Template.TemplateID)
	}
	if otfs.Template.ScopeFieldCount != 1 {
		t.Errorf("ScopeFieldCount should be 1, got %d", otfs.Template.ScopeFieldCount)
	}
	if len(otfs.Template.ScopeFields) != 1 {
		t.Fatalf("Expected 1 scope field, got %d", len(otfs.Template.ScopeFields))
	}
	if otfs.Template.ScopeFields[0].Type != ObservationDomainId {
		t.Errorf("Scope field type should be ObservationDomainId (%d), got %d",
			ObservationDomainId, otfs.Template.ScopeFields[0].Type)
	}
	if otfs.Template.ScopeFields[0].Length != 4 {
		t.Errorf("Scope field length should be 4, got %d", otfs.Template.ScopeFields[0].Length)
	}
	if otfs.Template.DataFieldCount != 2 {
		t.Errorf("DataFieldCount should be 2, got %d", otfs.Template.DataFieldCount)
	}
	if len(otfs.Template.DataFields) != 2 {
		t.Fatalf("Expected 2 data fields, got %d", len(otfs.Template.DataFields))
	}
	// Data field 0: ProcessName (variable-length, length=0 in template)
	if otfs.Template.DataFields[0].Type != ProcessName {
		t.Errorf("Data field[0] type should be ProcessName (%d), got %d",
			ProcessName, otfs.Template.DataFields[0].Type)
	}
	if otfs.Template.DataFields[0].Length != 0 {
		t.Errorf("Data field[0] length should be 0 (variable), got %d", otfs.Template.DataFields[0].Length)
	}
	// Data field 1: ProcessId
	if otfs.Template.DataFields[1].Type != ProcessId {
		t.Errorf("Data field[1] type should be ProcessId (%d), got %d",
			ProcessId, otfs.Template.DataFields[1].Type)
	}
	if otfs.Template.DataFields[1].Length != 4 {
		t.Errorf("Data field[1] length should be 4, got %d", otfs.Template.DataFields[1].Length)
	}
}

// ---------------------------------------------------------------------------
// Options Data tests
// ---------------------------------------------------------------------------

func TestOptionsDataFlowSet_Generate(t *testing.T) {
	t.Parallel()
	odfs := new(OptionsDataFlowSet).Generate(42, "flowgre-test", 12345)

	if odfs.FlowSetID != 257 {
		t.Errorf("FlowSetID should be 257 (matching Options TemplateID), got %d", odfs.FlowSetID)
	}
	if len(odfs.Records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(odfs.Records))
	}
	rec := odfs.Records[0]
	if rec.ObservationDomainId != 42 {
		t.Errorf("ObservationDomainId should be 42, got %d", rec.ObservationDomainId)
	}
	if rec.ProcessName != "flowgre-test" {
		t.Errorf("ProcessName should be 'flowgre-test', got %q", rec.ProcessName)
	}
	if rec.ProcessId != 12345 {
		t.Errorf("ProcessId should be 12345, got %d", rec.ProcessId)
	}
	// Length should be positive
	if odfs.Length == 0 {
		t.Error("Length should not be zero")
	}
}

func TestGenerateOptionsDataIPFIX(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()
	flow := GenerateOptionsDataIPFIX(99, session)

	if flow.Header.Version != 10 {
		t.Errorf("Version should be 10, got %d", flow.Header.Version)
	}
	if flow.Header.SourceID != 99 {
		t.Errorf("SourceID should be 99, got %d", flow.Header.SourceID)
	}
	if len(flow.OptionsDataFlowSets) != 1 {
		t.Fatalf("Expected 1 OptionsDataFlowSet, got %d", len(flow.OptionsDataFlowSets))
	}
	odfs := flow.OptionsDataFlowSets[0]
	if odfs.FlowSetID != 257 {
		t.Errorf("OptionsDataFlowSet FlowSetID should be 257, got %d", odfs.FlowSetID)
	}
	if len(odfs.Records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(odfs.Records))
	}
	// Process name should contain "flowgre"
	if odfs.Records[0].ProcessName == "" {
		t.Error("ProcessName should not be empty")
	}
	// PID should match actual process
	if odfs.Records[0].ProcessId != uint32(os.Getpid()) {
		t.Errorf("ProcessId should match os.Getpid(), got %d want %d",
			odfs.Records[0].ProcessId, os.Getpid())
	}
}

func TestOptionsData_ToBytes(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()
	flow := GenerateOptionsDataIPFIX(42, session)
	buf := flow.ToBytes()

	// Parse header
	var header Header
	if err := binary.Read(bytes.NewReader(buf.Bytes()), binary.BigEndian, &header); err != nil {
		t.Fatalf("Failed to parse header: %v", err)
	}
	if header.Version != 10 {
		t.Errorf("Parsed version should be 10, got %d", header.Version)
	}
	if header.SourceID != 42 {
		t.Errorf("Parsed SourceID should be 42, got %d", header.SourceID)
	}
}

// ---------------------------------------------------------------------------
// Minimal IPFIX Profile tests
// ---------------------------------------------------------------------------

func TestMinimalIPFIXProfile_Name(t *testing.T) {
	t.Parallel()
	p := &MinimalIPFIXProfile{}
	if p.Name() != "minimal" {
		t.Errorf("Name should be 'minimal', got %q", p.Name())
	}
}

func TestMinimalIPFIXProfile_TemplateFields(t *testing.T) {
	t.Parallel()
	p := &MinimalIPFIXProfile{}
	fields := p.TemplateFields()

	if len(fields) != 7 {
		t.Fatalf("Expected 7 fields, got %d", len(fields))
	}
	expected := []Field{
		{Type: InOctets, Length: 4},
		{Type: InPackets, Length: 4},
		{Type: SourceIPv4Address, Length: 4},
		{Type: DestinationIPv4Address, Length: 4},
		{Type: SourceTransportPort, Length: 2},
		{Type: DestinationTransportPort, Length: 2},
		{Type: ProtocolIdentifier, Length: 1},
	}
	for i, want := range expected {
		if fields[i] != want {
			t.Errorf("Field[%d] mismatch: got %+v, want %+v", i, fields[i], want)
		}
	}
}

func TestMinimalIPFIXFlow_Generate(t *testing.T) {
	t.Parallel()
	srcIP := net.ParseIP("10.0.0.1")
	dstIP := net.ParseIP("10.0.0.2")
	session := netflow.NewSession()

	mf := new(MinimalIPFIXFlow).Generate(srcIP, dstIP, utils.HTTPSPort, session)

	if mf.SourceIPv4Addr == 0 {
		t.Error("SourceIPv4Addr should not be zero")
	}
	if mf.DestIPv4Addr == 0 {
		t.Error("DestIPv4Addr should not be zero")
	}
	if mf.DestPort != uint16(utils.HTTPSPort) {
		t.Errorf("DestPort should be %d, got %d", utils.HTTPSPort, mf.DestPort)
	}
	if mf.ProtocolIdentifier != utils.TCPProto {
		t.Errorf("ProtocolIdentifier should be %d, got %d", utils.TCPProto, mf.ProtocolIdentifier)
	}
	if mf.InOctets == 0 {
		t.Error("InOctets should not be zero")
	}
	if mf.InPackets == 0 {
		t.Error("InPackets should not be zero")
	}
}

func TestMinimalIPFIXFlow_Generate_IPv6(t *testing.T) {
	t.Parallel()
	srcIP := net.ParseIP("2001:db8::1")
	dstIP := net.ParseIP("2001:db8::2")
	session := netflow.NewSession()

	mf := new(MinimalIPFIXFlow).Generate(srcIP, dstIP, utils.HTTPSPort, session)

	// Minimal profile only has IPv4 fields — they should be zeroed for IPv6
	if mf.SourceIPv4Addr != 0 {
		t.Errorf("IPv4 src should be zeroed for IPv6 input, got %d", mf.SourceIPv4Addr)
	}
	if mf.DestIPv4Addr != 0 {
		t.Errorf("IPv4 dst should be zeroed for IPv6 input, got %d", mf.DestIPv4Addr)
	}
}

func TestMinimalIPFIXFlow_Generate_AllProtocols(t *testing.T) {
	t.Parallel()
	srcIP := net.ParseIP("10.0.0.1")
	dstIP := net.ParseIP("10.0.0.2")
	session := netflow.NewSession()

	cases := []struct {
		port       int
		wantPort   uint16
		wantProto  uint8
	}{
		{utils.SSHPort, uint16(utils.SSHPort), utils.TCPProto},
		{utils.FTPPort, uint16(utils.FTPPort), utils.TCPProto},
		{utils.DNSPort, uint16(utils.DNSPort), utils.UDPProto},
		{utils.HTTPPort, uint16(utils.HTTPPort), utils.TCPProto},
		{utils.HTTPSPort, uint16(utils.HTTPSPort), utils.TCPProto},
		{99999, uint16(utils.HTTPSPort), utils.TCPProto}, // default
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("port_%d", tc.port), func(t *testing.T) {
			mf := new(MinimalIPFIXFlow).Generate(srcIP, dstIP, tc.port, session)
			if mf.DestPort != tc.wantPort {
				t.Errorf("DestPort: got %d, want %d", mf.DestPort, tc.wantPort)
			}
			if mf.ProtocolIdentifier != tc.wantProto {
				t.Errorf("ProtocolIdentifier: got %d, want %d", mf.ProtocolIdentifier, tc.wantProto)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Extended IPFIX Profile tests
// ---------------------------------------------------------------------------

func TestExtendedIPFIXProfile_Name(t *testing.T) {
	t.Parallel()
	p := &ExtendedIPFIXProfile{}
	if p.Name() != "extended" {
		t.Errorf("Name should be 'extended', got %q", p.Name())
	}
}

func TestExtendedIPFIXProfile_TemplateFields(t *testing.T) {
	t.Parallel()
	p := &ExtendedIPFIXProfile{}
	fields := p.TemplateFields()

	if len(fields) != 17 {
		t.Fatalf("Expected 17 fields, got %d", len(fields))
	}
	// Check first and last fields
	if fields[0].Type != InOctets {
		t.Errorf("First field should be InOctets, got %d", fields[0].Type)
	}
	if fields[len(fields)-1].Type != DestinationIPv6Address {
		t.Errorf("Last field should be DestinationIPv6Address, got %d", fields[len(fields)-1].Type)
	}
}

// ---------------------------------------------------------------------------
// Generic IPFIX Profile tests
// ---------------------------------------------------------------------------

func TestGenericIPFIXProfile_Name(t *testing.T) {
	t.Parallel()
	p := &GenericIPFIXProfile{}
	if p.Name() != "generic" {
		t.Errorf("Name should be 'generic', got %q", p.Name())
	}
}

// ---------------------------------------------------------------------------
// Debug helpers tests
// ---------------------------------------------------------------------------

func TestHeader_Size(t *testing.T) {
	t.Parallel()
	h := Header{
		Version:      10,
		FlowCount:    5,
		SysUptime:    1000,
		UnixSec:      1234567890,
		FlowSequence: 42,
		SourceID:     618,
	}
	want := binary.Size(Header{})
	got := h.size()
	if got != want {
		t.Errorf("Header.size(): got %d, want %d", got, want)
	}
}

func TestHeader_String(t *testing.T) {
	t.Parallel()
	h := Header{
		Version:      10,
		FlowCount:    5,
		SysUptime:    1000,
		UnixSec:      1234567890,
		FlowSequence: 42,
		SourceID:     618,
	}
	s := h.String()
	if s == "" {
		t.Error("Header.String() should not be empty")
	}
	// Should contain key values
	for _, substr := range []string{"Version: 10", "Count: 5", "SourceID: 618"} {
		if !bytes.Contains([]byte(s), []byte(substr)) {
			t.Errorf("Header.String() should contain %q, got: %s", substr, s)
		}
	}
}

func TestTemplateFlowSet_Size(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()
	tfs := new(TemplateFlowSet).Generate(session)

	size := tfs.size()
	if size != int(tfs.Length) {
		t.Errorf("TemplateFlowSet.size(): got %d, want %d (Length field)", size, tfs.Length)
	}
	if size <= 0 {
		t.Error("TemplateFlowSet.size() should be positive")
	}
}

func TestDataFlowSet_Size(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()
	dfs, err := new(DataFlowSet).Generate(5, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, session)
	if err != nil {
		t.Fatal(err)
	}

	size := dfs.size()
	if size != int(dfs.Length) {
		t.Errorf("DataFlowSet.size(): got %d, want %d (Length field)", size, dfs.Length)
	}
	if size <= 0 {
		t.Error("DataFlowSet.size() should be positive")
	}
}

func TestGetIPFIXSizes(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()
	flow, err := GenerateIPFIX(5, 42, "10.0.0.0/8", "10.0.0.0/8", session)
	if err != nil {
		t.Fatal(err)
	}
	s := GetIPFIXSizes(flow)

	if s == "" {
		t.Error("GetIPFIXSizes() should not be empty")
	}
	for _, substr := range []string{"Header Size:", "Template Size:", "Data Size:", "bytes"} {
		if !bytes.Contains([]byte(s), []byte(substr)) {
			t.Errorf("GetIPFIXSizes() should contain %q, got: %s", substr, s)
		}
	}
}

// ---------------------------------------------------------------------------
// IsValidIPFIX edge cases
// ---------------------------------------------------------------------------

func TestIsValidIPFIX_TooShort(t *testing.T) {
	t.Parallel()
	// Header is 20 bytes; a shorter payload should error
	ok, err := IsValidIPFIX([]byte{10})
	if ok {
		t.Error("IsValidIPFIX should reject too-short payload")
	}
	if err == nil {
		t.Error("IsValidIPFIX should return error for too-short payload")
	}
}

func TestIsValidIPFIX_Empty(t *testing.T) {
	t.Parallel()
	ok, err := IsValidIPFIX(nil)
	if ok {
		t.Error("IsValidIPFIX should reject nil payload")
	}
	if err == nil {
		t.Error("IsValidIPFIX should return error for nil payload")
	}
}

func TestIsValidIPFIX_RejectOtherVersions(t *testing.T) {
	t.Parallel()
	// Craft a 20-byte header with version 11
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint16(11)) // version
	binary.Write(buf, binary.BigEndian, uint16(1))  // flowCount
	binary.Write(buf, binary.BigEndian, uint32(1000)) // sysUptime
	binary.Write(buf, binary.BigEndian, uint32(1234567890)) // unixSec
	binary.Write(buf, binary.BigEndian, uint32(1)) // flowSequence
	binary.Write(buf, binary.BigEndian, uint32(42)) // sourceID

	ok, err := IsValidIPFIX(buf.Bytes())
	if ok {
		t.Error("IsValidIPFIX should reject version 11")
	}
	if err == nil {
		t.Error("IsValidIPFIX should return error for version 11")
	}
}

// ---------------------------------------------------------------------------
// UpdateTimeStamp edge cases
// ---------------------------------------------------------------------------

func TestUpdateTimeStamp_TooShort(t *testing.T) {
	t.Parallel()
	_, err := UpdateTimeStamp([]byte{10})
	if err == nil {
		t.Error("UpdateTimeStamp should return error for too-short payload")
	}
}

func TestUpdateTimeStamp_Empty(t *testing.T) {
	t.Parallel()
	_, err := UpdateTimeStamp(nil)
	if err == nil {
		t.Error("UpdateTimeStamp should return error for nil payload")
	}
}

func TestUpdateTimeStamp_PreservesPayload(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()
	flow, err := GenerateDataIPFIX(3, 42, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, session)
	if err != nil {
		t.Fatal(err)
	}
	buf := flow.ToBytes()
	original := buf.Bytes()

	updated, err := UpdateTimeStamp(original)
	if err != nil {
		t.Fatalf("UpdateTimeStamp error: %v", err)
	}

	// Length should be the same
	if len(updated) != len(original) {
		t.Errorf("UpdateTimeStamp changed payload length: got %d, want %d", len(updated), len(original))
	}

	// Only the UnixSec field (bytes 8-11) should differ
	for i := 0; i < len(original); i++ {
		if i >= 8 && i < 12 {
			continue // timestamp bytes may differ
		}
		if updated[i] != original[i] {
			t.Errorf("Byte[%d] changed unexpectedly: got 0x%02x, want 0x%02x",
				i, updated[i], original[i])
		}
	}
}

// ---------------------------------------------------------------------------
// ToBytes with Options Data — buffer length verification
// ---------------------------------------------------------------------------

func TestToBytes_OptionsData_BufferLengthMatchesFlowSetLengths(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()
	flow := GenerateOptionsDataIPFIX(42, session)
	buf := flow.ToBytes()

	expectedLen := binary.Size(flow.Header)
	for _, fs := range flow.OptionsDataFlowSets {
		expectedLen += int(fs.Length)
	}
	if buf.Len() != expectedLen {
		t.Errorf("OptionsData buffer length mismatch: got %d, want %d", buf.Len(), expectedLen)
	}
}

// ---------------------------------------------------------------------------
// Options Template + Options Data combined roundtrip
// ---------------------------------------------------------------------------

func TestOptionsTemplateAndData_RoundTrip(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()

	// Generate options template
	tFlow := GenerateTemplateIPFIX(100, session)
	tBuf := tFlow.ToBytes()

	// Parse and verify Options Template was serialized
	reader := bytes.NewReader(tBuf.Bytes())
	var header Header
	if err := binary.Read(reader, binary.BigEndian, &header); err != nil {
		t.Fatalf("Failed to parse header: %v", err)
	}
	if header.Version != 10 {
		t.Fatalf("Expected version 10, got %d", header.Version)
	}

	// Read flow sets
	foundTemplate := false
	foundOptionsTemplate := false
	for reader.Len() > 0 {
		var flowSetID, fsLength uint16
		if err := binary.Read(reader, binary.BigEndian, &flowSetID); err != nil {
			break
		}
		if err := binary.Read(reader, binary.BigEndian, &fsLength); err != nil {
			break
		}
		remaining := int(fsLength) - 4

		var templateID uint16
		if err := binary.Read(reader, binary.BigEndian, &templateID); err != nil {
			break
		}
		remaining -= 2

		if templateID == 256 {
			foundTemplate = true
		} else if templateID == 257 {
			foundOptionsTemplate = true
		}

		// Skip remaining bytes in this flow set
		reader.Seek(int64(remaining), 1)
	}

	if !foundTemplate {
		t.Error("Expected to find regular template (ID 256)")
	}
	if !foundOptionsTemplate {
		t.Error("Expected to find options template (ID 257)")
	}
}

// ---------------------------------------------------------------------------
// DataFlowSet edge cases
// ---------------------------------------------------------------------------

func TestDataFlowSet_Generate_ZeroFlowCount(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()
	dfs, err := new(DataFlowSet).Generate(0, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, session)
	if err != nil {
		t.Fatal(err)
	}

	if len(dfs.Items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(dfs.Items))
	}
	// Length should still be FlowSetID(2) + Length(2) = 4, possibly with padding
	if dfs.Length < 4 {
		t.Errorf("Length should be at least 4, got %d", dfs.Length)
	}
}

func TestGenericFlow_Generate_AllProtocols(t *testing.T) {
	t.Parallel()
	srcIP := net.ParseIP("10.0.0.1")
	dstIP := net.ParseIP("10.0.0.2")
	session := netflow.NewSession()

	cases := []struct {
		port      int
		wantPort  uint16
		wantProto uint8
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
			gf := new(GenericFlow).Generate(srcIP, dstIP, tc.port, session)
			if gf.DestPort != tc.wantPort {
				t.Errorf("DestPort: got %d, want %d", gf.DestPort, tc.wantPort)
			}
			if gf.ProtocolIdentifier != tc.wantProto {
				t.Errorf("ProtocolIdentifier: got %d, want %d", gf.ProtocolIdentifier, tc.wantProto)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// IPFIX version constant
// ---------------------------------------------------------------------------

func TestVersion_Constant(t *testing.T) {
	t.Parallel()
	if Version != 10 {
		t.Errorf("IPFIX Version should be 10, got %d", Version)
	}
}
