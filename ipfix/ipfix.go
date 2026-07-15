// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// IPFIX implements RFC 7011 (IP Flow Information Export) packet generation.
// IPFIX is the IETF standard successor to NetFlow v9, using IANA-defined field
// type numbers instead of Cisco-specific ones.

package ipfix

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/utils"
)

// IPFIX version number per RFC 7011.
const Version = 10

// RFC 7011 Section 3.3.2: Set ID constants.
const (
	SetIDTemplate        = 2
	SetIDOptionsTemplate = 3
)

// IANA IPFIX Information Element identifiers (RFC 7011 / IANA registry).
const (
	OctetDeltaCount             = 1
	PacketDeltaCount            = 2
	IPClassOfService            = 5
	ProtocolIdentifier          = 4
	SourceTransportPort         = 7
	SourceIPv4Address           = 8
	DestinationTransportPort    = 11
	DestinationIPv4Address      = 12
	PostOctetDeltaCount         = 23
	PostPacketDeltaCount        = 24
	SourceIPv6Address           = 27
	DestinationIPv6Address      = 28
	SourceIPv6PrefixLength      = 29
	DestinationIPv6PrefixLength = 30
	FlowDirection               = 61
	TCPFlags                    = 6
	FlowStartMilliseconds       = 152
	FlowEndMilliseconds         = 153
	FlowEndReason               = 136
	ObservationDomainId         = 149
)

// Header is the RFC 7011 Section 3.1 IPFIX Message Header (16 bytes).
type Header struct {
	Version             uint16 // 10 for IPFIX
	Length              uint16 // total message length including this header
	ExportTime          uint32 // Unix epoch seconds
	SequenceNumber      uint32 // cumulative count of Data Records
	ObservationDomainId uint32 // observation domain identifier
}

// Generate creates an RFC 7011 IPFIX Header.
func (h *Header) Generate(sourceID int, sequenceNum uint32) Header {
	return Header{
		Version:             Version,
		ExportTime:          uint32(time.Now().Unix()),
		SequenceNumber:      sequenceNum,
		ObservationDomainId: uint32(sourceID),
	}
}

// Field describes a single field in an IPFIX template.
type Field struct {
	Type   uint16
	Length uint16
}

// Template describes an IPFIX template record.
type Template struct {
	TemplateID uint16
	FieldCount uint16
	Fields     []Field
}

// TemplateFlowSet wraps a collection of Template records.
// Per RFC 7011, FlowSetID is 2 for Template Sets.
type TemplateFlowSet struct {
	FlowSetID uint16
	Length    uint16
	Templates []Template
	Padding   int
}

// Generate creates a TemplateFlowSet with IPFIX field types.
// If profile is nil, defaults to GenericIPFIXProfile for backward compatibility.
func (t *TemplateFlowSet) Generate(_ *netflow.Session, profile ...IPFIXFlowProfile) TemplateFlowSet {
	p := IPFIXFlowProfile(&GenericIPFIXProfile{})
	if len(profile) > 0 && profile[0] != nil {
		p = profile[0]
	}

	fields := p.TemplateFields()

	template := Template{
		TemplateID: 256,
		FieldCount: uint16(len(fields)),
		Fields:     fields,
	}

	// FlowSetID(2) + Length(2) + TemplateID(2) + FieldCount(2) + fields*N*4
	rawSize := 4 + 4 + len(fields)*4
	padding := 0
	remainder := rawSize % 4
	if remainder > 0 {
		padding = 4 - remainder
		rawSize += padding
	}

	return TemplateFlowSet{
		FlowSetID: SetIDTemplate,
		Length:    uint16(rawSize),
		Templates: []Template{template},
		Padding:   padding,
	}
}

// OptionsTemplate describes an RFC 7011 Section 3.4.2.2 Options Template record.
type OptionsTemplate struct {
	TemplateID      uint16
	FieldCount      uint16 // total field count (scope + non-scope)
	ScopeFieldCount uint16
	Fields          []Field // first ScopeFieldCount are scope, remainder are non-scope
}

// OptionsTemplateFlowSet wraps an Options Template record.
// Per RFC 7011, FlowSetID is 3 for Options Template Sets.
type OptionsTemplateFlowSet struct {
	FlowSetID uint16
	Length    uint16
	Template  OptionsTemplate
	Padding   int
}

// Generate creates an OptionsTemplateFlowSet.
func (o *OptionsTemplateFlowSet) Generate(_ *netflow.Session) OptionsTemplateFlowSet {
	scopeFields := []Field{
		{Type: ObservationDomainId, Length: 4},
	}

	// All fields (scope + non-scope)
	allFields := append([]Field{}, scopeFields...)

	// Calculate raw size:
	// FlowSetID(2) + Length(2) + TemplateID(2) + FieldCount(2) + ScopeFieldCount(2)
	// + allFields*N*4
	rawSize := 4 + 6 + len(allFields)*4
	padding := 0
	remainder := rawSize % 4
	if remainder > 0 {
		padding = 4 - remainder
		rawSize += padding
	}

	return OptionsTemplateFlowSet{
		FlowSetID: SetIDOptionsTemplate,
		Length:    uint16(rawSize),
		Template: OptionsTemplate{
			TemplateID:      257,
			FieldCount:      uint16(len(allFields)),
			ScopeFieldCount: uint16(len(scopeFields)),
			Fields:          allFields,
		},
		Padding: padding,
	}
}

// OptionsDataRecord holds a single Options Data record.
type OptionsDataRecord struct {
	ObservationDomainId uint32
}

// OptionsDataFlowSet holds Options Data records.
type OptionsDataFlowSet struct {
	FlowSetID uint16
	Length    uint16
	Records   []OptionsDataRecord
	Padding   int
}

// Generate creates an OptionsDataFlowSet with observation domain metadata.
func (o *OptionsDataFlowSet) Generate(sourceID int) OptionsDataFlowSet {
	records := []OptionsDataRecord{
		{
			ObservationDomainId: uint32(sourceID),
		},
	}

	// FlowSetID(2) + Length(2) + ObservationDomainId(4) = 8
	length := 8
	padding := 0
	remainder := length % 4
	if remainder > 0 {
		padding = 4 - remainder
		length += padding
	}

	return OptionsDataFlowSet{
		FlowSetID: 257,
		Length:    uint16(length),
		Records:   records,
		Padding:   padding,
	}
}

// GenericFlow represents an IPFIX flow record with IANA field types.
// Field order must match GetTemplateFields() exactly.
type GenericFlow struct {
	OctetDeltaCount      uint32
	PostOctetDeltaCount  uint32
	PacketDeltaCount     uint32
	PostPacketDeltaCount uint32
	SourceIPv4Addr       uint32
	DestIPv4Addr         uint32
	SourceIPv6Addr       [16]byte
	DestIPv6Addr         [16]byte
	SourceIPv6Prefix     uint8
	DestIPv6Prefix       uint8
	SourcePort           uint16
	DestPort             uint16
	ProtocolIdentifier   uint8
	TCPFlags             uint8
	FlowStartMillis      uint64
	FlowEndMillis        uint64
	FlowDirection        uint8
	IPClassOfService     uint8
	FlowEndReason        uint8
}

// GetTemplateFields returns the IPFIX field definitions for the template.
func (gf *GenericFlow) GetTemplateFields() []Field {
	return []Field{
		{Type: OctetDeltaCount, Length: 4},
		{Type: PostOctetDeltaCount, Length: 4},
		{Type: PacketDeltaCount, Length: 4},
		{Type: PostPacketDeltaCount, Length: 4},
		{Type: SourceIPv4Address, Length: 4},
		{Type: DestinationIPv4Address, Length: 4},
		{Type: SourceIPv6Address, Length: 16},
		{Type: DestinationIPv6Address, Length: 16},
		{Type: SourceIPv6PrefixLength, Length: 1},
		{Type: DestinationIPv6PrefixLength, Length: 1},
		{Type: SourceTransportPort, Length: 2},
		{Type: DestinationTransportPort, Length: 2},
		{Type: ProtocolIdentifier, Length: 1},
		{Type: TCPFlags, Length: 1},
		{Type: FlowStartMilliseconds, Length: 8},
		{Type: FlowEndMilliseconds, Length: 8},
		{Type: FlowDirection, Length: 1},
		{Type: IPClassOfService, Length: 1},
		{Type: FlowEndReason, Length: 1},
	}
}

// Generate creates a GenericFlow with randomly generated data.
func (gf *GenericFlow) Generate(srcIP net.IP, dstIP net.IP, flowSrcPort int, session *netflow.Session) GenericFlow {
	now := time.Now()
	epochMillis := uint64(now.UnixMilli())

	gf.OctetDeltaCount = utils.GenerateRand32(10000)
	gf.PostOctetDeltaCount = utils.GenerateRand32(10000)
	gf.PacketDeltaCount = utils.GenerateRand32(10000)
	gf.PostPacketDeltaCount = utils.GenerateRand32(10000)

	if srcIP.To4() != nil {
		gf.SourceIPv4Addr = utils.IPToNum(srcIP)
		gf.DestIPv4Addr = utils.IPToNum(dstIP)
		gf.SourceIPv6Addr = [16]byte{}
		gf.DestIPv6Addr = [16]byte{}
		gf.SourceIPv6Prefix = 0
		gf.DestIPv6Prefix = 0
	} else {
		gf.SourceIPv4Addr = 0
		gf.DestIPv4Addr = 0
		copy(gf.SourceIPv6Addr[:], srcIP.To16())
		copy(gf.DestIPv6Addr[:], dstIP.To16())
		gf.SourceIPv6Prefix = 64
		gf.DestIPv6Prefix = 64
	}

	gf.SourcePort = utils.GenerateRand16(10000)
	gf.TCPFlags = uint8(utils.RandomNum(0, 32))
	gf.FlowStartMillis = epochMillis - 100
	gf.FlowEndMillis = epochMillis - 10
	gf.FlowDirection = 0
	gf.IPClassOfService = 0
	gf.FlowEndReason = uint8(utils.RandomNum(0, 4))

	gf.DestPort, gf.ProtocolIdentifier = utils.ResolvePortProtocol(flowSrcPort)

	return *gf
}

// DataAny holds a data record of any type.
type DataAny any

// DataFlowSet holds flow data records.
type DataFlowSet struct {
	FlowSetID uint16
	Length    uint16
	Items     []DataAny
	Padding   int
}

// Generate creates a DataFlowSet with random flow data.
func (d *DataFlowSet) Generate(flowCount int, srcRange string, dstRange string, flowSrcPort int, session *netflow.Session, profile ...IPFIXFlowProfile) (DataFlowSet, error) {
	_ = profile

	protoPorts := utils.ProtoPorts

	items := make([]DataAny, flowCount)
	for i := range flowCount {
		srcIP, err := utils.RandomIPCIDR(srcRange)
		if err != nil {
			return DataFlowSet{}, fmt.Errorf("failed to generate src IP for flow %d: %w", i, err)
		}
		dstIP, err := utils.RandomIPCIDR(dstRange)
		if err != nil {
			return DataFlowSet{}, fmt.Errorf("failed to generate dst IP for flow %d: %w", i, err)
		}
		port := flowSrcPort
		if port == 0 {
			port = protoPorts[utils.RandomNum(0, len(protoPorts))]
		}
		items[i] = new(GenericFlow).Generate(srcIP, dstIP, port, session)
	}

	// Calculate length: FlowSetID(2) + Length(2) + records + padding
	recordSize := binary.Size(GenericFlow{})
	length := 4 + flowCount*recordSize
	padding := 0
	remainder := length % 4
	if remainder > 0 {
		padding = 4 - remainder
		length += padding
	}

	// Validate full-width length before converting to uint16.
	// Without this check, length wraps and estimatedSize() sees a small value,
	// allowing oversized packets to reserve sequence numbers before ToBytes() rejects them.
	if length > 65535 {
		return DataFlowSet{}, fmt.Errorf("DataFlowSet length %d exceeds maximum 65535 bytes", length)
	}

	return DataFlowSet{
		FlowSetID: 256,
		Length:    uint16(length),
		Items:     items,
		Padding:   padding,
	}, nil
}

// IPFIX is the complete IPFIX export packet structure.
type IPFIX struct {
	Header                  Header
	TemplateFlowSets        []TemplateFlowSet
	OptionsTemplateFlowSets []OptionsTemplateFlowSet
	DataFlowSets            []DataFlowSet
	OptionsDataFlowSets     []OptionsDataFlowSet
}

// dataRecordCount returns the total number of Data Records across all DataFlowSets
// and OptionsDataFlowSets. Per RFC 7011, only Data Records contribute to the
// Sequence Number.
func (f *IPFIX) dataRecordCount() int {
	count := 0
	for _, dfs := range f.DataFlowSets {
		count += len(dfs.Items)
	}
	for _, odfs := range f.OptionsDataFlowSets {
		count += len(odfs.Records)
	}
	return count
}

// estimatedSize returns the estimated message size in bytes.
// This is used to check for overflow before reserving sequence numbers.
func (f *IPFIX) estimatedSize() int {
	size := 16 // header
	for _, tfs := range f.TemplateFlowSets {
		size += int(tfs.Length)
	}
	for _, ots := range f.OptionsTemplateFlowSets {
		size += int(ots.Length)
	}
	for _, dfs := range f.DataFlowSets {
		size += int(dfs.Length)
	}
	for _, ods := range f.OptionsDataFlowSets {
		size += int(ods.Length)
	}
	return size
}

// mustWriteBinary writes data to a bytes.Buffer using binary.Write.
// bytes.Buffer.Write never fails, so errors are silently ignored.
// This helper satisfies gosec G104 (unhandled errors).
func mustWriteBinary(buf *bytes.Buffer, data any) {
	_ = binary.Write(buf, binary.BigEndian, data)
}

// ToBytes serializes the IPFIX structure to a byte buffer for wire transmission.
// The Header.Length is set to the total encoded message size.
// Returns an error if the message exceeds the 65535-byte IPFIX limit.
func (f *IPFIX) ToBytes() (bytes.Buffer, error) {
	var setsBuf bytes.Buffer

	// Serialize all FlowSets
	for _, tFlow := range f.TemplateFlowSets {
		mustWriteBinary(&setsBuf, tFlow.FlowSetID)
		mustWriteBinary(&setsBuf, tFlow.Length)
		for _, template := range tFlow.Templates {
			mustWriteBinary(&setsBuf, template.TemplateID)
			mustWriteBinary(&setsBuf, template.FieldCount)
			for _, field := range template.Fields {
				mustWriteBinary(&setsBuf, field.Type)
				mustWriteBinary(&setsBuf, field.Length)
			}
		}
		if tFlow.Padding > 0 {
			setsBuf.Write(bytes.Repeat([]byte{0}, tFlow.Padding))
		}
	}

	for _, oFlow := range f.OptionsTemplateFlowSets {
		mustWriteBinary(&setsBuf, oFlow.FlowSetID)
		mustWriteBinary(&setsBuf, oFlow.Length)
		t := oFlow.Template
		mustWriteBinary(&setsBuf, t.TemplateID)
		mustWriteBinary(&setsBuf, t.FieldCount)
		mustWriteBinary(&setsBuf, t.ScopeFieldCount)
		for _, field := range t.Fields {
			mustWriteBinary(&setsBuf, field.Type)
			mustWriteBinary(&setsBuf, field.Length)
		}
		if oFlow.Padding > 0 {
			setsBuf.Write(bytes.Repeat([]byte{0}, oFlow.Padding))
		}
	}

	for _, dFlow := range f.DataFlowSets {
		mustWriteBinary(&setsBuf, dFlow.FlowSetID)
		mustWriteBinary(&setsBuf, dFlow.Length)
		for _, item := range dFlow.Items {
			mustWriteBinary(&setsBuf, item)
		}
		if dFlow.Padding > 0 {
			setsBuf.Write(bytes.Repeat([]byte{0}, dFlow.Padding))
		}
	}

	for _, oData := range f.OptionsDataFlowSets {
		mustWriteBinary(&setsBuf, oData.FlowSetID)
		mustWriteBinary(&setsBuf, oData.Length)
		for _, rec := range oData.Records {
			mustWriteBinary(&setsBuf, rec.ObservationDomainId)
		}
		if oData.Padding > 0 {
			setsBuf.Write(bytes.Repeat([]byte{0}, oData.Padding))
		}
	}

	setsBytes := setsBuf.Bytes()
	totalLength := 16 + len(setsBytes)

	// Check for uint16 overflow
	if totalLength > 65535 {
		return bytes.Buffer{}, fmt.Errorf("IPFIX message size %d exceeds maximum 65535 bytes", totalLength)
	}

	// Build final buffer: header first, then sets
	var result bytes.Buffer
	result.Grow(totalLength)

	// Write header with correct Length
	h := f.Header
	h.Length = uint16(totalLength)
	mustWriteBinary(&result, h.Version)
	mustWriteBinary(&result, h.Length)
	mustWriteBinary(&result, h.ExportTime)
	mustWriteBinary(&result, h.SequenceNumber)
	mustWriteBinary(&result, h.ObservationDomainId)

	// Write sets
	result.Write(setsBytes)

	return result, nil
}

// fieldSpecifierSize returns the size of a field specifier at the given offset.
// Standard specifiers are 4 bytes; enterprise specifiers are 8 bytes.
// RFC 7011 §3.2: The Enterprise bit is the high bit of Element Length.
func fieldSpecifierSize(payload []byte, offset int) int {
	if offset+4 > len(payload) {
		return 4 // assume standard if we can't read
	}
	elementLength := binary.BigEndian.Uint16(payload[offset : offset+2])
	if elementLength&0x8000 != 0 {
		return 8 // enterprise specifier has 4-byte PEN
	}
	return 4
}

// validateTemplateRecord validates a Template record starting at setOffset.
// Returns the number of bytes consumed (excluding padding).
func validateTemplateRecord(payload []byte, setOffset, remaining int) (int, error) {
	if remaining < 4 {
		return 0, fmt.Errorf("insufficient data for Template record header")
	}
	templateID := binary.BigEndian.Uint16(payload[setOffset : setOffset+2])
	fieldCount := binary.BigEndian.Uint16(payload[setOffset+2 : setOffset+4])

	if templateID < 256 {
		return 0, fmt.Errorf("Template ID %d is below 256", templateID)
	}

	// RFC 7011 §8.1: A Template Record with Field Count of 0 is a
	// Template Withdrawal. It contains only the Template ID and Field Count.
	if fieldCount == 0 {
		return 4, nil
	}

	consumed := 4
	for i := uint16(0); i < fieldCount; i++ {
		if consumed+4 > remaining {
			return 0, fmt.Errorf("insufficient data for field specifier %d", i)
		}
		specSize := fieldSpecifierSize(payload, setOffset+consumed)
		if consumed+specSize > remaining {
			return 0, fmt.Errorf("enterprise field specifier %d exceeds remaining bytes", i)
		}
		consumed += specSize
	}
	return consumed, nil
}

// validateOptionsTemplateRecord validates an Options Template record.
// Returns the number of bytes consumed (excluding padding).
func validateOptionsTemplateRecord(payload []byte, setOffset, remaining int) (int, error) {
	if remaining < 4 {
		return 0, fmt.Errorf("insufficient data for Options Template record header")
	}
	templateID := binary.BigEndian.Uint16(payload[setOffset : setOffset+2])
	fieldCount := binary.BigEndian.Uint16(payload[setOffset+2 : setOffset+4])

	if templateID < 256 {
		return 0, fmt.Errorf("Options Template ID %d is below 256", templateID)
	}

	// RFC 7011 §8.1: Options Template Withdrawal is a 4-byte record with
	// Template ID and Field Count of zero (no Scope Field Count field).
	if fieldCount == 0 {
		return 4, nil
	}

	// Normal Options Template: requires Scope Field Count (§3.4.2.2).
	if remaining < 6 {
		return 0, fmt.Errorf("insufficient data for Scope Field Count")
	}
	scopeFieldCount := binary.BigEndian.Uint16(payload[setOffset+4 : setOffset+6])

	// RFC 7011 §3.4.2.2: Scope Field Count must be at least 1.
	if scopeFieldCount == 0 {
		return 0, fmt.Errorf("Scope Field Count must be at least 1")
	}
	if scopeFieldCount > fieldCount {
		return 0, fmt.Errorf("Scope Field Count %d > Field Count %d", scopeFieldCount, fieldCount)
	}

	consumed := 6
	for i := uint16(0); i < fieldCount; i++ {
		if consumed+4 > remaining {
			return 0, fmt.Errorf("insufficient data for field specifier %d", i)
		}
		specSize := fieldSpecifierSize(payload, setOffset+consumed)
		if consumed+specSize > remaining {
			return 0, fmt.Errorf("enterprise field specifier %d exceeds remaining bytes", i)
		}
		consumed += specSize
	}
	return consumed, nil
}

// IsValidIPFIX validates the given payload against RFC 7011.
func IsValidIPFIX(payload []byte) (bool, error) {
	if len(payload) < 16 {
		return false, fmt.Errorf("payload too short for IPFIX header: %d bytes", len(payload))
	}

	version := binary.BigEndian.Uint16(payload[0:2])
	if version != Version {
		return false, fmt.Errorf("version %d is not IPFIX (expected %d)", version, Version)
	}

	messageLength := binary.BigEndian.Uint16(payload[2:4])
	if int(messageLength) < 16 {
		return false, fmt.Errorf("declared message length %d is less than minimum header size", messageLength)
	}
	if int(messageLength) != len(payload) {
		return false, fmt.Errorf("declared message length %d does not match payload length %d", messageLength, len(payload))
	}

	offset := 16
	limit := int(messageLength)
	setCount := 0

	for offset < limit {
		if offset+4 > limit {
			return false, fmt.Errorf("insufficient data for FlowSet header at offset %d", offset)
		}

		setID := binary.BigEndian.Uint16(payload[offset : offset+2])
		setLength := binary.BigEndian.Uint16(payload[offset+2 : offset+4])

		if setID == 0 || setID == 1 {
			return false, fmt.Errorf("reserved Set ID %d at offset %d", setID, offset)
		}

		if setID >= 4 && setID <= 255 {
			return false, fmt.Errorf("unassigned Set ID %d at offset %d", setID, offset)
		}

		if int(setLength) < 4 {
			return false, fmt.Errorf("FlowSet length %d is less than minimum 4 at offset %d", setLength, offset)
		}

		setEnd := offset + int(setLength)
		if setEnd > limit {
			return false, fmt.Errorf("FlowSet at offset %d extends beyond message boundary", offset)
		}

		// Validate set contents based on type
		remaining := int(setLength) - 4 // subtract FlowSet header
		setOffset := offset + 4

		switch setID {
		case SetIDTemplate:
			// Template Set: one or more Template records
			recordsParsed := 0
			for remaining >= 4 {
				consumed, err := validateTemplateRecord(payload, setOffset, remaining)
				if err != nil {
					return false, fmt.Errorf("Template Set at offset %d: %w", offset, err)
				}
				setOffset += consumed
				remaining -= consumed
				recordsParsed++
			}
			if recordsParsed == 0 {
				return false, fmt.Errorf("Template Set at offset %d contains no records", offset)
			}
		case SetIDOptionsTemplate:
			// Options Template Set: one or more Options Template records.
			// Minimum is 4 bytes (withdrawal: TemplateID + FieldCount).
			recordsParsed := 0
			for remaining >= 4 {
				consumed, err := validateOptionsTemplateRecord(payload, setOffset, remaining)
				if err != nil {
					return false, fmt.Errorf("Options Template Set at offset %d: %w", offset, err)
				}
				setOffset += consumed
				remaining -= consumed
				recordsParsed++
			}
			if recordsParsed == 0 {
				return false, fmt.Errorf("Options Template Set at offset %d contains no records", offset)
			}
		default:
			// Data Set (ID >= 256): records are opaque, just check length
		}

		setCount++
		offset = setEnd
	}

	// RFC 7011 permits zero or more Sets. A 16-byte header-only message is valid.

	return true, nil
}

// UpdateTimeStamp updates the ExportTime field in an IPFIX packet header to the current time.
func UpdateTimeStamp(payload []byte) ([]byte, error) {
	if len(payload) < 16 {
		return nil, fmt.Errorf("payload too short for IPFIX header: %d bytes", len(payload))
	}

	result := make([]byte, len(payload))
	copy(result, payload)

	binary.BigEndian.PutUint32(result[4:8], uint32(time.Now().Unix()))

	return result, nil
}

// IPFIXSequence tracks the RFC 7011 Data Record sequence number.
// The sequence number counts Data Records exported before the current message.
// Template and Options Template Records do not increment it.
type IPFIXSequence struct {
	counter atomic.Uint32
}

// NewIPFIXSequence creates a new sequence tracker.
func NewIPFIXSequence() *IPFIXSequence {
	return &IPFIXSequence{}
}

// Reserve returns the current sequence number and atomically advances
// the counter by recordCount. The returned value is the sequence number
// for the message containing recordCount Data Records.
func (s *IPFIXSequence) Reserve(recordCount int) uint32 {
	return s.counter.Add(uint32(recordCount)) - uint32(recordCount)
}

// Current returns the current sequence number without advancing.
func (s *IPFIXSequence) Current() uint32 {
	return s.counter.Load()
}

// GenerateTemplateIPFIX creates an IPFIX packet containing template and options template FlowSets.
// The sequence number reflects the current count of Data Records sent.
func GenerateTemplateIPFIX(sourceID int, seq *IPFIXSequence) IPFIX {
	templateFlow := new(TemplateFlowSet).Generate(nil)
	optionsTemplate := new(OptionsTemplateFlowSet).Generate(nil)

	// Template messages carry the current Data Record count, don't advance it
	header := new(Header).Generate(sourceID, seq.Current())

	return IPFIX{
		Header:                  header,
		TemplateFlowSets:        []TemplateFlowSet{templateFlow},
		OptionsTemplateFlowSets: []OptionsTemplateFlowSet{optionsTemplate},
		DataFlowSets:            nil,
		OptionsDataFlowSets:     nil,
	}
}

// GenerateOptionsDataIPFIX creates an IPFIX packet containing Options Data records.
func GenerateOptionsDataIPFIX(sourceID int, seq *IPFIXSequence) IPFIX {
	optionsData := new(OptionsDataFlowSet).Generate(sourceID)

	// Options Data Records are Data Records; reserve sequence for 1 record
	header := new(Header).Generate(sourceID, seq.Reserve(1))

	return IPFIX{
		Header:              header,
		OptionsDataFlowSets: []OptionsDataFlowSet{optionsData},
	}
}

// GenerateDataIPFIX creates an IPFIX packet containing only data FlowSets.
func GenerateDataIPFIX(flowCount int, sourceID int, srcRange string, dstRange string, flowSrcPort int, seq *IPFIXSequence) (IPFIX, error) {
	session := netflow.NewSession()
	dataFlow, err := new(DataFlowSet).Generate(flowCount, srcRange, dstRange, flowSrcPort, session)
	if err != nil {
		return IPFIX{}, fmt.Errorf("generate data flow set: %w", err)
	}

	// Build the packet without header to check size first
	flow := IPFIX{
		DataFlowSets: []DataFlowSet{dataFlow},
	}
	if flow.estimatedSize() > 65535 {
		return IPFIX{}, fmt.Errorf("IPFIX message size %d exceeds maximum 65535 bytes", flow.estimatedSize())
	}

	// Size is OK; now reserve sequence range
	flow.Header = new(Header).Generate(sourceID, seq.Reserve(flowCount))

	return flow, nil
}

// GenerateIPFIX creates an IPFIX packet containing both template and data FlowSets.
func GenerateIPFIX(flowCount int, sourceID int, srcRange string, dstRange string, seq *IPFIXSequence) (IPFIX, error) {
	templateFlow := new(TemplateFlowSet).Generate(nil)
	session := netflow.NewSession()
	dataFlow, err := new(DataFlowSet).Generate(flowCount, srcRange, dstRange, utils.HTTPSPort, session)
	if err != nil {
		return IPFIX{}, fmt.Errorf("generate data flow set: %w", err)
	}

	// Build the packet without header to check size first
	flow := IPFIX{
		TemplateFlowSets: []TemplateFlowSet{templateFlow},
		DataFlowSets:     []DataFlowSet{dataFlow},
	}
	if flow.estimatedSize() > 65535 {
		return IPFIX{}, fmt.Errorf("IPFIX message size %d exceeds maximum 65535 bytes", flow.estimatedSize())
	}

	// Size is OK; now reserve sequence range
	flow.Header = new(Header).Generate(sourceID, seq.Reserve(flowCount))

	return flow, nil
}

// size returns the size of the Header in bytes (always 16 per RFC 7011).
func (h *Header) size() int {
	return 16
}

// String returns a human-readable representation of the Header.
func (h *Header) String() string {
	return fmt.Sprintf("Version: %d Length: %d ExportTime: %d SequenceNumber: %d ObservationDomainId: %d",
		h.Version, h.Length, h.ExportTime, h.SequenceNumber, h.ObservationDomainId)
}

// size returns the size of the TemplateFlowSet in bytes.
func (t *TemplateFlowSet) size() int {
	size := binary.Size(t.FlowSetID)
	size += binary.Size(t.Length)
	for _, i := range t.Templates {
		size += binary.Size(i.TemplateID)
		size += binary.Size(i.FieldCount)
		for _, f := range i.Fields {
			size += binary.Size(f.Type)
			size += binary.Size(f.Length)
		}
	}
	size += t.Padding
	return size
}

// size returns the size of the DataFlowSet in bytes.
func (d *DataFlowSet) size() int {
	size := binary.Size(d.FlowSetID)
	size += binary.Size(d.Length)
	for _, item := range d.Items {
		size += binary.Size(item)
	}
	size += d.Padding
	return size
}

// GetIPFIXSizes returns a human-readable string with the size breakdown of an IPFIX packet.
func GetIPFIXSizes(ipfix IPFIX) string {
	output := "Header Size: " + fmt.Sprintf("%d", ipfix.Header.size()) + " bytes\n"
	tSize := 0
	for _, tFlow := range ipfix.TemplateFlowSets {
		tSize += tFlow.size()
	}
	output += "Template Size: " + fmt.Sprintf("%d", tSize) + " bytes\n"
	dSize := 0
	for _, dFlow := range ipfix.DataFlowSets {
		dSize += dFlow.size()
	}
	output += "Data Size: " + fmt.Sprintf("%d", dSize) + " bytes\n"
	return output
}
