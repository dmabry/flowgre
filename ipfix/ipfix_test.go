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
	flowCount := 10
	sourceID := 618
	session := netflow.NewSession()
	header := new(Header).Generate(flowCount, sourceID, session)

	if header.Version != 10 {
		t.Errorf("Header returned wrong version! Got: %d Want: 10", header.Version)
	}
	if header.SysUptime == 0 {
		t.Error("Header SysUptime should not be zero")
	}
	if header.FlowCount != uint16(flowCount) {
		t.Errorf("Header FlowCount wrong! Got: %d Want: %d", header.FlowCount, flowCount)
	}
	if header.UnixSec == 0 {
		t.Error("Header UnixSec should not be zero")
	}
	if header.FlowSequence == 0 {
		t.Error("Header FlowSequence should not be zero")
	}
	if header.SourceID != uint32(sourceID) {
		t.Errorf("Header SourceID wrong! Got: %d Want: %d", header.SourceID, sourceID)
	}
}

func TestGenerateTemplateIPFIX(t *testing.T) {
	t.Parallel()
	sourceID := 618
	session := netflow.NewSession()
	flow := GenerateTemplateIPFIX(sourceID, session)

	if len(flow.TemplateFlowSets) < 1 {
		t.Fatal("No template flowsets generated")
	}

	for _, tFlow := range flow.TemplateFlowSets {
		if tFlow.FlowSetID != 0 {
			t.Errorf("FlowSetID wrong! Got: %d Want: 0", tFlow.FlowSetID)
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
	session := netflow.NewSession()
	flow, err := GenerateDataIPFIX(flowCount, sourceID, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, session)
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
	session := netflow.NewSession()
	flow, err := GenerateIPFIX(flowCount, sourceID, "10.0.0.0/8", "10.0.0.0/8", session)
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
	if flow.Header.FlowCount != uint16(flowCount+1) {
		t.Errorf("FlowCount wrong! Got: %d Want: %d", flow.Header.FlowCount, flowCount+1)
	}
}

func TestToBytes_RoundTrip(t *testing.T) {
	t.Parallel()
	sourceID := 618
	flowCount := 10
	session := netflow.NewSession()

	tFlow := GenerateTemplateIPFIX(sourceID, session)
	dFlow, err := GenerateDataIPFIX(flowCount, sourceID, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, session)
	if err != nil {
		t.Fatal(err)
	}

	tBuf := tFlow.ToBytes()
	dBuf := dFlow.ToBytes()

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

	// Parse Template
	tparsed := IPFIX{}
	treader := bytes.NewReader(tread)
	err = binary.Read(treader, binary.BigEndian, &tparsed.Header)
	if err != nil {
		t.Errorf("Failed to parse IPFIX Header! Got: %v", err)
	}
	if int(tparsed.Header.SourceID) != sourceID {
		t.Errorf("Failed to parse IPFIX Header Source ID! Got: %d Want: %d",
			int(tparsed.Header.SourceID), sourceID)
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
		if flowSetID != 0 {
			// Not a template FlowSet, skip
			break
		}
		var fsLength uint16
		if err := binary.Read(treader, binary.BigEndian, &fsLength); err != nil {
			break
		}
		remaining := int(fsLength) - 4 // subtract FlowSetID + Length

		// Check if this is an Options Template (TemplateID >= 256 with ScopeFieldCount)
		var templateID uint16
		if err := binary.Read(treader, binary.BigEndian, &templateID); err != nil {
			break
		}
		remaining -= 2

		if templateID == 257 {
			// Options Template - parse it
			var scopeFieldCount uint16
			if err := binary.Read(treader, binary.BigEndian, &scopeFieldCount); err != nil {
				break
			}
			remaining -= 2
			scopeFields := make([]Field, scopeFieldCount)
			for i := range scopeFieldCount {
				if err := binary.Read(treader, binary.BigEndian, &scopeFields[i]); err != nil {
					break
				}
				remaining -= 4
			}
			var dataFieldCount uint16
			if err := binary.Read(treader, binary.BigEndian, &dataFieldCount); err != nil {
				break
			}
			remaining -= 2
			dataFields := make([]Field, dataFieldCount)
			for i := range dataFieldCount {
				if err := binary.Read(treader, binary.BigEndian, &dataFields[i]); err != nil {
					break
				}
				remaining -= 4
			}
			tparsed.OptionsTemplateFlowSets = append(tparsed.OptionsTemplateFlowSets, OptionsTemplateFlowSet{
				FlowSetID: flowSetID,
				Length:    fsLength,
				Template: OptionsTemplate{
					TemplateID:      templateID,
					ScopeFieldCount: scopeFieldCount,
					ScopeFields:     scopeFields,
					DataFieldCount:  dataFieldCount,
					DataFields:      dataFields,
				},
				Padding: remaining,
			})
			// Skip padding
			if remaining > 0 {
				treader.Seek(int64(remaining), 1)
			}
		} else {
			// Regular Data Template
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
			// Skip padding
			if remaining > 0 {
				treader.Seek(int64(remaining), 1)
			}
		}
	}

	if !cmp.Equal(tFlow, tparsed) {
		t.Error("Failed: Generated IPFIX Template Flow and Parsed are different!")
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
	if int(dparsed.Header.SourceID) != sourceID {
		t.Errorf("Failed to parse IPFIX Data Header Source ID! Got: %d Want: %d",
			int(dparsed.Header.SourceID), sourceID)
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
		t.Error("Failed: Generated IPFIX Data Flow and Parsed are different!")
	} else {
		t.Log("Generated IPFIX Data Flow and Parsed Match!")
	}
}

func TestIsValidIPFIX_AcceptVersion10(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()
	flow := GenerateTemplateIPFIX(100, session)
	buf := flow.ToBytes()

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
	session := netflow.NewSession()
	flow := GenerateTemplateIPFIX(100, session)
	buf := flow.ToBytes()

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

	if header.UnixSec < uint32(before) {
		t.Errorf("Updated timestamp should be >= current time. Got: %d Want: >= %d",
			header.UnixSec, before)
	}
	if header.Version != 10 {
		t.Errorf("Updated header version should still be 10. Got: %d", header.Version)
	}
}

func TestFlowSequenceMonotonicallyIncreases(t *testing.T) {
	t.Parallel()
	session := netflow.NewSession()
	f1, err := GenerateDataIPFIX(1, 42, "10.0.0.0/8", "10.0.0.0/8", 443, session)
	if err != nil {
		t.Fatal(err)
	}
	f2, err := GenerateDataIPFIX(1, 42, "10.0.0.0/8", "10.0.0.0/8", 443, session)
	if err != nil {
		t.Fatal(err)
	}
	f3, err := GenerateDataIPFIX(1, 42, "10.0.0.0/8", "10.0.0.0/8", 443, session)
	if err != nil {
		t.Fatal(err)
	}

	if f1.Header.FlowSequence >= f2.Header.FlowSequence {
		t.Errorf("FlowSequence not monotonically increasing: f1=%d >= f2=%d",
			f1.Header.FlowSequence, f2.Header.FlowSequence)
	}
	if f2.Header.FlowSequence >= f3.Header.FlowSequence {
		t.Errorf("FlowSequence not monotonically increasing: f2=%d >= f3=%d",
			f2.Header.FlowSequence, f3.Header.FlowSequence)
	}
}

func TestToBytes_BufferLengthMatchesFlowSetLengths(t *testing.T) {
	t.Parallel()
	sourceID := 618
	flowCount := 10
	session := netflow.NewSession()

	// Test template-only packet
	tFlow := GenerateTemplateIPFIX(sourceID, session)
	tBuf := tFlow.ToBytes()
	expectedTLen := binary.Size(tFlow.Header)
	for _, fs := range tFlow.TemplateFlowSets {
		expectedTLen += int(fs.Length)
	}
	for _, fs := range tFlow.OptionsTemplateFlowSets {
		expectedTLen += int(fs.Length)
	}
	if tBuf.Len() != expectedTLen {
		t.Errorf("Template buffer length mismatch: got %d, want %d (header %d + flowsets %d)",
			tBuf.Len(), expectedTLen, binary.Size(tFlow.Header),
			expectedTLen-binary.Size(tFlow.Header))
	}

	// Test data-only packet
	dFlow, err := GenerateDataIPFIX(flowCount, sourceID, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, session)
	if err != nil {
		t.Fatal(err)
	}
	dBuf := dFlow.ToBytes()
	expectedDLen := binary.Size(dFlow.Header)
	for _, fs := range dFlow.DataFlowSets {
		expectedDLen += int(fs.Length)
	}
	if dBuf.Len() != expectedDLen {
		t.Errorf("Data buffer length mismatch: got %d, want %d (header %d + flowsets %d)",
			dBuf.Len(), expectedDLen, binary.Size(dFlow.Header),
			expectedDLen-binary.Size(dFlow.Header))
	}

	// Test combined packet
	cFlow, err := GenerateIPFIX(flowCount, sourceID, "10.0.0.0/8", "10.0.0.0/8", session)
	if err != nil {
		t.Fatal(err)
	}
	cBuf := cFlow.ToBytes()
	expectedCLen := binary.Size(cFlow.Header)
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

	// Expected IPFIX field types (IANA IPFIX Information Model RFC 7011)
	expectedTypes := []uint16{
		1026, // inOctets
		1028, // outOctets
		1025, // inPackets
		1027, // outPackets
		8,    // sourceIPv4Address
		12,   // destinationIPv4Address
		25,   // sourceIPv6Address
		26,   // destinationIPv6Address
		47,   // sourceIPv6PrefixLength
		48,   // destinationIPv6PrefixLength
		7,    // sourceTransportPort
		11,   // destinationTransportPort
		4,    // protocolIdentifier
		6,    // tcpFlags
		152,  // flowStartMilliseconds
		153,  // flowEndMilliseconds
		1024, // flowDirection
		3,    // ipClassOfService
		157,  // flowEndReason
	}

	for i, expected := range expectedTypes {
		if fields[i].Type != expected {
			t.Errorf("Field[%d] type wrong! Got: %d Want: %d", i, fields[i].Type, expected)
		}
	}

	// Verify field lengths match expected sizes
	expectedLengths := []uint16{4, 4, 4, 4, 4, 4, 16, 16, 1, 1, 2, 2, 1, 1, 4, 4, 1, 1, 1}
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
	session := netflow.NewSession()

	result := new(GenericFlow).Generate(srcIP, dstIP, utils.HTTPSPort, session)

	// IPv4 fields should be zeroed
	if result.SourceIPv4Addr != 0 {
		t.Errorf("expected zeroed IPv4 src, got %d", result.SourceIPv4Addr)
	}
	if result.DestIPv4Addr != 0 {
		t.Errorf("expected zeroed IPv4 dst, got %d", result.DestIPv4Addr)
	}

	// IPv6 fields should be populated
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

	// Prefix should be 64
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
	session := netflow.NewSession()

	result := new(GenericFlow).Generate(srcIP, dstIP, utils.HTTPSPort, session)

	// IPv4 fields should be populated
	if result.SourceIPv4Addr == 0 {
		t.Error("expected non-zero IPv4 src")
	}
	if result.DestIPv4Addr == 0 {
		t.Error("expected non-zero IPv4 dst")
	}

	// IPv6 fields should be zeroed
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
	session := netflow.NewSession()
	flow, err := GenerateDataIPFIX(flowCount, sourceID, "2001:db8::/32", "2001:db8::/32", utils.HTTPSPort, session)
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

		// Verify each item has IPv6 addresses populated
		for i, item := range dFlow.Items {
			gf, ok := item.(GenericFlow)
			if !ok {
				t.Fatalf("Item %d is not a GenericFlow", i)
			}
			// IPv6 addresses should be populated (non-zero)
			if gf.SourceIPv6Addr == [16]byte{} {
				t.Errorf("Item %d: expected non-zero IPv6 src", i)
			}
			if gf.DestIPv6Addr == [16]byte{} {
				t.Errorf("Item %d: expected non-zero IPv6 dst", i)
			}
			// IPv4 addresses should be zeroed
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
	session := netflow.NewSession()
	flow, err := GenerateIPFIX(flowCount, sourceID, "2001:db8::/32", "2001:db8::/32", session)
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
	if flow.Header.FlowCount != uint16(flowCount+1) {
		t.Errorf("FlowCount wrong! Got: %d Want: %d", flow.Header.FlowCount, flowCount+1)
	}
}

func TestToBytes_IPv6_RoundTrip(t *testing.T) {
	t.Parallel()
	sourceID := 618
	flowCount := 10
	session := netflow.NewSession()

	dFlow, err := GenerateDataIPFIX(flowCount, sourceID, "2001:db8::/32", "2001:db8::/32", utils.HTTPSPort, session)
	if err != nil {
		t.Fatal(err)
	}
	dBuf := dFlow.ToBytes()

	// Read back
	dLength := dBuf.Len()
	dread := make([]byte, dLength)
	dn, err := dBuf.Read(dread)
	if err != nil {
		t.Fatalf("Error during IPFIX Data Read! Got: %v", err)
	}
	if dn != dLength {
		t.Fatalf("Returned invalid IPFIX Data buffer length! Got: %d Want: %d", dn, dBuf.Len())
	}

	// Parse Data — skip the IPFIX header first
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
		// Verify IPv6 addresses survived serialization
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
	session := netflow.NewSession()

	dFlow, err := GenerateDataIPFIX(flowCount, sourceID, "2001:db8::/32", "2001:db8::/32", utils.HTTPSPort, session)
	if err != nil {
		t.Fatal(err)
	}
	dBuf := dFlow.ToBytes()
	expectedDLen := binary.Size(dFlow.Header)
	for _, fs := range dFlow.DataFlowSets {
		expectedDLen += int(fs.Length)
	}
	if dBuf.Len() != expectedDLen {
		t.Errorf("IPv6 Data buffer length mismatch: got %d, want %d (header %d + flowsets %d)",
			dBuf.Len(), expectedDLen, binary.Size(dFlow.Header),
			expectedDLen-binary.Size(dFlow.Header))
	}

	cFlow, err := GenerateIPFIX(flowCount, sourceID, "2001:db8::/32", "2001:db8::/32", session)
	if err != nil {
		t.Fatal(err)
	}
	cBuf := cFlow.ToBytes()
	expectedCLen := binary.Size(cFlow.Header)
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
