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
	"time"

	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/utils"
)

// IPFIX version number per RFC 7011.
const Version = 10

// Port constants (aliases for backwards compatibility within this package)
const (
	httpsPort    = utils.HTTPSPort
	sshPort      = utils.SSHPort
	ftpPort      = utils.FTPPort
	dnsPort      = utils.DNSPort
	httpPort     = utils.HTTPPort
	ntpPort      = utils.NTPPort
	snmpPort     = utils.SNMPPort
	imapsPort    = utils.IMAPSPort
	mysqlPort    = utils.MySQLPort
	httpAltPort  = utils.HTTPAltPort
	httpsAltPort = utils.HTTPSAltPort
	p2pPort      = utils.P2PPort
	btPort       = utils.BTPort
)

// Protocol constants (aliases for backwards compatibility within this package)
const (
	tcpProto  = utils.TCPProto
	udpProto  = utils.UDPProto
	icmpProto = utils.ICMPProto
)

// IANA IPFIX field type constants (RFC 7011 / Information Model)
const (
	ProtocolIdentifier       = 4
	SourceTransportPort      = 7
	SourceIPv4Address        = 8
	DestinationTransportPort = 11
	DestinationIPv4Address   = 12
	FlowDirection            = 1024
	InPackets                = 1025
	InOctets                 = 1026
	OutPackets               = 1027
	OutOctets                = 1028
	IPClassOfService         = 3
	TCPFlags                 = 6
	FlowStartMilliseconds    = 152
	FlowEndMilliseconds      = 153
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
func (t *TemplateFlowSet) Generate(session *netflow.Session) TemplateFlowSet {
	gf := new(GenericFlow)
	fields := gf.GetTemplateFields()

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

// GenericFlow represents an IPFIX flow record with IANA field types.
type GenericFlow struct {
	InOctets           uint32
	OutOctets          uint32
	InPackets          uint32
	OutPackets         uint32
	SourceIPv4Addr     uint32
	DestIPv4Addr       uint32
	SourcePort         uint16
	DestPort           uint16
	ProtocolIdentifier uint8
	TCPFlags           uint8
	FlowStartMillis    uint32
	FlowEndMillis      uint32
	FlowDirection      uint8
	IPClassOfService   uint8
}

// GetTemplateFields returns the IPFIX field definitions for the template.
func (gf *GenericFlow) GetTemplateFields() []Field {
	return []Field{
		{Type: InOctets, Length: 4},
		{Type: OutOctets, Length: 4},
		{Type: InPackets, Length: 4},
		{Type: OutPackets, Length: 4},
		{Type: SourceIPv4Address, Length: 4},
		{Type: DestinationIPv4Address, Length: 4},
		{Type: SourceTransportPort, Length: 2},
		{Type: DestinationTransportPort, Length: 2},
		{Type: ProtocolIdentifier, Length: 1},
		{Type: TCPFlags, Length: 1},
		{Type: FlowStartMilliseconds, Length: 4},
		{Type: FlowEndMilliseconds, Length: 4},
		{Type: FlowDirection, Length: 1},
		{Type: IPClassOfService, Length: 1},
	}
}

// Generate creates a GenericFlow with randomly generated data.
func (gf *GenericFlow) Generate(srcIP net.IP, dstIP net.IP, flowSrcPort int, session *netflow.Session) GenericFlow {
	now := time.Now().UnixNano()
	startTime := session.StartTime()
	uptimeMillis := uint32((now-startTime)/int64(time.Millisecond)) + 1000

	gf.InOctets = utils.GenerateRand32(10000)
	gf.OutOctets = utils.GenerateRand32(10000)
	gf.InPackets = utils.GenerateRand32(10000)
	gf.OutPackets = utils.GenerateRand32(10000)
	gf.SourceIPv4Addr = utils.IPToNum(srcIP)
	gf.DestIPv4Addr = utils.IPToNum(dstIP)
	gf.SourcePort = utils.GenerateRand16(10000)
	gf.TCPFlags = uint8(utils.RandomNum(0, 32))
	gf.FlowStartMillis = uptimeMillis - 100
	gf.FlowEndMillis = uptimeMillis - 10
	gf.FlowDirection = 0
	gf.IPClassOfService = 0

	switch flowSrcPort {
	case sshPort:
		gf.DestPort = uint16(sshPort)
		gf.ProtocolIdentifier = tcpProto
	case ftpPort:
		gf.DestPort = uint16(ftpPort)
		gf.ProtocolIdentifier = tcpProto
	case dnsPort:
		gf.DestPort = uint16(dnsPort)
		gf.ProtocolIdentifier = udpProto
	case httpPort:
		gf.DestPort = uint16(httpPort)
		gf.ProtocolIdentifier = tcpProto
	case httpsPort:
		gf.DestPort = uint16(httpsPort)
		gf.ProtocolIdentifier = tcpProto
	case ntpPort:
		gf.DestPort = uint16(ntpPort)
		gf.ProtocolIdentifier = udpProto
	case snmpPort:
		gf.DestPort = uint16(snmpPort)
		gf.ProtocolIdentifier = udpProto
	case imapsPort:
		gf.DestPort = uint16(imapsPort)
		gf.ProtocolIdentifier = tcpProto
	case mysqlPort:
		gf.DestPort = uint16(mysqlPort)
		gf.ProtocolIdentifier = tcpProto
	case httpAltPort:
		gf.DestPort = uint16(httpAltPort)
		gf.ProtocolIdentifier = tcpProto
	case httpsAltPort:
		gf.DestPort = uint16(httpsAltPort)
		gf.ProtocolIdentifier = tcpProto
	case p2pPort:
		gf.DestPort = uint16(p2pPort)
		gf.ProtocolIdentifier = tcpProto
	case btPort:
		gf.DestPort = uint16(btPort)
		gf.ProtocolIdentifier = tcpProto
	default:
		gf.DestPort = uint16(httpsPort)
		gf.ProtocolIdentifier = tcpProto
	}

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
func (d *DataFlowSet) Generate(flowCount int, srcRange string, dstRange string, flowSrcPort int, session *netflow.Session) DataFlowSet {
	protoPorts := []int{21, 22, 53, 80, 443, 123, 161, 993, 3306, 8080, 8443, 6681, 6682}

	items := make([]DataAny, flowCount)
	for i := range flowCount {
		srcIP, err := utils.RandomIP(srcRange)
		if err != nil {
			// Proceed with zero IP on error
			srcIP = net.IP{0, 0, 0, 0}
		}
		dstIP, err := utils.RandomIP(dstRange)
		if err != nil {
			dstIP = net.IP{0, 0, 0, 0}
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
	}
}

// IPFIX is the complete IPFIX export packet structure.
type IPFIX struct {
	Header           Header
	TemplateFlowSets []TemplateFlowSet
	DataFlowSets     []DataFlowSet
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
	binary.Write(&buf, binary.BigEndian, header)
	binary.Write(&buf, binary.BigEndian, remainder)
	return buf.Bytes(), nil
}

// GenerateTemplateIPFIX creates an IPFIX packet containing only a template FlowSet.
func GenerateTemplateIPFIX(sourceID int, session *netflow.Session) IPFIX {
	templateFlow := new(TemplateFlowSet).Generate(session)
	header := new(Header).Generate(1, sourceID, session)
	return IPFIX{
		Header:           header,
		TemplateFlowSets: []TemplateFlowSet{templateFlow},
		DataFlowSets:     nil,
	}
}

// GenerateDataIPFIX creates an IPFIX packet containing only data FlowSets.
func GenerateDataIPFIX(flowCount int, sourceID int, srcRange string, dstRange string, flowSrcPort int, session *netflow.Session) IPFIX {
	dataFlow := new(DataFlowSet).Generate(flowCount, srcRange, dstRange, flowSrcPort, session)
	header := new(Header).Generate(1, sourceID, session)
	return IPFIX{
		Header:       header,
		DataFlowSets: []DataFlowSet{dataFlow},
	}
}

// GenerateIPFIX creates an IPFIX packet containing both template and data FlowSets.
func GenerateIPFIX(flowCount int, sourceID int, srcRange string, dstRange string, session *netflow.Session) IPFIX {
	templateFlow := new(TemplateFlowSet).Generate(session)
	dataFlow := new(DataFlowSet).Generate(flowCount, srcRange, dstRange, httpsPort, session)
	header := new(Header).Generate(flowCount+1, sourceID, session)
	return IPFIX{
		Header:           header,
		TemplateFlowSets: []TemplateFlowSet{templateFlow},
		DataFlowSets:     []DataFlowSet{dataFlow},
	}
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
