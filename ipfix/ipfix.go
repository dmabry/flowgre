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
	"log"
	"net"
	"os"
	"time"

	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/utils"
)

// IPFIX version number per RFC 7011.
const Version = 10

// IANA IPFIX field type constants (RFC 7011 / Information Model)
const (
	ProtocolIdentifier          = 4
	SourceTransportPort         = 7
	SourceIPv4Address           = 8
	DestinationTransportPort    = 11
	DestinationIPv4Address      = 12
	SourceIPv6Address           = 25
	DestinationIPv6Address      = 26
	SourceIPv6PrefixLength      = 47
	DestinationIPv6PrefixLength = 48
	FlowDirection               = 1024
	InPackets                   = 1025
	InOctets                    = 1026
	OutPackets                  = 1027
	OutOctets                   = 1028
	IPClassOfService            = 3
	TCPFlags                    = 6
	FlowStartMilliseconds       = 152
	FlowEndMilliseconds         = 153
	FlowEndReason               = 157
	// Options Template fields
	ObservationDomainId = 31
	ProcessName         = 279
	ProcessId           = 278
)

// Header is the IPFIX export set header, structurally identical to NetFlow v9
// but with version=10.
type Header struct {
	Version      uint16
	FlowCount    uint16
	SysUptime    uint32
	UnixSec      uint32
	FlowSequence uint32
	SourceID     uint32
}

// Generate creates an IPFIX Header with version 10.
func (h *Header) Generate(flowSetCount int, sourceID int, session *netflow.Session) Header {
	now := time.Now().UnixNano()
	secs := now / int64(time.Second)
	startTime := session.StartTime()
	sysUptime := uint32((now-startTime)/int64(time.Millisecond)) + 1000

	return Header{
		Version:      Version,
		SysUptime:    sysUptime,
		UnixSec:      uint32(secs),
		FlowCount:    uint16(flowSetCount),
		FlowSequence: session.NextSeq(),
		SourceID:     uint32(sourceID),
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
// Per IPFIX spec, FlowSetID is always 0 for template FlowSets.
type TemplateFlowSet struct {
	FlowSetID uint16
	Length    uint16
	Templates []Template
	Padding   int
}

// Generate creates a TemplateFlowSet with IPFIX field types.
// If profile is nil, defaults to GenericIPFIXProfile for backward compatibility.
func (t *TemplateFlowSet) Generate(session *netflow.Session, profile ...IPFIXFlowProfile) TemplateFlowSet {
	p := IPFIXFlowProfile(&GenericIPFIXProfile{}) // default
	if len(profile) > 0 && profile[0] != nil {
		p = profile[0]
	}

	fields := p.TemplateFields()

	template := Template{
		TemplateID: 256,
		FieldCount: uint16(len(fields)),
		Fields:     fields,
	}

	rawSize := 4 + 4 + len(fields)*4 // FlowSetID+Length + TemplateID+FieldCount + fields
	padding := 0
	remainder := rawSize % 4
	if remainder > 0 {
		padding = 4 - remainder
		rawSize += padding
	}

	return TemplateFlowSet{
		FlowSetID: 0,
		Length:    uint16(rawSize),
		Templates: []Template{template},
		Padding:   padding,
	}
}

// OptionsTemplate describes an IPFIX Options Template record.
// Per RFC 7011, Options Templates have two field lists: Scope Fields and Data Fields.
type OptionsTemplate struct {
	TemplateID      uint16
	ScopeFieldCount uint16
	ScopeFields     []Field
	DataFieldCount  uint16
	DataFields      []Field
}

// OptionsTemplateFlowSet wraps an Options Template record.
// Per IPFIX spec, FlowSetID is always 0 for template FlowSets (including options).
type OptionsTemplateFlowSet struct {
	FlowSetID uint16
	Length    uint16
	Template  OptionsTemplate
	Padding   int
}

// Generate creates an OptionsTemplateFlowSet with process metadata fields.
func (o *OptionsTemplateFlowSet) Generate(session *netflow.Session) OptionsTemplateFlowSet {
	scopeFields := []Field{
		{Type: ObservationDomainId, Length: 4},
	}
	dataFields := []Field{
		{Type: ProcessName, Length: 0}, // variable-length, set at serialization
		{Type: ProcessId, Length: 4},
	}

	// Calculate raw size: FlowSetID(2) + Length(2) + TemplateID(2) + ScopeFieldCount(2)
	// + ScopeFields[Scope(2)+Len(2)] + DataFieldCount(2) + DataFields[(2+2)*N]
	rawSize := 4 + 4 + len(scopeFields)*4 + 2 + len(dataFields)*4
	padding := 0
	remainder := rawSize % 4
	if remainder > 0 {
		padding = 4 - remainder
		rawSize += padding
	}

	return OptionsTemplateFlowSet{
		FlowSetID: 0,
		Length:    uint16(rawSize),
		Template: OptionsTemplate{
			TemplateID:      257,
			ScopeFieldCount: uint16(len(scopeFields)),
			ScopeFields:     scopeFields,
			DataFieldCount:  uint16(len(dataFields)),
			DataFields:      dataFields,
		},
		Padding: padding,
	}
}

// OptionsDataRecord holds a single Options Data record.
type OptionsDataRecord struct {
	ObservationDomainId uint32
	ProcessName         string
	ProcessId           uint32
}

// OptionsDataFlowSet holds Options Data records.
type OptionsDataFlowSet struct {
	FlowSetID uint16
	Length    uint16
	Records   []OptionsDataRecord
	Padding   int
}

// Generate creates an OptionsDataFlowSet with process metadata.
func (o *OptionsDataFlowSet) Generate(sourceID int, processName string, pid uint32) OptionsDataFlowSet {
	records := []OptionsDataRecord{
		{
			ObservationDomainId: uint32(sourceID),
			ProcessName:         processName,
			ProcessId:           pid,
		},
	}

	// Calculate length: FlowSetID(2) + Length(2) + ScopeValues + DataValues + padding
	// Scope: ObservationDomainId(4)
	// Data: ProcessNameLen(2) + ProcessName(N) + ProcessId(4)
	recordSize := 4 + 2 + len(processName) + 4
	length := 4 + recordSize
	padding := 0
	remainder := length % 4
	if remainder > 0 {
		padding = 4 - remainder
		length += padding
	}

	return OptionsDataFlowSet{
		FlowSetID: 257, // Must match Options TemplateID
		Length:    uint16(length),
		Records:   records,
		Padding:   padding,
	}
}

// GenericFlow represents an IPFIX flow record with IANA field types.
// Field order must match GetTemplateFields() exactly — binary.Write
// serializes in struct field order, and the template defines the wire format.
type GenericFlow struct {
	InOctets           uint32
	OutOctets          uint32
	InPackets          uint32
	OutPackets         uint32
	SourceIPv4Addr     uint32
	DestIPv4Addr       uint32
	SourceIPv6Addr     [16]byte
	DestIPv6Addr       [16]byte
	SourceIPv6Prefix   uint8
	DestIPv6Prefix     uint8
	SourcePort         uint16
	DestPort           uint16
	ProtocolIdentifier uint8
	TCPFlags           uint8
	FlowStartMillis    uint32
	FlowEndMillis      uint32
	FlowDirection      uint8
	IPClassOfService   uint8
	FlowEndReason      uint8
}

// GetTemplateFields returns the IPFIX field definitions for the template.
// Field order must match GenericFlow struct field order exactly.
func (gf *GenericFlow) GetTemplateFields() []Field {
	return []Field{
		{Type: InOctets, Length: 4},
		{Type: OutOctets, Length: 4},
		{Type: InPackets, Length: 4},
		{Type: OutPackets, Length: 4},
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
		{Type: FlowStartMilliseconds, Length: 4},
		{Type: FlowEndMilliseconds, Length: 4},
		{Type: FlowDirection, Length: 1},
		{Type: IPClassOfService, Length: 1},
		{Type: FlowEndReason, Length: 1},
	}
}

// Generate creates a GenericFlow with randomly generated data.
// Populates both IPv4 and IPv6 fields based on the input IP version.
func (gf *GenericFlow) Generate(srcIP net.IP, dstIP net.IP, flowSrcPort int, session *netflow.Session) GenericFlow {
	now := time.Now().UnixNano()
	startTime := session.StartTime()
	uptimeMillis := uint32((now-startTime)/int64(time.Millisecond)) + 1000

	gf.InOctets = utils.GenerateRand32(10000)
	gf.OutOctets = utils.GenerateRand32(10000)
	gf.InPackets = utils.GenerateRand32(10000)
	gf.OutPackets = utils.GenerateRand32(10000)

	// Populate IP addresses based on version
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
		gf.SourceIPv6Prefix = 64 // Default /64
		gf.DestIPv6Prefix = 64   // Default /64
	}

	gf.SourcePort = utils.GenerateRand16(10000)
	gf.TCPFlags = uint8(utils.RandomNum(0, 32))
	gf.FlowStartMillis = uptimeMillis - 100
	gf.FlowEndMillis = uptimeMillis - 10
	gf.FlowDirection = 0
	gf.IPClassOfService = 0
	gf.FlowEndReason = uint8(utils.RandomNum(0, 4)) // 0=active, 1=idle, 2=other, 3=exporterReset, 4=exporterShutdown

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
// If profile is nil, defaults to GenericIPFIXProfile for backward compatibility.
func (d *DataFlowSet) Generate(flowCount int, srcRange string, dstRange string, flowSrcPort int, session *netflow.Session, profile ...IPFIXFlowProfile) (DataFlowSet, error) {
	_ = profile // reserved for future profile-aware data generation

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

	return DataFlowSet{
		FlowSetID: 256, // Must match TemplateID
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

// ToBytes serializes the IPFIX structure to a byte buffer for wire transmission.
func (f *IPFIX) ToBytes() bytes.Buffer {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, f.Header); err != nil {
		log.Printf("[ERROR] Issue writing IPFIX header: %v", err)
	}

	for _, tFlow := range f.TemplateFlowSets {
		if err := binary.Write(&buf, binary.BigEndian, tFlow.FlowSetID); err != nil {
			log.Printf("[ERROR] Issue writing IPFIX Template FlowSetID: %v", err)
		}
		if err := binary.Write(&buf, binary.BigEndian, tFlow.Length); err != nil {
			log.Printf("[ERROR] Issue writing IPFIX Template Length: %v", err)
		}
		for _, template := range tFlow.Templates {
			if err := binary.Write(&buf, binary.BigEndian, template.TemplateID); err != nil {
				log.Printf("[ERROR] Issue writing IPFIX Template ID: %v", err)
			}
			if err := binary.Write(&buf, binary.BigEndian, template.FieldCount); err != nil {
				log.Printf("[ERROR] Issue writing IPFIX Template FieldCount: %v", err)
			}
			for _, field := range template.Fields {
				if err := binary.Write(&buf, binary.BigEndian, field.Type); err != nil {
					log.Printf("[ERROR] Issue writing IPFIX Field Type: %v", err)
				}
				if err := binary.Write(&buf, binary.BigEndian, field.Length); err != nil {
					log.Printf("[ERROR] Issue writing IPFIX Field Length: %v", err)
				}
			}
		}
		// Padding to 32-bit boundary per IPFIX RFC 7011
		if tFlow.Padding > 0 {
			padBytes := bytes.Repeat([]byte{0}, tFlow.Padding)
			if err := binary.Write(&buf, binary.BigEndian, padBytes); err != nil {
				log.Printf("[ERROR] Issue writing IPFIX Template Padding: %v", err)
			}
		}
	}

	// Serialize Options Template FlowSets
	for _, oFlow := range f.OptionsTemplateFlowSets {
		if err := binary.Write(&buf, binary.BigEndian, oFlow.FlowSetID); err != nil {
			log.Printf("[ERROR] Issue writing IPFIX Options Template FlowSetID: %v", err)
		}
		if err := binary.Write(&buf, binary.BigEndian, oFlow.Length); err != nil {
			log.Printf("[ERROR] Issue writing IPFIX Options Template Length: %v", err)
		}
		t := oFlow.Template
		if err := binary.Write(&buf, binary.BigEndian, t.TemplateID); err != nil {
			log.Printf("[ERROR] Issue writing IPFIX Options Template ID: %v", err)
		}
		if err := binary.Write(&buf, binary.BigEndian, t.ScopeFieldCount); err != nil {
			log.Printf("[ERROR] Issue writing IPFIX Options ScopeFieldCount: %v", err)
		}
		for _, field := range t.ScopeFields {
			if err := binary.Write(&buf, binary.BigEndian, field.Type); err != nil {
				log.Printf("[ERROR] Issue writing IPFIX Options Scope Field: %v", err)
			}
			if err := binary.Write(&buf, binary.BigEndian, field.Length); err != nil {
				log.Printf("[ERROR] Issue writing IPFIX Options Scope Field Length: %v", err)
			}
		}
		if err := binary.Write(&buf, binary.BigEndian, t.DataFieldCount); err != nil {
			log.Printf("[ERROR] Issue writing IPFIX Options DataFieldCount: %v", err)
		}
		for _, field := range t.DataFields {
			if err := binary.Write(&buf, binary.BigEndian, field.Type); err != nil {
				log.Printf("[ERROR] Issue writing IPFIX Options Data Field: %v", err)
			}
			if err := binary.Write(&buf, binary.BigEndian, field.Length); err != nil {
				log.Printf("[ERROR] Issue writing IPFIX Options Data Field Length: %v", err)
			}
		}
		// Padding to 32-bit boundary per IPFIX RFC 7011
		if oFlow.Padding > 0 {
			padBytes := bytes.Repeat([]byte{0}, oFlow.Padding)
			if err := binary.Write(&buf, binary.BigEndian, padBytes); err != nil {
				log.Printf("[ERROR] Issue writing IPFIX Options Template Padding: %v", err)
			}
		}
	}

	for _, dFlow := range f.DataFlowSets {
		if err := binary.Write(&buf, binary.BigEndian, dFlow.FlowSetID); err != nil {
			log.Printf("[ERROR] Issue writing IPFIX Data FlowSetID: %v", err)
		}
		if err := binary.Write(&buf, binary.BigEndian, dFlow.Length); err != nil {
			log.Printf("[ERROR] Issue writing IPFIX Data FlowSet Length: %v", err)
		}
		for _, item := range dFlow.Items {
			if err := binary.Write(&buf, binary.BigEndian, item); err != nil {
				log.Printf("[ERROR] Issue writing IPFIX Data FlowSet Field: %v", err)
			}
		}
		// Padding to 32-bit boundary per IPFIX RFC 7011
		if dFlow.Padding > 0 {
			padBytes := bytes.Repeat([]byte{0}, dFlow.Padding)
			if err := binary.Write(&buf, binary.BigEndian, padBytes); err != nil {
				log.Printf("[ERROR] Issue writing IPFIX Data Padding: %v", err)
			}
		}
	}

	// Serialize Options Data FlowSets
	for _, oData := range f.OptionsDataFlowSets {
		if err := binary.Write(&buf, binary.BigEndian, oData.FlowSetID); err != nil {
			log.Printf("[ERROR] Issue writing IPFIX Options Data FlowSetID: %v", err)
		}
		if err := binary.Write(&buf, binary.BigEndian, oData.Length); err != nil {
			log.Printf("[ERROR] Issue writing IPFIX Options Data Length: %v", err)
		}
		for _, rec := range oData.Records {
			// Scope values: ObservationDomainId
			if err := binary.Write(&buf, binary.BigEndian, rec.ObservationDomainId); err != nil {
				log.Printf("[ERROR] Issue writing IPFIX Options Scope Value: %v", err)
			}
			// Data values: ProcessName (variable-length with 2-byte length prefix) + ProcessId
			if err := binary.Write(&buf, binary.BigEndian, uint16(len(rec.ProcessName))); err != nil {
				log.Printf("[ERROR] Issue writing IPFIX Options ProcessName length: %v", err)
			}
			if err := binary.Write(&buf, binary.BigEndian, []byte(rec.ProcessName)); err != nil {
				log.Printf("[ERROR] Issue writing IPFIX Options ProcessName: %v", err)
			}
			if err := binary.Write(&buf, binary.BigEndian, rec.ProcessId); err != nil {
				log.Printf("[ERROR] Issue writing IPFIX Options ProcessId: %v", err)
			}
		}
		// Padding to 32-bit boundary per IPFIX RFC 7011
		if oData.Padding > 0 {
			padBytes := bytes.Repeat([]byte{0}, oData.Padding)
			if err := binary.Write(&buf, binary.BigEndian, padBytes); err != nil {
				log.Printf("[ERROR] Issue writing IPFIX Options Data Padding: %v", err)
			}
		}
	}

	return buf
}

// IsValidIPFIX checks whether the given payload has a valid IPFIX header (version 10).
func IsValidIPFIX(payload []byte) (bool, error) {
	header := Header{}
	reader := bytes.NewReader(payload)
	if err := binary.Read(reader, binary.BigEndian, &header); err != nil {
		return false, err
	}
	if header.Version != Version {
		return false, fmt.Errorf("header version is %d, expected IPFIX version %d", header.Version, Version)
	}
	return true, nil
}

// UpdateTimeStamp updates the UnixSec timestamp in an IPFIX packet header to the current time.
func UpdateTimeStamp(payload []byte) ([]byte, error) {
	header := Header{}
	reader := bytes.NewReader(payload)
	if err := binary.Read(reader, binary.BigEndian, &header); err != nil {
		return nil, err
	}
	remainder := make([]byte, len(payload)-20)
	if err := binary.Read(reader, binary.BigEndian, &remainder); err != nil {
		return nil, err
	}

	now := time.Now().UnixNano()
	header.UnixSec = uint32(now / int64(time.Second))

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, header); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.BigEndian, remainder); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GenerateTemplateIPFIX creates an IPFIX packet containing both data and options templates.
func GenerateTemplateIPFIX(sourceID int, session *netflow.Session) IPFIX {
	templateFlow := new(TemplateFlowSet).Generate(session)
	optionsTemplate := new(OptionsTemplateFlowSet).Generate(session)
	header := new(Header).Generate(1, sourceID, session)
	return IPFIX{
		Header:                  header,
		TemplateFlowSets:        []TemplateFlowSet{templateFlow},
		OptionsTemplateFlowSets: []OptionsTemplateFlowSet{optionsTemplate},
		DataFlowSets:            nil,
		OptionsDataFlowSets:     nil,
	}
}

// GenerateOptionsDataIPFIX creates an IPFIX packet containing Options Data records.
// This exports process metadata (process name and PID) to the collector.
func GenerateOptionsDataIPFIX(sourceID int, session *netflow.Session) IPFIX {
	// Use the actual process name and PID
	processName := "flowgre-generator"
	pid := uint32(os.Getpid())

	optionsData := new(OptionsDataFlowSet).Generate(sourceID, processName, pid)
	header := new(Header).Generate(1, sourceID, session)
	return IPFIX{
		Header:              header,
		OptionsDataFlowSets: []OptionsDataFlowSet{optionsData},
	}
}

// GenerateDataIPFIX creates an IPFIX packet containing only data FlowSets.
func GenerateDataIPFIX(flowCount int, sourceID int, srcRange string, dstRange string, flowSrcPort int, session *netflow.Session) (IPFIX, error) {
	dataFlow, err := new(DataFlowSet).Generate(flowCount, srcRange, dstRange, flowSrcPort, session)
	if err != nil {
		return IPFIX{}, fmt.Errorf("generate data flow set: %w", err)
	}
	header := new(Header).Generate(1, sourceID, session)
	return IPFIX{
		Header:       header,
		DataFlowSets: []DataFlowSet{dataFlow},
	}, nil
}

// GenerateIPFIX creates an IPFIX packet containing both template and data FlowSets.
func GenerateIPFIX(flowCount int, sourceID int, srcRange string, dstRange string, session *netflow.Session) (IPFIX, error) {
	templateFlow := new(TemplateFlowSet).Generate(session)
	dataFlow, err := new(DataFlowSet).Generate(flowCount, srcRange, dstRange, utils.HTTPSPort, session)
	if err != nil {
		return IPFIX{}, fmt.Errorf("generate data flow set: %w", err)
	}
	header := new(Header).Generate(flowCount+1, sourceID, session)
	return IPFIX{
		Header:           header,
		TemplateFlowSets: []TemplateFlowSet{templateFlow},
		DataFlowSets:     []DataFlowSet{dataFlow},
	}, nil
}

// size returns the size of the Header in bytes.
func (h *Header) size() int {
	return binary.Size(h.Version) +
		binary.Size(h.FlowCount) +
		binary.Size(h.SysUptime) +
		binary.Size(h.UnixSec) +
		binary.Size(h.FlowSequence) +
		binary.Size(h.SourceID)
}

// String returns a human-readable representation of the Header.
func (h *Header) String() string {
	return fmt.Sprintf("Version: %d Count: %d SysUptime: %d UnixSec: %d FlowSequence: %d SourceID: %d",
		h.Version, h.FlowCount, h.SysUptime, h.UnixSec, h.FlowSequence, h.SourceID)
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
