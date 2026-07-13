// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/dmabry/flowgre/utils"
)

func TestHeader_Generate(t *testing.T) {
	t.Parallel()
	flowCount := 10
	sourceID := 618
	session := NewSession()
	header := new(Header).Generate(flowCount, sourceID, session)

	if header.Version != 9 {
		t.Errorf("Header returned the wrong version! Got: %d Want: 9", header.Version)
	}
	if header.SysUptime == 0 {
		t.Errorf("Header returned the wrong system uptime! Got: %d Want: value", header.SysUptime)
	}
	if header.FlowCount != uint16(flowCount) {
		t.Errorf("Header returned the wrong flow count! Got: %d Want: %d", header.FlowCount, flowCount)
	}
	if header.UnixSec == 0 {
		t.Errorf("Header returned the wrong unix seconds! Got: %d Want: value", header.UnixSec)
	}
	if header.FlowSequence == 0 {
		t.Errorf("Header returned the wrong flow sequence! Got: %d Want: value", header.FlowSequence)
	}
	if header.SourceID != uint32(sourceID) {
		t.Errorf("Header returned the wrong source id! Got: %d Want: %d",
			header.SourceID, sourceID)
	}
}

func TestGenerateTemplateNetflow(t *testing.T) {
	t.Parallel()
	sourceID := 618
	session := NewSession()
	flow := GenerateTemplateNetflow(sourceID, session)
	if len(flow.TemplateFlowSets) < 1 {
		t.Errorf("Returned incorrect number of Template Flows! Got: %d Want: >1", len(flow.TemplateFlowSets))
	} else {
		for _, tFlow := range flow.TemplateFlowSets {
			if tFlow.FlowSetID != 0 {
				t.Errorf("Returned wrong FlowSetID! Got: %d Want: %d", tFlow.FlowSetID, 0)
			}
			if tFlow.Length != 80 {
				t.Errorf("Returned wrong length! Got: %d Want: %d", tFlow.Length, 80)
			}
		}
	}
}

func TestGenerateDataNetflow(t *testing.T) {
	t.Parallel()
	flowcount := 10
	sourceID := 618
	session := NewSession()
	flow, err := GenerateDataNetflow(flowcount, sourceID, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, session)
	if err != nil {
		t.Fatalf("GenerateDataNetflow failed: %v", err)
	}

	if len(flow.DataFlowSets) < 1 {
		t.Errorf("Returned incorrect number of Data Flows! Got: %d Want >: %d", len(flow.DataFlowSets), 1)
	} else {
		for _, dFlow := range flow.DataFlowSets {
			if dFlow.FlowSetID < 256 {
				t.Errorf("Returned invalid FlowSetID! Got: %d Want >: %d", dFlow.FlowSetID, 256)
			}
			if dFlow.Length < 64 {
				t.Errorf("Returned invalid length for given parameters! Got: %d Want: %d", dFlow.Length, 64)
			}
			if len(dFlow.Items) < flowcount {
				t.Errorf("Returned invalid number of flows! Got: %d Want: %d", len(dFlow.Items), flowcount)
			}
		}
	}
}

func TestToBytes(t *testing.T) {
	t.Parallel()
	// Generate Netflow Data
	sourceID := 618
	flowcount := 10
	session := NewSession()
	tflow := GenerateTemplateNetflow(sourceID, session)
	dflow, err := GenerateDataNetflow(flowcount, sourceID, "10.0.0.0/8", "10.0.0.0/8", utils.HTTPSPort, session)
	if err != nil {
		t.Fatalf("GenerateDataNetflow failed: %v", err)
	}
	// Convert to Bytes
	tbuf := tflow.ToBytes()
	dbuf := dflow.ToBytes()
	// Verify Bytes match generated Netflow Data
	tLength := tbuf.Len()
	dLength := dbuf.Len()
	tread := make([]byte, tLength)
	tn, err := tbuf.Read(tread)
	if err != nil {
		t.Errorf("Error during NetFlow Template Read! Got: %v", err)
	}
	if tn != tLength {
		t.Errorf("Returned invalid NetFlow Template buffer length! Got: %d Want: %d", tn, tbuf.Len())
	}
	dread := make([]byte, dLength)
	dn, err := dbuf.Read(dread)
	if err != nil {
		t.Errorf("Error during NetFlow Data Read! Got: %v", err)
	}
	if dn != dLength {
		t.Errorf("Returned invalid Netflow Data buffer length! Got: %d Want: %d", dn, dbuf.Len())
	}
	// Create readers and parse into new netflow structs
	tparsed := Netflow{}
	dparsed := Netflow{}
	treader := bytes.NewReader(tread)
	dreader := bytes.NewReader(dread)
	// Parse Template Header
	err = binary.Read(treader, binary.BigEndian, &tparsed.Header)
	if err != nil {
		t.Errorf("Failed to parse Netflow Header! Got: %v", err)
	}
	if int(tparsed.Header.SourceID) != sourceID {
		t.Errorf("Failed to parse Netflow Header Source ID! Got: %d Want: %d",
			int(tparsed.Header.SourceID), sourceID)
	}
	// Parse TemplateFlow
	tFlowCount := int(tparsed.Header.FlowCount)
	for range tFlowCount {
		tFlowSet := new(TemplateFlowSet)
		template := new(Template)
		err := binary.Read(treader, binary.BigEndian, &tFlowSet.FlowSetID)
		if err != nil {
			t.Errorf("Failed to parse Netflow FlowSetID! Got: %v", err)
		}
		err = binary.Read(treader, binary.BigEndian, &tFlowSet.Length)
		if err != nil {
			t.Errorf("Failed to parse Netflow FlowSet Length! Got: %v", err)
		}
		err = binary.Read(treader, binary.BigEndian, &template.TemplateID)
		if err != nil {
			t.Errorf("Failed to parse Netflow FlowSet TemplateID! Got: %v", err)
		}
		err = binary.Read(treader, binary.BigEndian, &template.FieldCount)
		if err != nil {
			t.Errorf("Failed to parse Netflow FlowSet FieldCount! Got: %v", err)
		}
		fc := int(template.FieldCount)
		for range fc {
			tField := new(Field)
			err := binary.Read(treader, binary.BigEndian, &tField.Type)
			if err != nil {
				t.Errorf("Failed to parse Netflow FlowSet Field Type! Got: %v", err)
			}
			err = binary.Read(treader, binary.BigEndian, &tField.Length)
			if err != nil {
				t.Errorf("Failed to parse Netflow FlowSet Field Length! Got: %v", err)
			}
			template.Fields = append(template.Fields, *tField)
		}
		tFlowSet.Templates = append(tFlowSet.Templates, *template)
		tparsed.TemplateFlowSets = append(tparsed.TemplateFlowSets, *tFlowSet)
		t.Log("Completed Template Flow parsing successfully")
	}
	// Parse Data Header
	err = binary.Read(dreader, binary.BigEndian, &dparsed.Header)
	if err != nil {
		t.Errorf("Failed to parse Netflow Header! Got: %v", err)
	}
	if int(dparsed.Header.SourceID) != sourceID {
		t.Errorf("Failed to parse Netflow Header Source ID! Got: %d Want: %d",
			int(dparsed.Header.SourceID), sourceID)
	}
	// Parse DataFlows
	dFlowSet := new(DataFlowSet)
	err = binary.Read(dreader, binary.BigEndian, &dFlowSet.FlowSetID)
	if err != nil {
		t.Errorf("Failed to parse Netflow Data FlowSetID! Got: %v", err)
	}
	err = binary.Read(dreader, binary.BigEndian, &dFlowSet.Length)
	if err != nil {
		t.Errorf("Failed to parse Netflow Data Length! Got: %v", err)
	}
	// I know the field count from the template generated above.  Going to use that
	dataItems := make([]any, flowcount)
	for i := range flowcount {
		dataItem := GenericFlow{}
		err := binary.Read(dreader, binary.BigEndian, &dataItem)
		if err != nil {
			t.Errorf("Issue reading in GenericFlow")
		}
		dataItems[i] = dataItem
	}
	dFlowSet.Items = dataItems
	if dreader.Len() > 0 {
		// read the rest as padding
		padLength := dreader.Len()
		padding := make([]byte, padLength)
		err := binary.Read(dreader, binary.BigEndian, padding)
		if err != nil {
			t.Errorf("Failed to parse Netflow Data Padding! Got: %v", err)
		}
		dFlowSet.Padding = padLength
	}
	dparsed.DataFlowSets = append(dparsed.DataFlowSets, *dFlowSet)
	t.Log("Completed Data Flow parsing successfully")
	// make sure they are equal
	if !cmp.Equal(tflow, tparsed) {
		t.Error("Failed Generated Netflow Template Flow and Parsed is different!")
	} else {
		t.Log("Generated Netflow Template Flow and Parsed Match!")
	}
	if !cmp.Equal(dflow, dparsed) {
		t.Error("Failed Generated Netflow Data Flow and Parsed is different!")
	} else {
		t.Log("Generated Netflow Data Flow and Parsed Match!")
	}
}

func TestGenericFlowIPv6(t *testing.T) {
	t.Parallel()

	srcIP := net.ParseIP("2001:db8::1")
	dstIP := net.ParseIP("2001:db8::2")
	session := NewSession()

	gf := new(GenericFlow)
	result := gf.Generate(srcIP, dstIP, utils.HTTPSPort, session)

	// IPv4 fields should be zeroed
	if result.Ipv4SrcAddr != 0 {
		t.Errorf("expected zeroed IPv4 src, got %d", result.Ipv4SrcAddr)
	}
	if result.Ipv4DstAddr != 0 {
		t.Errorf("expected zeroed IPv4 dst, got %d", result.Ipv4DstAddr)
	}

	// IPv6 fields should be populated
	expectedSrc := [16]byte{}
	copy(expectedSrc[:], srcIP.To16())
	if result.Ipv6SrcAddr != expectedSrc {
		t.Errorf("IPv6 src mismatch: got %v want %v", result.Ipv6SrcAddr, expectedSrc)
	}
	expectedDst := [16]byte{}
	copy(expectedDst[:], dstIP.To16())
	if result.Ipv6DstAddr != expectedDst {
		t.Errorf("IPv6 dst mismatch: got %v want %v", result.Ipv6DstAddr, expectedDst)
	}

	// Mask should be 64
	if result.Ipv6SrcMask != 64 {
		t.Errorf("expected IPv6 src mask 64, got %d", result.Ipv6SrcMask)
	}
	if result.Ipv6DstMask != 64 {
		t.Errorf("expected IPv6 dst mask 64, got %d", result.Ipv6DstMask)
	}
}

func TestGenericFlowIPv4BackwardCompat(t *testing.T) {
	t.Parallel()

	srcIP := net.ParseIP("10.0.0.1")
	dstIP := net.ParseIP("10.0.0.2")
	session := NewSession()

	gf := new(GenericFlow)
	result := gf.Generate(srcIP, dstIP, utils.HTTPSPort, session)

	// IPv4 fields should be populated
	if result.Ipv4SrcAddr != utils.IPToNum(srcIP) {
		t.Errorf("IPv4 src mismatch: got %d want %d", result.Ipv4SrcAddr, utils.IPToNum(srcIP))
	}
	if result.Ipv4DstAddr != utils.IPToNum(dstIP) {
		t.Errorf("IPv4 dst mismatch: got %d want %d", result.Ipv4DstAddr, utils.IPToNum(dstIP))
	}

	// IPv6 fields should be zeroed
	if result.Ipv6SrcAddr != [16]byte{} {
		t.Errorf("expected zeroed IPv6 src, got %v", result.Ipv6SrcAddr)
	}
	if result.Ipv6DstAddr != [16]byte{} {
		t.Errorf("expected zeroed IPv6 dst, got %v", result.Ipv6DstAddr)
	}
	if result.Ipv6SrcMask != 0 {
		t.Errorf("expected zeroed IPv6 src mask, got %d", result.Ipv6SrcMask)
	}
	if result.Ipv6DstMask != 0 {
		t.Errorf("expected zeroed IPv6 dst mask, got %d", result.Ipv6DstMask)
	}
}

func TestTemplateFieldsMatchStruct(t *testing.T) {
	t.Parallel()

	fields := new(GenericFlow).GetTemplateFields()

	// Template should have 18 fields (14 original + 4 IPv6)
	if len(fields) != 18 {
		t.Errorf("expected 18 template fields, got %d", len(fields))
	}

	// Verify field types are in correct order
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

func TestIPv6DataNetflow(t *testing.T) {
	t.Parallel()

	// Generate data flow with IPv6 CIDRs
	flowcount := 10
	sourceID := 618
	session := NewSession()
	flow, err := GenerateDataNetflow(flowcount, sourceID, "2001:db8::/48", "2001:db8:1::/48", utils.HTTPSPort, session)
	if err != nil {
		t.Fatalf("GenerateDataNetflow failed: %v", err)
	}

	if len(flow.DataFlowSets) < 1 {
		t.Fatal("expected at least one data flow set")
	}

	dFlow := flow.DataFlowSets[0]
	if len(dFlow.Items) != flowcount {
		t.Errorf("expected %d items, got %d", flowcount, len(dFlow.Items))
	}

	// Verify each item has valid IPv6 addresses
	for i, item := range dFlow.Items {
		gf, ok := item.(GenericFlow)
		if !ok {
			t.Errorf("item[%d]: expected GenericFlow, got %T", i, item)
			continue
		}
		// IPv6 addresses should not be all zeros
		if gf.Ipv6SrcAddr == [16]byte{} {
			t.Errorf("item[%d]: IPv6 src addr is all zeros", i)
		}
		if gf.Ipv6DstAddr == [16]byte{} {
			t.Errorf("item[%d]: IPv6 dst addr is all zeros", i)
		}
	}
}

func TestIPv6ToBytesRoundTrip(t *testing.T) {
	t.Parallel()

	sourceID := 618
	flowcount := 5
	session := NewSession()
	tflow := GenerateTemplateNetflow(sourceID, session)
	dflow, err := GenerateDataNetflow(flowcount, sourceID, "2001:db8::/48", "2001:db8:1::/48", utils.HTTPSPort, session)
	if err != nil {
		t.Fatalf("GenerateDataNetflow failed: %v", err)
	}

	// Serialize and deserialize
	tbuf := tflow.ToBytes()
	dbuf := dflow.ToBytes()

	// Verify template round-trip
	tLength := tbuf.Len()
	tread := make([]byte, tLength)
	tn, _ := tbuf.Read(tread)
	if tn != tLength {
		t.Fatalf("template buffer read mismatch: got %d want %d", tn, tLength)
	}

	// Verify data round-trip
	dLength := dbuf.Len()
	dread := make([]byte, dLength)
	dn, _ := dbuf.Read(dread)
	if dn != dLength {
		t.Fatalf("data buffer read mismatch: got %d want %d", dn, dLength)
	}

	// Parse data flow and verify IPv6 fields survive round-trip
	dreader := bytes.NewReader(dread)
	binary.Read(dreader, binary.BigEndian, &NetflowHeader{}) // skip header
	dFlowSet := new(DataFlowSet)
	binary.Read(dreader, binary.BigEndian, &dFlowSet.FlowSetID)
	binary.Read(dreader, binary.BigEndian, &dFlowSet.Length)

	for i := range flowcount {
		dataItem := GenericFlow{}
		err := binary.Read(dreader, binary.BigEndian, &dataItem)
		if err != nil {
			t.Fatalf("failed to read item[%d]: %v", i, err)
		}
		if dataItem.Ipv6SrcAddr == [16]byte{} {
			t.Errorf("item[%d]: IPv6 src lost in round-trip", i)
		}
		if dataItem.Ipv6DstAddr == [16]byte{} {
			t.Errorf("item[%d]: IPv6 dst lost in round-trip", i)
		}
	}
}

type NetflowHeader struct {
	Version      uint16
	FlowCount    uint16
	SysUptime    uint32
	UnixSec      uint32
	FlowSequence uint32
	SourceID     uint32
}
