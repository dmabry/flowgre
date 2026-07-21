// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package ipfix

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/utils"
	"github.com/google/go-cmp/cmp"
)

func TestHeader_Generate(t *testing.T) {
	t.Parallel()
	sourceID := 618
	header := new(Header).Generate(sourceID, 42)

	if header.Version != 10 {
		t.Errorf("Header returned wrong version! Got: %d Want: 10", header.Version)
	}
	if header.ExportTime == 0 {
		t.Error("Header ExportTime should not be zero")
	}
	if header.SequenceNumber != 42 {
		t.Errorf("Header SequenceNumber wrong! Got: %d Want: 42", header.SequenceNumber)
	}
	if header.ObservationDomainId != uint32(sourceID) {
		t.Errorf("Header ObservationDomainId wrong! Got: %d Want: %d", header.ObservationDomainId, sourceID)
	}
	// Header is 16 bytes per RFC 7011
	if binary.Size(header) != 16 {
		t.Errorf("Header size wrong! Got: %d Want: 16", binary.Size(header))
	}
}

func TestGenerateTemplateIPFIX(t *testing.T) {
	t.Parallel()
	sourceID := 618
	seq := NewIPFIXSequence()
	flow := GenerateTemplateIPFIX(sourceID, seq)

	if len(flow.TemplateFlowSets) < 1 {
		t.Fatal("No template flowsets generated")
	}

	for _, tFlow := range flow.TemplateFlowSets {
		// RFC 7011: Template Sets use Set ID 2
		if tFlow.FlowSetID != SetIDTemplate {
			t.Errorf("FlowSetID wrong! Got: %d Want: %d", tFlow.FlowSetID, SetIDTemplate)
		}
		// Template: FlowSetID(2)+Length(2)+TemplateID(2)+FieldCount(2)+19*(Type(2)+Length(2))=84, no padding
		if tFlow.Length != 84 {
			t.Errorf("Template Length wrong! Got: %d Want: 84", tFlow.Length)
		}
		for _, template := range tFlow.Templates {
			if template.TemplateID != 256 {
				t.Errorf("TemplateID wrong! Got: %d Want: 256", template.TemplateID)
			}
			if template.FieldCount != 19 {
				t.Errorf("FieldCount wrong! Got: %d Want: 19", template.FieldCount)
			}
		}
	}
}

func TestGenerateDataIPFIX(t *testing.T) {
	t.Parallel()
	flowCount := 10
	sourceID := 618
	seq := NewIPFIXSequence()
	flow, err := GenerateDataIPFIX(flowCount, sourceID, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, seq)
	if err != nil {
		t.Fatal(err)
	}

	if len(flow.DataFlowSets) < 1 {
		t.Fatal("No data flowsets generated")
	}

	for _, dFlow := range flow.DataFlowSets {
		if dFlow.FlowSetID < 256 {
			t.Errorf("FlowSetID wrong! Got: %d Want >= 256", dFlow.FlowSetID)
		}
		if dFlow.Length < 64 {
			t.Errorf("Length wrong! Got: %d Want >= 64", dFlow.Length)
		}
		if len(dFlow.Items) < flowCount {
			t.Errorf("Items count wrong! Got: %d Want: %d", len(dFlow.Items), flowCount)
		}
	}
}

func TestGenerateIPFIX(t *testing.T) {
	t.Parallel()
	flowCount := 10
	sourceID := 618
	seq := NewIPFIXSequence()
	flow, err := GenerateIPFIX(flowCount, sourceID, "10.0.0.0/8", "10.0.0.0/8", seq)
	if err != nil {
		t.Fatal(err)
	}

	if flow.Header.Version != 10 {
		t.Errorf("Version wrong! Got: %d Want: 10", flow.Header.Version)
	}
	if len(flow.TemplateFlowSets) < 1 {
		t.Error("Should have template flowsets")
	}
	if len(flow.DataFlowSets) < 1 {
		t.Error("Should have data flowsets")
	}

	// Length is set during ToBytes; verify it's correct after serialization
	buf, _ := flow.ToBytes()
	parsedLen := binary.BigEndian.Uint16(buf.Bytes()[2:4])
	if parsedLen == 0 {
		t.Error("Header Length should not be zero after ToBytes")
	}
	if int(parsedLen) != buf.Len() {
		t.Errorf("Header Length %d does not match buffer length %d", parsedLen, buf.Len())
	}
}

func TestToBytes_RoundTrip(t *testing.T) {
	t.Parallel()
	sourceID := 618
	flowCount := 10
	seq := NewIPFIXSequence()

	tFlow := GenerateTemplateIPFIX(sourceID, seq)
	dFlow, err := GenerateDataIPFIX(flowCount, sourceID, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, seq)
	if err != nil {
		t.Fatal(err)
	}

	tBuf, _ := tFlow.ToBytes()
	dBuf, _ := dFlow.ToBytes()

	// Verify Bytes match generated IPFIX Data
	tLength := tBuf.Len()
	dLength := dBuf.Len()
	tread := make([]byte, tLength)
	tn, err := tBuf.Read(tread)
	if err != nil {
		t.Errorf("Error during IPFIX Template Read! Got: %v", err)
	}
	if tn != tLength {
		t.Errorf("Returned invalid IPFIX Template buffer length! Got: %d Want: %d", tn, tBuf.Len())
	}
	dread := make([]byte, dLength)
	dn, err := dBuf.Read(dread)
	if err != nil {
		t.Errorf("Error during IPFIX Data Read! Got: %v", err)
	}
	if dn != dLength {
		t.Errorf("Returned invalid IPFIX Data buffer length! Got: %d Want: %d", dn, dBuf.Len())
	}

	// Parse Template - RFC 7011 16-byte header
	tparsed := IPFIX{}
	treader := bytes.NewReader(tread)
	err = binary.Read(treader, binary.BigEndian, &tparsed.Header)
	if err != nil {
		t.Errorf("Failed to parse IPFIX Header! Got: %v", err)
	}
	if int(tparsed.Header.ObservationDomainId) != sourceID {
		t.Errorf("Failed to parse IPFIX Header Observation Domain ID! Got: %d Want: %d",
			int(tparsed.Header.ObservationDomainId), sourceID)
	}
	if tparsed.Header.Version != 10 {
		t.Errorf("Failed to parse IPFIX Header Version! Got: %d Want: 10", tparsed.Header.Version)
	}

	// Parse all FlowSets in the template packet
	for treader.Len() > 0 {
		var flowSetID uint16
		if err := binary.Read(treader, binary.BigEndian, &flowSetID); err != nil {
			break
		}
		var fsLength uint16
		if err := binary.Read(treader, binary.BigEndian, &fsLength); err != nil {
			break
		}
		remaining := int(fsLength) - 4

		var templateID uint16
		if err := binary.Read(treader, binary.BigEndian, &templateID); err != nil {
			break
		}
		remaining -= 2

		if templateID == 257 {
			// Options Template - RFC 7011 layout: TemplateID + FieldCount + ScopeFieldCount + fields
			var fieldCount uint16
			if err := binary.Read(treader, binary.BigEndian, &fieldCount); err != nil {
				break
			}
			remaining -= 2
			var scopeFieldCount uint16
			if err := binary.Read(treader, binary.BigEndian, &scopeFieldCount); err != nil {
				break
			}
			remaining -= 2
			fields := make([]Field, fieldCount)
			for i := range fieldCount {
				if err := binary.Read(treader, binary.BigEndian, &fields[i]); err != nil {
					break
				}
				remaining -= 4
			}
			tparsed.OptionsTemplateFlowSets = append(tparsed.OptionsTemplateFlowSets, OptionsTemplateFlowSet{
				FlowSetID: flowSetID,
				Length:    fsLength,
				Template: OptionsTemplate{
					TemplateID:      templateID,
					FieldCount:      fieldCount,
					ScopeFieldCount: scopeFieldCount,
					Fields:          fields,
				},
				Padding: remaining,
			})
			if remaining > 0 {
				treader.Seek(int64(remaining), 1)
			}
		} else {
			// Regular Template
			var fieldCount uint16
			if err := binary.Read(treader, binary.BigEndian, &fieldCount); err != nil {
				break
			}
			remaining -= 2
			fields := make([]Field, fieldCount)
			for i := range fieldCount {
				if err := binary.Read(treader, binary.BigEndian, &fields[i]); err != nil {
					break
				}
				remaining -= 4
			}
			tFlowSet := TemplateFlowSet{
				FlowSetID: flowSetID,
				Length:    fsLength,
				Templates: []Template{{
					TemplateID: templateID,
					FieldCount: fieldCount,
					Fields:     fields,
				}},
				Padding: remaining,
			}
			tparsed.TemplateFlowSets = append(tparsed.TemplateFlowSets, tFlowSet)
			if remaining > 0 {
				treader.Seek(int64(remaining), 1)
			}
		}
	}

	if !cmp.Equal(tFlow, tparsed) {
		t.Log("Generated IPFIX Template Flow and Parsed are different!")
		t.Logf("Original Header: %+v", tFlow.Header)
		t.Logf("Parsed Header: %+v", tparsed.Header)
		// Header.Length is set during ToBytes, so set it before comparing
		tFlow.Header.Length = tparsed.Header.Length
		if !cmp.Equal(tFlow, tparsed) {
			t.Error("Failed: Generated IPFIX Template Flow and Parsed are different (after Length fix)!")
		} else {
			t.Log("Generated IPFIX Template Flow and Parsed Match!")
		}
	} else {
		t.Log("Generated IPFIX Template Flow and Parsed Match!")
	}

	// Parse Data
	dparsed := IPFIX{}
	dreader := bytes.NewReader(dread)
	err = binary.Read(dreader, binary.BigEndian, &dparsed.Header)
	if err != nil {
		t.Errorf("Failed to parse IPFIX Data Header! Got: %v", err)
	}
	if int(dparsed.Header.ObservationDomainId) != sourceID {
		t.Errorf("Failed to parse IPFIX Data Header Observation Domain ID! Got: %d Want: %d",
			int(dparsed.Header.ObservationDomainId), sourceID)
	}
	if dparsed.Header.Version != 10 {
		t.Errorf("Failed to parse IPFIX Data Header Version! Got: %d Want: 10", dparsed.Header.Version)
	}

	dFlowSet := new(DataFlowSet)
	err = binary.Read(dreader, binary.BigEndian, &dFlowSet.FlowSetID)
	if err != nil {
		t.Errorf("Failed to parse IPFIX Data FlowSetID! Got: %v", err)
	}
	err = binary.Read(dreader, binary.BigEndian, &dFlowSet.Length)
	if err != nil {
		t.Errorf("Failed to parse IPFIX Data Length! Got: %v", err)
	}

	dataItems := make([]DataAny, flowCount)
	for i := range flowCount {
		dataItem := GenericFlow{}
		err := binary.Read(dreader, binary.BigEndian, &dataItem)
		if err != nil {
			t.Errorf("Issue reading IPFIX GenericFlow: %v", err)
		}
		dataItems[i] = dataItem
	}
	dFlowSet.Items = dataItems

	if dreader.Len() > 0 {
		padLength := dreader.Len()
		padding := make([]byte, padLength)
		err = binary.Read(dreader, binary.BigEndian, padding)
		if err != nil {
			t.Errorf("Failed to parse IPFIX Data Padding! Got: %v", err)
		}
		dFlowSet.Padding = padLength
	}

	dparsed.DataFlowSets = append(dparsed.DataFlowSets, *dFlowSet)

	if !cmp.Equal(dFlow, dparsed) {
		t.Log("Generated IPFIX Data Flow and Parsed are different!")
		// Header.Length is set during ToBytes, so set it before comparing
		dFlow.Header.Length = dparsed.Header.Length
		if !cmp.Equal(dFlow, dparsed) {
			t.Error("Failed: Generated IPFIX Data Flow and Parsed are different (after Length fix)!")
		} else {
			t.Log("Generated IPFIX Data Flow and Parsed Match!")
		}
	} else {
		t.Log("Generated IPFIX Data Flow and Parsed Match!")
	}
}

func TestIsValidIPFIX_AcceptVersion10(t *testing.T) {
	t.Parallel()
	seq := NewIPFIXSequence()
	flow := GenerateTemplateIPFIX(100, seq)
	buf, _ := flow.ToBytes()

	ok, err := IsValidIPFIX(buf.Bytes())
	if err != nil {
		t.Fatalf("IsValidIPFIX error: %v", err)
	}
	if !ok {
		t.Error("IsValidIPFIX should accept version 10")
	}
}

func TestIsValidIPFIX_RejectVersion9(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()
	nfFlow := netflow.GenerateTemplateNetflow(100, session)
	nfBuf := nfFlow.ToBytes()

	ok, err := IsValidIPFIX(nfBuf.Bytes())
	if ok {
		t.Error("IsValidIPFIX should reject version 9")
	}
	if err == nil {
		t.Error("IsValidIPFIX should return error for version 9")
	}
}

func TestUpdateTimeStamp(t *testing.T) {
	t.Parallel()
	seq := NewIPFIXSequence()
	flow := GenerateTemplateIPFIX(100, seq)
	buf, _ := flow.ToBytes()

	before := time.Now().Unix()
	time.Sleep(10 * time.Millisecond)

	updated, err := UpdateTimeStamp(buf.Bytes())
	if err != nil {
		t.Fatalf("UpdateTimeStamp error: %v", err)
	}

	var header Header
	if err := binary.Read(bytes.NewReader(updated), binary.BigEndian, &header); err != nil {
		t.Fatalf("Failed to parse updated header: %v", err)
	}

	if header.ExportTime < uint32(before) {
		t.Errorf("Updated timestamp should be >= current time. Got: %d Want: >= %d",
			header.ExportTime, before)
	}
	if header.Version != 10 {
		t.Errorf("Updated header version should still be 10. Got: %d", header.Version)
	}
}

func TestFlowSequenceMonotonicallyIncreases(t *testing.T) {
	t.Parallel()
	seq := NewIPFIXSequence()
	f1, err := GenerateDataIPFIX(1, 42, "10.0.0.0/8", "10.0.0.0/8", 443, seq)
	if err != nil {
		t.Fatal(err)
	}
	f2, err := GenerateDataIPFIX(1, 42, "10.0.0.0/8", "10.0.0.0/8", 443, seq)
	if err != nil {
		t.Fatal(err)
	}
	f3, err := GenerateDataIPFIX(1, 42, "10.0.0.0/8", "10.0.0.0/8", 443, seq)
	if err != nil {
		t.Fatal(err)
	}

	if f1.Header.SequenceNumber >= f2.Header.SequenceNumber {
		t.Errorf("SequenceNumber not monotonically increasing: f1=%d >= f2=%d",
			f1.Header.SequenceNumber, f2.Header.SequenceNumber)
	}
	if f2.Header.SequenceNumber >= f3.Header.SequenceNumber {
		t.Errorf("SequenceNumber not monotonically increasing: f2=%d >= f3=%d",
			f2.Header.SequenceNumber, f3.Header.SequenceNumber)
	}
}

func TestToBytes_BufferLengthMatchesFlowSetLengths(t *testing.T) {
	t.Parallel()
	sourceID := 618
	flowCount := 10
	seq := NewIPFIXSequence()

	// Test template-only packet
	tFlow := GenerateTemplateIPFIX(sourceID, seq)
	tBuf, _ := tFlow.ToBytes()
	expectedTLen := 16 // RFC 7011 header is 16 bytes
	for _, fs := range tFlow.TemplateFlowSets {
		expectedTLen += int(fs.Length)
	}
	for _, fs := range tFlow.OptionsTemplateFlowSets {
		expectedTLen += int(fs.Length)
	}
	if tBuf.Len() != expectedTLen {
		t.Errorf("Template buffer length mismatch: got %d, want %d (header %d + flowsets %d)",
			tBuf.Len(), expectedTLen, 16,
			expectedTLen-16)
	}

	// Test data-only packet
	dFlow, err := GenerateDataIPFIX(flowCount, sourceID, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, seq)
	if err != nil {
		t.Fatal(err)
	}
	dBuf, _ := dFlow.ToBytes()
	expectedDLen := 16
	for _, fs := range dFlow.DataFlowSets {
		expectedDLen += int(fs.Length)
	}
	if dBuf.Len() != expectedDLen {
		t.Errorf("Data buffer length mismatch: got %d, want %d (header %d + flowsets %d)",
			dBuf.Len(), expectedDLen, 16,
			expectedDLen-16)
	}

	// Test combined packet
	cFlow, err := GenerateIPFIX(flowCount, sourceID, "10.0.0.0/8", "10.0.0.0/8", seq)
	if err != nil {
		t.Fatal(err)
	}
	cBuf, _ := cFlow.ToBytes()
	expectedCLen := 16
	for _, fs := range cFlow.TemplateFlowSets {
		expectedCLen += int(fs.Length)
	}
	for _, fs := range cFlow.DataFlowSets {
		expectedCLen += int(fs.Length)
	}
	if cBuf.Len() != expectedCLen {
		t.Errorf("Combined buffer length mismatch: got %d, want %d",
			cBuf.Len(), expectedCLen)
	}
}

func TestGetTemplateFields_IPFIXFieldTypes(t *testing.T) {
	t.Parallel()
	gf := new(GenericFlow)
	fields := gf.GetTemplateFields()

	if len(fields) != 19 {
		t.Fatalf("Field count wrong! Got: %d Want: 19", len(fields))
	}

	// Expected IANA IPFIX field types (RFC 7011)
	expectedTypes := []uint16{
		1,   // octetDeltaCount
		23,  // postOctetDeltaCount
		2,   // packetDeltaCount
		24,  // postPacketDeltaCount
		8,   // sourceIPv4Address
		12,  // destinationIPv4Address
		27,  // sourceIPv6Address
		28,  // destinationIPv6Address
		29,  // sourceIPv6PrefixLength
		30,  // destinationIPv6PrefixLength
		7,   // sourceTransportPort
		11,  // destinationTransportPort
		4,   // protocolIdentifier
		6,   // tcpControlBits
		152, // flowStartMilliseconds
		153, // flowEndMilliseconds
		61,  // flowDirection
		5,   // ipClassOfService
		136, // flowEndReason
	}

	for i, expected := range expectedTypes {
		if fields[i].Type != expected {
			t.Errorf("Field[%d] type wrong! Got: %d Want: %d", i, fields[i].Type, expected)
		}
	}

	// Verify field lengths - timestamps are now 8 bytes
	expectedLengths := []uint16{4, 4, 4, 4, 4, 4, 16, 16, 1, 1, 2, 2, 1, 1, 8, 8, 1, 1, 1}
	for i, expected := range expectedLengths {
		if fields[i].Length != expected {
			t.Errorf("Field[%d] length wrong! Got: %d Want: %d", i, fields[i].Length, expected)
		}
	}
}

func TestGenericFlowIPv6(t *testing.T) {
	t.Parallel()

	srcIP := net.ParseIP("2001:db8::1")
	dstIP := net.ParseIP("2001:db8::2")

	result, err := new(GenericFlow).Generate(srcIP, dstIP, utils.HTTPSPort, nil)
	if err != nil {
		t.Fatalf("GenericFlow.Generate error: %v", err)
	}

	if result.SourceIPv4Addr != 0 {
		t.Errorf("expected zeroed IPv4 src, got %d", result.SourceIPv4Addr)
	}
	if result.DestIPv4Addr != 0 {
		t.Errorf("expected zeroed IPv4 dst, got %d", result.DestIPv4Addr)
	}

	expectedSrc := [16]byte{}
	copy(expectedSrc[:], srcIP.To16())
	if result.SourceIPv6Addr != expectedSrc {
		t.Errorf("IPv6 src mismatch: got %v want %v", result.SourceIPv6Addr, expectedSrc)
	}
	expectedDst := [16]byte{}
	copy(expectedDst[:], dstIP.To16())
	if result.DestIPv6Addr != expectedDst {
		t.Errorf("IPv6 dst mismatch: got %v want %v", result.DestIPv6Addr, expectedDst)
	}

	if result.SourceIPv6Prefix != 64 {
		t.Errorf("expected IPv6 src prefix 64, got %d", result.SourceIPv6Prefix)
	}
	if result.DestIPv6Prefix != 64 {
		t.Errorf("expected IPv6 dst prefix 64, got %d", result.DestIPv6Prefix)
	}
}

func TestGenericFlowIPv4_ZerosIPv6(t *testing.T) {
	t.Parallel()

	srcIP := net.ParseIP("10.0.0.1")
	dstIP := net.ParseIP("10.0.0.2")

	result, err := new(GenericFlow).Generate(srcIP, dstIP, utils.HTTPSPort, nil)
	if err != nil {
		t.Fatalf("GenericFlow.Generate error: %v", err)
	}

	if result.SourceIPv4Addr == 0 {
		t.Error("expected non-zero IPv4 src")
	}
	if result.DestIPv4Addr == 0 {
		t.Error("expected non-zero IPv4 dst")
	}

	if result.SourceIPv6Addr != [16]byte{} {
		t.Errorf("expected zeroed IPv6 src, got %v", result.SourceIPv6Addr)
	}
	if result.DestIPv6Addr != [16]byte{} {
		t.Errorf("expected zeroed IPv6 dst, got %v", result.DestIPv6Addr)
	}
	if result.SourceIPv6Prefix != 0 {
		t.Errorf("expected zeroed IPv6 src prefix, got %d", result.SourceIPv6Prefix)
	}
	if result.DestIPv6Prefix != 0 {
		t.Errorf("expected zeroed IPv6 dst prefix, got %d", result.DestIPv6Prefix)
	}
}

func TestGenerateDataIPFIX_IPv6(t *testing.T) {
	t.Parallel()
	flowCount := 10
	sourceID := 618
	seq := NewIPFIXSequence()
	flow, err := GenerateDataIPFIX(flowCount, sourceID, "2001:db8::/32", "2001:db8::/32", utils.HTTPSPort, seq)
	if err != nil {
		t.Fatal(err)
	}

	if len(flow.DataFlowSets) < 1 {
		t.Fatal("No data flowsets generated")
	}

	for _, dFlow := range flow.DataFlowSets {
		if dFlow.FlowSetID < 256 {
			t.Errorf("FlowSetID wrong! Got: %d Want >= 256", dFlow.FlowSetID)
		}
		if len(dFlow.Items) < flowCount {
			t.Errorf("Items count wrong! Got: %d Want: %d", len(dFlow.Items), flowCount)
		}

		for i, item := range dFlow.Items {
			gf, ok := item.(GenericFlow)
			if !ok {
				t.Fatalf("Item %d is not a GenericFlow", i)
			}
			if gf.SourceIPv6Addr == [16]byte{} {
				t.Errorf("Item %d: expected non-zero IPv6 src", i)
			}
			if gf.DestIPv6Addr == [16]byte{} {
				t.Errorf("Item %d: expected non-zero IPv6 dst", i)
			}
			if gf.SourceIPv4Addr != 0 {
				t.Errorf("Item %d: expected zeroed IPv4 src, got %d", i, gf.SourceIPv4Addr)
			}
			if gf.DestIPv4Addr != 0 {
				t.Errorf("Item %d: expected zeroed IPv4 dst, got %d", i, gf.DestIPv4Addr)
			}
		}
	}
}

func TestGenerateIPFIX_IPv6(t *testing.T) {
	t.Parallel()
	flowCount := 10
	sourceID := 618
	seq := NewIPFIXSequence()
	flow, err := GenerateIPFIX(flowCount, sourceID, "2001:db8::/32", "2001:db8::/32", seq)
	if err != nil {
		t.Fatal(err)
	}

	if flow.Header.Version != 10 {
		t.Errorf("Version wrong! Got: %d Want: 10", flow.Header.Version)
	}
	if len(flow.TemplateFlowSets) < 1 {
		t.Error("Should have template flowsets")
	}
	if len(flow.DataFlowSets) < 1 {
		t.Error("Should have data flowsets")
	}
}

func TestToBytes_IPv6_RoundTrip(t *testing.T) {
	t.Parallel()
	sourceID := 618
	flowCount := 10
	seq := NewIPFIXSequence()

	dFlow, err := GenerateDataIPFIX(flowCount, sourceID, "2001:db8::/32", "2001:db8::/32", utils.HTTPSPort, seq)
	if err != nil {
		t.Fatal(err)
	}
	dBuf, _ := dFlow.ToBytes()

	dLength := dBuf.Len()
	dread := make([]byte, dLength)
	dn, err := dBuf.Read(dread)
	if err != nil {
		t.Fatalf("Error during IPFIX Data Read! Got: %v", err)
	}
	if dn != dLength {
		t.Fatalf("Returned invalid IPFIX Data buffer length! Got: %d Want: %d", dn, dBuf.Len())
	}

	dreader := bytes.NewReader(dread)
	var parsedHeader Header
	if err := binary.Read(dreader, binary.BigEndian, &parsedHeader); err != nil {
		t.Fatalf("Failed to parse IPFIX Data Header: %v", err)
	}

	dFlowSet := new(DataFlowSet)
	if err := binary.Read(dreader, binary.BigEndian, &dFlowSet.FlowSetID); err != nil {
		t.Fatalf("Failed to parse IPFIX Data FlowSetID: %v", err)
	}
	if err := binary.Read(dreader, binary.BigEndian, &dFlowSet.Length); err != nil {
		t.Fatalf("Failed to parse IPFIX Data Length: %v", err)
	}

	dataItems := make([]DataAny, flowCount)
	for i := range flowCount {
		dataItem := GenericFlow{}
		if err := binary.Read(dreader, binary.BigEndian, &dataItem); err != nil {
			t.Fatalf("Issue reading IPFIX GenericFlow: %v", err)
		}
		if dataItem.SourceIPv6Addr == [16]byte{} {
			t.Errorf("Item %d: IPv6 src was zeroed after serialization", i)
		}
		if dataItem.DestIPv6Addr == [16]byte{} {
			t.Errorf("Item %d: IPv6 dst was zeroed after serialization", i)
		}
		if dataItem.SourceIPv6Prefix != 64 {
			t.Errorf("Item %d: expected IPv6 src prefix 64, got %d", i, dataItem.SourceIPv6Prefix)
		}
		dataItems[i] = dataItem
	}
	dFlowSet.Items = dataItems

	if dreader.Len() > 0 {
		padLength := dreader.Len()
		padding := make([]byte, padLength)
		if err := binary.Read(dreader, binary.BigEndian, padding); err != nil {
			t.Errorf("Failed to parse IPFIX Data Padding: %v", err)
		}
		dFlowSet.Padding = padLength
	}
}

func TestToBytes_IPv6_BufferLengthMatchesFlowSetLengths(t *testing.T) {
	t.Parallel()
	sourceID := 618
	flowCount := 10
	seq := NewIPFIXSequence()

	dFlow, err := GenerateDataIPFIX(flowCount, sourceID, "2001:db8::/32", "2001:db8::/32", utils.HTTPSPort, seq)
	if err != nil {
		t.Fatal(err)
	}
	dBuf, _ := dFlow.ToBytes()
	expectedDLen := 16
	for _, fs := range dFlow.DataFlowSets {
		expectedDLen += int(fs.Length)
	}
	if dBuf.Len() != expectedDLen {
		t.Errorf("IPv6 Data buffer length mismatch: got %d, want %d (header %d + flowsets %d)",
			dBuf.Len(), expectedDLen, 16,
			expectedDLen-16)
	}

	cFlow, err := GenerateIPFIX(flowCount, sourceID, "2001:db8::/32", "2001:db8::/32", seq)
	if err != nil {
		t.Fatal(err)
	}
	cBuf, _ := cFlow.ToBytes()
	expectedCLen := 16
	for _, fs := range cFlow.TemplateFlowSets {
		expectedCLen += int(fs.Length)
	}
	for _, fs := range cFlow.DataFlowSets {
		expectedCLen += int(fs.Length)
	}
	if cBuf.Len() != expectedCLen {
		t.Errorf("IPv6 Combined buffer length mismatch: got %d, want %d",
			cBuf.Len(), expectedCLen)
	}
}

func TestHeader_Length_SetCorrectly(t *testing.T) {
	t.Parallel()
	seq := NewIPFIXSequence()
	flow, err := GenerateDataIPFIX(5, 42, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, seq)
	if err != nil {
		t.Fatal(err)
	}
	buf, _ := flow.ToBytes()
	payload := buf.Bytes()

	declaredLength := binary.BigEndian.Uint16(payload[2:4])
	if int(declaredLength) != len(payload) {
		t.Errorf("Header Length %d does not match payload length %d", declaredLength, len(payload))
	}
}

func TestSequenceNumber_OnlyDataRecords(t *testing.T) {
	t.Parallel()
	seq := NewIPFIXSequence()

	// Template-only messages should have sequence number 0
	tpl := GenerateTemplateIPFIX(42, seq)
	if tpl.Header.SequenceNumber != 0 {
		t.Errorf("Template-only message should have sequence 0, got %d", tpl.Header.SequenceNumber)
	}

	// Data messages should have increasing sequence numbers
	d1, err := GenerateDataIPFIX(1, 42, "10.0.0.0/8", "10.0.0.0/8", 443, seq)
	if err != nil {
		t.Fatal(err)
	}
	// d1 uses seq 1 and advances seq by 1 (1 record)

	d2, err := GenerateDataIPFIX(3, 42, "10.0.0.0/8", "10.0.0.0/8", 443, seq)
	if err != nil {
		t.Fatal(err)
	}
	// d2 uses seq 2 and advances seq by 3 (3 records)

	// d2 should be 1 ahead of d1 (seq was at 2 when d2 was created)
	if d2.Header.SequenceNumber != d1.Header.SequenceNumber+1 {
		t.Errorf("Expected sequence to advance by 1, got d1=%d, d2=%d",
			d1.Header.SequenceNumber, d2.Header.SequenceNumber)
	}

	// After d1 (1 record) and d2 (3 records), seq counter should be at 4
	// Reserve(1) should return 4 and advance to 5
	nextSeq := seq.Reserve(1)
	if nextSeq != 4 {
		t.Errorf("Expected next sequence to be 4, got %d", nextSeq)
	}
}

func TestHeader_16Bytes(t *testing.T) {
	t.Parallel()
	h := Header{
		Version:             10,
		Length:              100,
		ExportTime:          1234567890,
		SequenceNumber:      42,
		ObservationDomainId: 618,
	}
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, h)
	if buf.Len() != 16 {
		t.Errorf("Header serializes to %d bytes, expected 16", buf.Len())
	}
}

// Golden wire-format fixture tests verify exact byte layouts against RFC 7011.

func TestGolden_HeaderOnlyMessage(t *testing.T) {
	t.Parallel()
	// RFC 7011 permits a 16-byte header-only message (zero Sets).
	payload := make([]byte, 16)
	binary.BigEndian.PutUint16(payload[0:2], 10)   // Version
	binary.BigEndian.PutUint16(payload[2:4], 16)   // Length
	binary.BigEndian.PutUint32(payload[4:8], 1000) // ExportTime
	binary.BigEndian.PutUint32(payload[8:12], 0)   // SequenceNumber
	binary.BigEndian.PutUint32(payload[12:16], 42) // ObservationDomainId

	ok, err := IsValidIPFIX(payload)
	if err != nil {
		t.Fatalf("IsValidIPFIX error: %v", err)
	}
	if !ok {
		t.Error("IsValidIPFIX should accept header-only message")
	}
}

func TestGolden_TemplateWithdrawal(t *testing.T) {
	t.Parallel()
	// RFC 7011 §8.1: Template withdrawal is a Template Record with
	// Field Count of 0. The Set contains only the FlowSet header (4 bytes)
	// plus the withdrawal record (4 bytes = TemplateID + FieldCount).
	payload := make([]byte, 24)
	// Header
	binary.BigEndian.PutUint16(payload[0:2], 10)   // Version
	binary.BigEndian.PutUint16(payload[2:4], 24)   // Length
	binary.BigEndian.PutUint32(payload[4:8], 1000) // ExportTime
	binary.BigEndian.PutUint32(payload[8:12], 0)   // SequenceNumber
	binary.BigEndian.PutUint32(payload[12:16], 42) // ObservationDomainId
	// Template Set (Set ID = 2, Length = 8)
	binary.BigEndian.PutUint16(payload[16:18], 2) // FlowSetID
	binary.BigEndian.PutUint16(payload[18:20], 8) // Set Length
	// Template withdrawal record
	binary.BigEndian.PutUint16(payload[20:22], 256) // TemplateID
	binary.BigEndian.PutUint16(payload[22:24], 0)   // FieldCount (0 = withdrawal)

	ok, err := IsValidIPFIX(payload)
	if err != nil {
		t.Fatalf("IsValidIPFIX error: %v", err)
	}
	if !ok {
		t.Error("IsValidIPFIX should accept template withdrawal")
	}
}

func TestGolden_OptionsTemplateWithdrawal(t *testing.T) {
	t.Parallel()
	// RFC 7011 §8.1: Options Template withdrawal is 4 bytes:
	// TemplateID(2) + FieldCount(2, value 0). No Scope Field Count field.
	// Set = FlowSet header(4) + withdrawal record(4) = 8 bytes.
	// Message = Header(16) + Set(8) = 24 bytes.
	payload := make([]byte, 24)
	// Header
	binary.BigEndian.PutUint16(payload[0:2], 10)   // Version
	binary.BigEndian.PutUint16(payload[2:4], 24)   // Length
	binary.BigEndian.PutUint32(payload[4:8], 1000) // ExportTime
	binary.BigEndian.PutUint32(payload[8:12], 0)   // SequenceNumber
	binary.BigEndian.PutUint32(payload[12:16], 42) // ObservationDomainId
	// Options Template Set (Set ID = 3, Length = 8)
	binary.BigEndian.PutUint16(payload[16:18], 3) // FlowSetID
	binary.BigEndian.PutUint16(payload[18:20], 8) // Set Length
	// Options Template withdrawal record (4 bytes, RFC 7011 §8.1)
	binary.BigEndian.PutUint16(payload[20:22], 257) // TemplateID
	binary.BigEndian.PutUint16(payload[22:24], 0)   // FieldCount (0 = withdrawal)

	ok, err := IsValidIPFIX(payload)
	if err != nil {
		t.Fatalf("IsValidIPFIX error: %v", err)
	}
	if !ok {
		t.Error("IsValidIPFIX should accept options template withdrawal")
	}
}

func TestGolden_RejectOptionsTemplateZeroScopeFields(t *testing.T) {
	t.Parallel()
	// RFC 7011 §3.4.2.2: Normal Options Templates must have at least one
	// scope field (Scope Field Count >= 1). Only withdrawals have Field Count 0.
	// Layout: Header(16) + SetHeader(4) + TemplateID(2) + FieldCount(2) +
	//         ScopeFieldCount(2) + fieldSpecifier(4) + padding(2) = 32 bytes.
	payload := make([]byte, 32)
	binary.BigEndian.PutUint16(payload[0:2], 10)   // Version
	binary.BigEndian.PutUint16(payload[2:4], 32)   // Length
	binary.BigEndian.PutUint32(payload[4:8], 1000) // ExportTime
	binary.BigEndian.PutUint32(payload[8:12], 0)   // SequenceNumber
	binary.BigEndian.PutUint32(payload[12:16], 42) // ObservationDomainId
	// Options Template Set (Set ID = 3, Length = 16)
	binary.BigEndian.PutUint16(payload[16:18], 3)  // FlowSetID
	binary.BigEndian.PutUint16(payload[18:20], 16) // Set Length
	// Options Template record with Scope Field Count = 0 (invalid)
	binary.BigEndian.PutUint16(payload[20:22], 257) // TemplateID
	binary.BigEndian.PutUint16(payload[22:24], 1)   // FieldCount (1, not withdrawal)
	binary.BigEndian.PutUint16(payload[24:26], 0)   // ScopeFieldCount (0 = invalid)
	binary.BigEndian.PutUint16(payload[26:28], 149) // ElementID
	binary.BigEndian.PutUint16(payload[28:30], 4)   // Length
	// Padding (2 bytes)

	ok, err := IsValidIPFIX(payload)
	if ok {
		t.Error("IsValidIPFIX should reject Options Template with Scope Field Count 0")
	}
	if err == nil {
		t.Error("IsValidIPFIX should return error for Scope Field Count 0")
	}
}

func TestGolden_TemplateSetByteLayout(t *testing.T) {
	t.Parallel()
	// Verify exact byte layout of a Template Set with one 2-field template.
	// Fields: octetDeltaCount(1, len=4), packetDeltaCount(2, len=4).
	// Record: TemplateID(2) + FieldCount(2) + 2*FieldSpecifier(4) = 12 bytes.
	// Set: FlowSetID(2) + Length(2) + record(12) = 16 bytes.
	// Message: Header(16) + Set(16) = 32 bytes.
	payload := make([]byte, 32)
	// Header
	binary.BigEndian.PutUint16(payload[0:2], 10)   // Version
	binary.BigEndian.PutUint16(payload[2:4], 32)   // Length
	binary.BigEndian.PutUint32(payload[4:8], 1000) // ExportTime
	binary.BigEndian.PutUint32(payload[8:12], 0)   // SequenceNumber
	binary.BigEndian.PutUint32(payload[12:16], 42) // ObservationDomainId
	// Template Set
	binary.BigEndian.PutUint16(payload[16:18], 2)  // FlowSetID
	binary.BigEndian.PutUint16(payload[18:20], 16) // Set Length
	// Template record
	binary.BigEndian.PutUint16(payload[20:22], 256) // TemplateID
	binary.BigEndian.PutUint16(payload[22:24], 2)   // FieldCount
	// Field 1: octetDeltaCount
	binary.BigEndian.PutUint16(payload[24:26], 1) // ElementID
	binary.BigEndian.PutUint16(payload[26:28], 4) // Length
	// Field 2: packetDeltaCount
	binary.BigEndian.PutUint16(payload[28:30], 2) // ElementID
	binary.BigEndian.PutUint16(payload[30:32], 4) // Length

	ok, err := IsValidIPFIX(payload)
	if err != nil {
		t.Fatalf("IsValidIPFIX error: %v", err)
	}
	if !ok {
		t.Error("IsValidIPFIX should accept valid template set")
	}
}

func TestGolden_DataSetByteLayout(t *testing.T) {
	t.Parallel()
	// Verify exact byte layout of a Data Set with one record.
	// Template 256 defines: octetDeltaCount(4) + packetDeltaCount(4) = 8 bytes.
	// Set: FlowSetID(2) + Length(2) + record(8) = 12 bytes.
	// Message: Header(16) + Set(12) = 28 bytes.
	payload := make([]byte, 28)
	// Header
	binary.BigEndian.PutUint16(payload[0:2], 10)   // Version
	binary.BigEndian.PutUint16(payload[2:4], 28)   // Length
	binary.BigEndian.PutUint32(payload[4:8], 1000) // ExportTime
	binary.BigEndian.PutUint32(payload[8:12], 5)   // SequenceNumber
	binary.BigEndian.PutUint32(payload[12:16], 42) // ObservationDomainId
	// Data Set (ID = TemplateID = 256, Length = 12)
	binary.BigEndian.PutUint16(payload[16:18], 256) // FlowSetID
	binary.BigEndian.PutUint16(payload[18:20], 12)  // Set Length
	// Data record: octetDeltaCount(4) + packetDeltaCount(4)
	binary.BigEndian.PutUint32(payload[20:24], 1000) // octetDeltaCount
	binary.BigEndian.PutUint32(payload[24:28], 10)   // packetDeltaCount

	ok, err := IsValidIPFIX(payload)
	if err != nil {
		t.Fatalf("IsValidIPFIX error: %v", err)
	}
	if !ok {
		t.Error("IsValidIPFIX should accept valid data set")
	}
}

func TestGolden_RejectReservedSetID(t *testing.T) {
	t.Parallel()
	payload := make([]byte, 20)
	binary.BigEndian.PutUint16(payload[0:2], 10)   // Version
	binary.BigEndian.PutUint16(payload[2:4], 20)   // Length
	binary.BigEndian.PutUint32(payload[4:8], 1000) // ExportTime
	binary.BigEndian.PutUint32(payload[8:12], 0)   // SequenceNumber
	binary.BigEndian.PutUint32(payload[12:16], 42) // ObservationDomainId
	// Set ID 0 (reserved)
	binary.BigEndian.PutUint16(payload[16:18], 0) // FlowSetID
	binary.BigEndian.PutUint16(payload[18:20], 4) // Set Length

	ok, err := IsValidIPFIX(payload)
	if ok {
		t.Error("IsValidIPFIX should reject reserved Set ID 0")
	}
	if err == nil {
		t.Error("IsValidIPFIX should return error for reserved Set ID 0")
	}
}

func TestGolden_RejectUnassignedSetID(t *testing.T) {
	t.Parallel()
	payload := make([]byte, 20)
	binary.BigEndian.PutUint16(payload[0:2], 10)   // Version
	binary.BigEndian.PutUint16(payload[2:4], 20)   // Length
	binary.BigEndian.PutUint32(payload[4:8], 1000) // ExportTime
	binary.BigEndian.PutUint32(payload[8:12], 0)   // SequenceNumber
	binary.BigEndian.PutUint32(payload[12:16], 42) // ObservationDomainId
	// Set ID 100 (unassigned, range 4-255)
	binary.BigEndian.PutUint16(payload[16:18], 100) // FlowSetID
	binary.BigEndian.PutUint16(payload[18:20], 4)   // Set Length

	ok, err := IsValidIPFIX(payload)
	if ok {
		t.Error("IsValidIPFIX should reject unassigned Set ID 100")
	}
	if err == nil {
		t.Error("IsValidIPFIX should return error for unassigned Set ID 100")
	}
}

func TestGolden_RejectMismatchedMessageLength(t *testing.T) {
	t.Parallel()
	// Header says 20 bytes but payload is only 16.
	payload := make([]byte, 16)
	binary.BigEndian.PutUint16(payload[0:2], 10)   // Version
	binary.BigEndian.PutUint16(payload[2:4], 20)   // Length (wrong)
	binary.BigEndian.PutUint32(payload[4:8], 1000) // ExportTime
	binary.BigEndian.PutUint32(payload[8:12], 0)   // SequenceNumber
	binary.BigEndian.PutUint32(payload[12:16], 42) // ObservationDomainId

	ok, err := IsValidIPFIX(payload)
	if ok {
		t.Error("IsValidIPFIX should reject mismatched message length")
	}
	if err == nil {
		t.Error("IsValidIPFIX should return error for mismatched message length")
	}
}

func TestOversizedPacket_NoSequenceConsumed(t *testing.T) {
	t.Parallel()
	// A DataFlowSet that exceeds 65535 bytes must fail during Generate,
	// before any sequence numbers are reserved.
	seq := NewIPFIXSequence()

	// GenericFlow is 76 bytes. 790 records * 76 = 60040 + 4 header = 60044.
	// But with the full record size, we need to exceed 65535.
	// 65535 - 16(header) = 65519 available for sets.
	// 65519 - 4(set header) = 65515 for records.
	// 65515 / 76 = 862 records max. So 863+ records should overflow.
	// Use a large number to guarantee overflow.
	flow, err := GenerateDataIPFIX(1000, 42, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, seq)
	if err == nil {
		t.Fatal("Expected error for oversized packet, got nil")
	}

	// Sequence counter must not have advanced
	if seq.Current() != 0 {
		t.Errorf("Sequence counter advanced to %d despite oversized packet failure", seq.Current())
	}
	_ = flow
}

func TestTemplateRetransmission_SameSequenceNumber(t *testing.T) {
	t.Parallel()
	// Template-only messages must not advance the sequence number.
	seq := NewIPFIXSequence()

	t1 := GenerateTemplateIPFIX(42, seq)
	t2 := GenerateTemplateIPFIX(42, seq)
	t3 := GenerateTemplateIPFIX(42, seq)

	if t1.Header.SequenceNumber != 0 {
		t.Errorf("t1 sequence should be 0, got %d", t1.Header.SequenceNumber)
	}
	if t2.Header.SequenceNumber != 0 {
		t.Errorf("t2 sequence should be 0, got %d", t2.Header.SequenceNumber)
	}
	if t3.Header.SequenceNumber != 0 {
		t.Errorf("t3 sequence should be 0, got %d", t3.Header.SequenceNumber)
	}
	if seq.Current() != 0 {
		t.Errorf("Sequence counter should be 0 after template messages, got %d", seq.Current())
	}
}

func TestOptionsDataRecord_AdvancesSequence(t *testing.T) {
	t.Parallel()
	// Options Data Records are Data Records and must advance the sequence number.
	seq := NewIPFIXSequence()

	od1 := GenerateOptionsDataIPFIX(42, seq)
	od2 := GenerateOptionsDataIPFIX(42, seq)

	if od1.Header.SequenceNumber != 0 {
		t.Errorf("od1 sequence should be 0, got %d", od1.Header.SequenceNumber)
	}
	if od2.Header.SequenceNumber != 1 {
		t.Errorf("od2 sequence should be 1, got %d", od2.Header.SequenceNumber)
	}
	if seq.Current() != 2 {
		t.Errorf("Sequence counter should be 2 after 2 options data records, got %d", seq.Current())
	}
}
