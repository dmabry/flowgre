// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package ipfix

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/dmabry/flowgre/utils"
)

// ---------------------------------------------------------------------------
// Options Template tests
// ---------------------------------------------------------------------------

func TestOptionsTemplateFlowSet_Generate(t *testing.T) {
	t.Parallel()
	otfs := new(OptionsTemplateFlowSet).Generate(nil)

	// RFC 7011: Options Template Sets use Set ID 3
	if otfs.FlowSetID != SetIDOptionsTemplate {
		t.Errorf("FlowSetID should be %d for Options Template Sets, got %d", SetIDOptionsTemplate, otfs.FlowSetID)
	}
	if otfs.Template.TemplateID != 257 {
		t.Errorf("TemplateID should be 257, got %d", otfs.Template.TemplateID)
	}
	if otfs.Template.ScopeFieldCount != 1 {
		t.Errorf("ScopeFieldCount should be 1, got %d", otfs.Template.ScopeFieldCount)
	}
	if otfs.Template.FieldCount < 1 {
		t.Errorf("FieldCount should be >= 1, got %d", otfs.Template.FieldCount)
	}
	if len(otfs.Template.Fields) != int(otfs.Template.FieldCount) {
		t.Fatalf("Fields length should match FieldCount, got %d fields, %d count",
			len(otfs.Template.Fields), otfs.Template.FieldCount)
	}
	if otfs.Template.Fields[0].Type != ObservationDomainId {
		t.Errorf("Scope field type should be ObservationDomainId (%d), got %d",
			ObservationDomainId, otfs.Template.Fields[0].Type)
	}
	if otfs.Template.Fields[0].Length != 4 {
		t.Errorf("Scope field length should be 4, got %d", otfs.Template.Fields[0].Length)
	}
}

// ---------------------------------------------------------------------------
// Options Data tests
// ---------------------------------------------------------------------------

func TestOptionsDataFlowSet_Generate(t *testing.T) {
	t.Parallel()
	odfs := new(OptionsDataFlowSet).Generate(42)

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
	if odfs.Length == 0 {
		t.Error("Length should not be zero")
	}
}

func TestGenerateOptionsDataIPFIX(t *testing.T) {
	t.Parallel()
	seq := NewIPFIXSequence()
	flow := GenerateOptionsDataIPFIX(99, seq)

	if flow.Header.Version != 10 {
		t.Errorf("Version should be 10, got %d", flow.Header.Version)
	}
	if flow.Header.ObservationDomainId != 99 {
		t.Errorf("ObservationDomainId should be 99, got %d", flow.Header.ObservationDomainId)
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
}

func TestOptionsData_ToBytes(t *testing.T) {
	t.Parallel()
	seq := NewIPFIXSequence()
	flow := GenerateOptionsDataIPFIX(42, seq)
	buf, _ := flow.ToBytes()

	var header Header
	if err := binary.Read(bytes.NewReader(buf.Bytes()), binary.BigEndian, &header); err != nil {
		t.Fatalf("Failed to parse header: %v", err)
	}
	if header.Version != 10 {
		t.Errorf("Parsed version should be 10, got %d", header.Version)
	}
	if header.ObservationDomainId != 42 {
		t.Errorf("Parsed ObservationDomainId should be 42, got %d", header.ObservationDomainId)
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
		{Type: OctetDeltaCount, Length: 4},
		{Type: PacketDeltaCount, Length: 4},
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

	mf := new(MinimalIPFIXFlow).Generate(srcIP, dstIP, utils.HTTPSPort, nil)

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
	if mf.OctetDeltaCount == 0 {
		t.Error("OctetDeltaCount should not be zero")
	}
	if mf.PacketDeltaCount == 0 {
		t.Error("PacketDeltaCount should not be zero")
	}
}

func TestMinimalIPFIXFlow_Generate_IPv6(t *testing.T) {
	t.Parallel()
	srcIP := net.ParseIP("2001:db8::1")
	dstIP := net.ParseIP("2001:db8::2")

	mf := new(MinimalIPFIXFlow).Generate(srcIP, dstIP, utils.HTTPSPort, nil)

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
		{99999, uint16(utils.HTTPSPort), utils.TCPProto},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("port_%d", tc.port), func(t *testing.T) {
			mf := new(MinimalIPFIXFlow).Generate(srcIP, dstIP, tc.port, nil)
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
	if fields[0].Type != OctetDeltaCount {
		t.Errorf("First field should be OctetDeltaCount, got %d", fields[0].Type)
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
		Version:             10,
		Length:              100,
		ExportTime:          1234567890,
		SequenceNumber:      42,
		ObservationDomainId: 618,
	}
	want := 16
	got := h.size()
	if got != want {
		t.Errorf("Header.size(): got %d, want %d", got, want)
	}
}

func TestHeader_String(t *testing.T) {
	t.Parallel()
	h := Header{
		Version:             10,
		Length:              100,
		ExportTime:          1234567890,
		SequenceNumber:      42,
		ObservationDomainId: 618,
	}
	s := h.String()
	if s == "" {
		t.Error("Header.String() should not be empty")
	}
	for _, substr := range []string{"Version: 10", "ObservationDomainId: 618"} {
		if !bytes.Contains([]byte(s), []byte(substr)) {
			t.Errorf("Header.String() should contain %q, got: %s", substr, s)
		}
	}
}

func TestTemplateFlowSet_Size(t *testing.T) {
	t.Parallel()
	tfs := new(TemplateFlowSet).Generate(nil)

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
	dfs, err := new(DataFlowSet).Generate(5, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, nil)
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
	seq := NewIPFIXSequence()
	flow, err := GenerateIPFIX(5, 42, "10.0.0.0/8", "10.0.0.0/8", seq)
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
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint16(11))
	binary.Write(buf, binary.BigEndian, uint16(16))
	binary.Write(buf, binary.BigEndian, uint32(1234567890))
	binary.Write(buf, binary.BigEndian, uint32(1))
	binary.Write(buf, binary.BigEndian, uint32(42))

	ok, err := IsValidIPFIX(buf.Bytes())
	if ok {
		t.Error("IsValidIPFIX should reject version 11")
	}
	if err == nil {
		t.Error("IsValidIPFIX should return error for version 11")
	}
}

func TestIsValidIPFIX_RejectReservedSetIDs(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint16(10))
	binary.Write(buf, binary.BigEndian, uint16(20))
	binary.Write(buf, binary.BigEndian, uint32(1234567890))
	binary.Write(buf, binary.BigEndian, uint32(1))
	binary.Write(buf, binary.BigEndian, uint32(42))
	binary.Write(buf, binary.BigEndian, uint16(0))
	binary.Write(buf, binary.BigEndian, uint16(4))

	ok, err := IsValidIPFIX(buf.Bytes())
	if ok {
		t.Error("IsValidIPFIX should reject reserved Set ID 0")
	}
	if err == nil {
		t.Error("IsValidIPFIX should return error for reserved Set ID")
	}
}

func TestIsValidIPFIX_RejectUnassignedSetIDs(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint16(10))
	binary.Write(buf, binary.BigEndian, uint16(20))
	binary.Write(buf, binary.BigEndian, uint32(1234567890))
	binary.Write(buf, binary.BigEndian, uint32(1))
	binary.Write(buf, binary.BigEndian, uint32(42))
	binary.Write(buf, binary.BigEndian, uint16(100))
	binary.Write(buf, binary.BigEndian, uint16(4))

	ok, err := IsValidIPFIX(buf.Bytes())
	if ok {
		t.Error("IsValidIPFIX should reject unassigned Set ID 100")
	}
	if err == nil {
		t.Error("IsValidIPFIX should return error for unassigned Set ID")
	}
}

func TestIsValidIPFIX_MessageLengthMismatch(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint16(10))
	binary.Write(buf, binary.BigEndian, uint16(100))
	binary.Write(buf, binary.BigEndian, uint32(1234567890))
	binary.Write(buf, binary.BigEndian, uint32(1))
	binary.Write(buf, binary.BigEndian, uint32(42))

	ok, err := IsValidIPFIX(buf.Bytes())
	if ok {
		t.Error("IsValidIPFIX should reject length mismatch")
	}
	if err == nil {
		t.Error("IsValidIPFIX should return error for length mismatch")
	}
}

func TestIsValidIPFIX_RejectTrailingData(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint16(10))
	binary.Write(buf, binary.BigEndian, uint16(16))
	binary.Write(buf, binary.BigEndian, uint32(1234567890))
	binary.Write(buf, binary.BigEndian, uint32(1))
	binary.Write(buf, binary.BigEndian, uint32(42))
	buf.Write([]byte{0, 0, 0, 0})

	ok, err := IsValidIPFIX(buf.Bytes())
	if ok {
		t.Error("IsValidIPFIX should reject trailing data")
	}
	if err == nil {
		t.Error("IsValidIPFIX should return error for trailing data")
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
	seq := NewIPFIXSequence()
	flow, err := GenerateDataIPFIX(3, 42, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, seq)
	if err != nil {
		t.Fatal(err)
	}
	buf, _ := flow.ToBytes()
	original := buf.Bytes()

	updated, err := UpdateTimeStamp(original)
	if err != nil {
		t.Fatalf("UpdateTimeStamp error: %v", err)
	}

	if len(updated) != len(original) {
		t.Errorf("UpdateTimeStamp changed payload length: got %d, want %d", len(updated), len(original))
	}

	for i := 0; i < len(original); i++ {
		if i >= 4 && i < 8 {
			continue
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
	seq := NewIPFIXSequence()
	flow := GenerateOptionsDataIPFIX(42, seq)
	buf, _ := flow.ToBytes()

	expectedLen := 16
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
	seq := NewIPFIXSequence()

	tFlow := GenerateTemplateIPFIX(100, seq)
	tBuf, _ := tFlow.ToBytes()

	reader := bytes.NewReader(tBuf.Bytes())
	var header Header
	if err := binary.Read(reader, binary.BigEndian, &header); err != nil {
		t.Fatalf("Failed to parse header: %v", err)
	}
	if header.Version != 10 {
		t.Fatalf("Expected version 10, got %d", header.Version)
	}

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

		if flowSetID == SetIDTemplate && templateID == 256 {
			foundTemplate = true
		} else if flowSetID == SetIDOptionsTemplate && templateID == 257 {
			foundOptionsTemplate = true
		}

		reader.Seek(int64(remaining-2), 1)
	}

	if !foundTemplate {
		t.Error("Expected to find regular template (Set ID 2, Template ID 256)")
	}
	if !foundOptionsTemplate {
		t.Error("Expected to find options template (Set ID 3, Template ID 257)")
	}
}

// ---------------------------------------------------------------------------
// DataFlowSet edge cases
// ---------------------------------------------------------------------------

func TestDataFlowSet_Generate_ZeroFlowCount(t *testing.T) {
	t.Parallel()
	dfs, err := new(DataFlowSet).Generate(0, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(dfs.Items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(dfs.Items))
	}
	if dfs.Length < 4 {
		t.Errorf("Length should be at least 4, got %d", dfs.Length)
	}
}

func TestGenericFlow_Generate_AllProtocols(t *testing.T) {
	t.Parallel()
	srcIP := net.ParseIP("10.0.0.1")
	dstIP := net.ParseIP("10.0.0.2")

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
		{99999, uint16(utils.HTTPSPort), utils.TCPProto},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("port_%d", tc.port), func(t *testing.T) {
			gf := new(GenericFlow).Generate(srcIP, dstIP, tc.port, nil)
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

// ---------------------------------------------------------------------------
// GenericFlow timestamp fields are uint64
// ---------------------------------------------------------------------------

func TestGenericFlow_EpochMilliseconds(t *testing.T) {
	t.Parallel()
	srcIP := net.ParseIP("10.0.0.1")
	dstIP := net.ParseIP("10.0.0.2")

	gf := new(GenericFlow).Generate(srcIP, dstIP, utils.HTTPSPort, nil)

	nowMillis := uint64(time.Now().UnixMilli())
	if gf.FlowStartMillis == 0 {
		t.Error("FlowStartMillis should not be zero")
	}
	if gf.FlowEndMillis == 0 {
		t.Error("FlowEndMillis should not be zero")
	}
	if gf.FlowStartMillis > nowMillis+1000 || gf.FlowStartMillis < nowMillis-1000 {
		t.Errorf("FlowStartMillis %d is not close to current epoch millis %d",
			gf.FlowStartMillis, nowMillis)
	}
	if gf.FlowEndMillis < gf.FlowStartMillis {
		t.Errorf("FlowEndMillis %d < FlowStartMillis %d", gf.FlowEndMillis, gf.FlowStartMillis)
	}
}
