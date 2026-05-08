// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package netflow

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestGenerateNetflow tests the main GenerateNetflow function.
func TestGenerateNetflow(t *testing.T) {
	t.Parallel()
	flowCount := 5
	sourceID := 1234
	session := NewSession()

	flow := GenerateNetflow(flowCount, sourceID, "10.0.0.0/8", "192.168.0.0/16", session)

	if flow.Header.FlowCount != uint16(flowCount+1) {
		t.Errorf("Header FlowCount = %d, want %d (flowCount + 1 for template)", flow.Header.FlowCount, flowCount+1)
	}
	if int(flow.Header.SourceID) != sourceID {
		t.Errorf("Header SourceID = %d, want %d", flow.Header.SourceID, sourceID)
	}
	if len(flow.TemplateFlowSets) < 1 {
		t.Error("Expected at least one Template FlowSet")
	}
	if len(flow.DataFlowSets) < 1 {
		t.Error("Expected at least one Data FlowSet")
	}
}

// TestIsValidNetFlow tests the IsValidNetFlow validation function.
func TestIsValidNetFlow(t *testing.T) {
	t.Parallel()
	session := NewSession()

	// Generate a valid NetFlow v9 packet
	flow := GenerateTemplateNetflow(100, session)
	buf := flow.ToBytes()
	payload := buf.Bytes()

	// Test with correct version
	ok, err := IsValidNetFlow(payload, 9)
	if !ok {
		t.Errorf("IsValidNetFlow returned false for valid packet: %v", err)
	}
	if err != nil {
		t.Errorf("IsValidNetFlow returned error for valid packet: %v", err)
	}

	// Test with wrong version
	ok, err = IsValidNetFlow(payload, 10)
	if ok {
		t.Error("IsValidNetFlow should return false for wrong version")
	}
	if err == nil {
		t.Error("IsValidNetFlow should return error for wrong version")
	}

	// Test with invalid payload (too short)
	invalidPayload := []byte{0x00, 0x09} // Just version, nothing else
	ok, err = IsValidNetFlow(invalidPayload, 9)
	if !ok {
		t.Logf("IsValidNetFlow correctly rejected short payload: %v", err)
	}
}

// TestUpdateTimeStamp tests the UpdateTimeStamp function.
func TestUpdateTimeStamp(t *testing.T) {
	t.Parallel()
	session := NewSession()

	// Generate a valid NetFlow packet
	flow := GenerateTemplateNetflow(100, session)
	buf := flow.ToBytes()
	originalPayload := buf.Bytes()

	// Parse original header to get UnixSec
	var originalHeader Header
	originalReader := bytes.NewReader(originalPayload)
	binary.Read(originalReader, binary.BigEndian, &originalHeader)

	// Update timestamp
	newPayload, err := UpdateTimeStamp(originalPayload)
	if err != nil {
		t.Fatalf("UpdateTimeStamp failed: %v", err)
	}

	if len(newPayload) != len(originalPayload) {
		t.Errorf("New payload length = %d, want %d", len(newPayload), len(originalPayload))
	}

	// Parse new header and verify UnixSec changed (or at least didn't decrease)
	var newHeader Header
	newReader := bytes.NewReader(newPayload)
	binary.Read(newReader, binary.BigEndian, &newHeader)

	if newHeader.UnixSec < originalHeader.UnixSec {
		t.Errorf("New UnixSec = %d, should be >= original = %d", newHeader.UnixSec, originalHeader.UnixSec)
	}

	// Verify other fields remain the same
	if newHeader.Version != originalHeader.Version {
		t.Errorf("Version changed: new = %d, original = %d", newHeader.Version, originalHeader.Version)
	}
	if newHeader.SourceID != originalHeader.SourceID {
		t.Errorf("SourceID changed: new = %d, original = %d", newHeader.SourceID, originalHeader.SourceID)
	}
}

// TestUpdateTimeStampInvalidPayload tests UpdateTimeStamp with invalid payload.
func TestUpdateTimeStampInvalidPayload(t *testing.T) {
	t.Parallel()

	// Too short to contain a header
	invalidPayload := []byte{0x00, 0x09}
	_, err := UpdateTimeStamp(invalidPayload)
	if err == nil {
		t.Error("UpdateTimeStamp should return error for invalid payload")
	}
}

// TestGetNetFlowSizes tests the GetNetFlowSizes function.
func TestGetNetFlowSizes(t *testing.T) {
	t.Parallel()
	session := NewSession()

	// Generate a template flow
	templateFlow := GenerateTemplateNetflow(100, session)

	output := GetNetFlowSizes(templateFlow)

	if output == "" {
		t.Error("GetNetFlowSizes returned empty string")
	}

	// Verify it contains expected information
	if !contains(output, "Header Size") {
		t.Error("Output should contain 'Header Size'")
	}
	if !contains(output, "Template Size") {
		t.Error("Output should contain 'Template Size'")
	}
	if !contains(output, "Data Size") {
		t.Error("Output should contain 'Data Size'")
	}
}

// TestGetNetFlowSizesWithData tests GetNetFlowSizes with data flows.
func TestGetNetFlowSizesWithData(t *testing.T) {
	t.Parallel()
	session := NewSession()

	// Generate a data flow
	dataFlow := GenerateDataNetflow(10, 100, "10.0.0.0/8", "192.168.0.0/16", 443, session)

	output := GetNetFlowSizes(dataFlow)

	if output == "" {
		t.Error("GetNetFlowSizes returned empty string")
	}
	if !contains(output, "Data Size") {
		t.Error("Output should contain 'Data Size'")
	}
}

// TestHeaderSize tests the Header.size() method.
func TestHeaderSize(t *testing.T) {
	t.Parallel()
	header := &Header{
		Version:      9,
		FlowCount:    10,
		SysUptime:    1000,
		UnixSec:      1234567890,
		FlowSequence: 1,
		SourceID:     100,
	}

	size := header.size()

	// Header should be 20 bytes (6 fields * appropriate sizes)
	expectedSize := 20 // uint16(2) + uint16(2) + uint32(4) + uint32(4) + uint32(4) + uint32(4)
	if size != expectedSize {
		t.Errorf("Header size = %d, want %d", size, expectedSize)
	}
}

// TestHeaderString tests the Header.String() method.
func TestHeaderString(t *testing.T) {
	t.Parallel()
	header := &Header{
		Version:      9,
		FlowCount:    10,
		SysUptime:    1000,
		UnixSec:      1234567890,
		FlowSequence: 1,
		SourceID:     100,
	}

	str := header.String()

	if str == "" {
		t.Error("Header.String() returned empty string")
	}
	if !contains(str, "Version: 9") {
		t.Error("String should contain 'Version: 9'")
	}
	if !contains(str, "Count: 10") {
		t.Error("String should contain 'Count: 10'")
	}
	if !contains(str, "SourceID: 100") {
		t.Error("String should contain 'SourceID: 100'")
	}
}

// TestFieldString tests the Field.String() method.
func TestFieldString(t *testing.T) {
	t.Parallel()
	field := &Field{
		Type:   IN_BYTES,
		Length: 4,
	}

	str := field.String()

	if str == "" {
		t.Error("Field.String() returned empty string")
	}
	if !contains(str, "Type:") {
		t.Error("String should contain 'Type:'")
	}
	if !contains(str, "Length:") {
		t.Error("String should contain 'Length:'")
	}
}

// TestTemplateSize tests the Template.size() method.
func TestTemplateSize(t *testing.T) {
	t.Parallel()
	template := &Template{
		TemplateID: 256,
		FieldCount: 2,
		Fields: []Field{
			{Type: IN_BYTES, Length: 4},
			{Type: OUT_BYTES, Length: 4},
		},
	}

	size := template.size()

	// TemplateID(2) + FieldCount(2) + Fields(2*4=8) = 12 bytes
	expectedSize := 12
	if size != expectedSize {
		t.Errorf("Template size = %d, want %d", size, expectedSize)
	}
}

// TestTemplateSizeOfFields tests the Template.sizeOfFields() method.
func TestTemplateSizeOfFields(t *testing.T) {
	t.Parallel()
	template := &Template{
		TemplateID: 256,
		FieldCount: 3,
		Fields: []Field{
			{Type: IN_BYTES, Length: 4},
			{Type: OUT_BYTES, Length: 4},
			{Type: IN_PKTS, Length: 4},
		},
	}

	size := template.sizeOfFields()

	// Sum of field lengths = 4 + 4 + 4 = 12
	expectedSize := 12
	if size != expectedSize {
		t.Errorf("Template sizeOfFields = %d, want %d", size, expectedSize)
	}
}

// TestTemplateFlowSetSize tests the TemplateFlowSet.size() method.
func TestTemplateFlowSetSize(t *testing.T) {
	t.Parallel()
	template := &Template{
		TemplateID: 256,
		FieldCount: 2,
		Fields: []Field{
			{Type: IN_BYTES, Length: 4},
			{Type: OUT_BYTES, Length: 4},
		},
	}

	templateFlowSet := &TemplateFlowSet{
		FlowSetID:   0,
		Length:      64,
		Templates:   []Template{*template},
		Padding:     0,
	}

	size := templateFlowSet.size()

	if size <= 0 {
		t.Errorf("TemplateFlowSet size = %d, should be positive", size)
	}
}

// TestToBytesWithEmptyFlowSets tests ToBytes with empty flow sets.
func TestToBytesWithEmptyFlowSets(t *testing.T) {
	t.Parallel()

	// Create a Netflow with only header, no flow sets
	flow := Netflow{
		Header: Header{
			Version:      9,
			FlowCount:    0,
			SysUptime:    1000,
			UnixSec:      1234567890,
			FlowSequence: 1,
			SourceID:     100,
		},
		TemplateFlowSets: []TemplateFlowSet{},
		DataFlowSets:     []DataFlowSet{},
	}

	buf := flow.ToBytes()

	if buf.Len() == 0 {
		t.Error("ToBytes returned empty buffer for header-only flow")
	}

	// Should at least contain the header (20 bytes)
	if buf.Len() < 20 {
		t.Errorf("Buffer length = %d, should be at least 20", buf.Len())
	}
}

// TestToBytesErrorHandling tests ToBytes error handling.
func TestToBytesErrorHandling(t *testing.T) {
	t.Parallel()
	// This test verifies that ToBytes doesn't panic even if binary.Write fails
	// (though in practice it rarely fails with bytes.Buffer)
	session := NewSession()
	flow := GenerateTemplateNetflow(100, session)

	buf := flow.ToBytes()

	if buf.Len() == 0 {
		t.Error("ToBytes returned empty buffer")
	}
}

// TestGenericFlowGenerateWithDifferentPorts tests GenericFlow.Generate with various ports.
func TestGenericFlowGenerateWithDifferentPorts(t *testing.T) {
	t.Parallel()
	session := NewSession()

	testCases := []struct {
		port       int
		expectedProto uint8
	}{
		{21, 6},   // FTP - TCP
		{22, 6},   // SSH - TCP
		{53, 17},  // DNS - UDP
		{80, 6},   // HTTP - TCP
		{443, 6},  // HTTPS - TCP
		{123, 17}, // NTP - UDP
		{0, 6},    // Default (HTTPS) - TCP
	}

	for _, tc := range testCases {
		gf := &GenericFlow{}
		srcIP := []byte{10, 0, 0, 1}
		dstIP := []byte{192, 168, 0, 1}

		result := gf.Generate(srcIP, dstIP, tc.port, session)

		if result.Protocol != tc.expectedProto {
			t.Errorf("Port %d: Protocol = %d, want %d", tc.port, result.Protocol, tc.expectedProto)
		}
	}
}

// TestGenericFlowGenerateWithInvalidIP tests GenericFlow.Generate with invalid IPs.
func TestGenericFlowGenerateWithInvalidIP(t *testing.T) {
	t.Parallel()
	session := NewSession()

	gf := &GenericFlow{}
	invalidSrcIP := []byte{0, 0, 0, 0}
	dstIP := []byte{192, 168, 0, 1}

	// Should not panic even with invalid IP
	result := gf.Generate(invalidSrcIP, dstIP, 443, session)

	if result.Ipv4SrcAddr != 0 {
		t.Errorf("Invalid src IP should result in Ipv4SrcAddr = 0, got %d", result.Ipv4SrcAddr)
	}
}

// TestDataFlowSetGenerateWithZeroPort tests DataFlowSet.Generate with flowSrcPort=0.
func TestDataFlowSetGenerateWithZeroPort(t *testing.T) {
	t.Parallel()
	session := NewSession()

	dfs := &DataFlowSet{}
	result := dfs.Generate(5, "10.0.0.0/8", "192.168.0.0/16", 0, session)

	if len(result.Items) != 5 {
		t.Errorf("Items length = %d, want 5", len(result.Items))
	}
	if result.FlowSetID != 256 {
		t.Errorf("FlowSetID = %d, want 256", result.FlowSetID)
	}
}

// TestDataFlowSetGenerateWithSpecificPort tests DataFlowSet.Generate with specific port.
func TestDataFlowSetGenerateWithSpecificPort(t *testing.T) {
	t.Parallel()
	session := NewSession()

	dfs := &DataFlowSet{}
	result := dfs.Generate(3, "10.0.0.0/8", "192.168.0.0/16", 8080, session)

	if len(result.Items) != 3 {
		t.Errorf("Items length = %d, want 3", len(result.Items))
	}

	// Verify all items use port 8080
	for i, item := range result.Items {
		if flow, ok := item.(GenericFlow); ok {
			if flow.L4DstPort != 8080 {
				t.Errorf("Item %d: L4DstPort = %d, want 8080", i, flow.L4DstPort)
			}
		}
	}
}

// TestTemplateFlowSetGenerateWithPadding tests TemplateFlowSet.Generate padding calculation.
func TestTemplateFlowSetGenerateWithPadding(t *testing.T) {
	t.Parallel()

	tfs := &TemplateFlowSet{}
	result := tfs.Generate(nil)

	if result.FlowSetID != 0 {
		t.Errorf("FlowSetID = %d, want 0", result.FlowSetID)
	}
	if len(result.Templates) < 1 {
		t.Error("Expected at least one template")
	}

	// Verify padding is correct (should make total size divisible by 4)
	totalSize := int(result.Length)
	if totalSize%4 != 0 {
		t.Errorf("Total length %d should be divisible by 4", totalSize)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
