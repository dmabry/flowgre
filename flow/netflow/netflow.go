// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Netflow v9 funcs and structs used for generating netflow packet to be put on the wire

package netflow

import (
	"bytes"
	"encoding/binary"
	"github.com/dmabry/flowgre/utils"
	"log"
	"net"
	"strconv"
	"time"
)

// StartTime Start time for this instance, used to compute sysUptime
var StartTime = time.Now().UnixNano()

// Current sysUptime in msec
var sysUptime uint32 = 0

// Counter of flow packets
var flowSequence uint32 = 0

// Constants for ports
const (
	ftpPort      = 21
	sshPort      = 22
	dnsPort      = 53
	httpPort     = 80
	httpsPort    = 443
	ntpPort      = 123
	snmpPort     = 161
	imapsPort    = 993
	mysqlPort    = 3306
	httpsAltPort = 8080
	p2pPort      = 6681
	btPort       = 6682
)

// Constants for protocols
const (
	tcpProto = 6
	udpProto = 17
)

// Constants for Field Types
const (
	IN_BYTES                     = 1
	IN_PKTS                      = 2
	FLOWS                        = 3
	PROTOCOL                     = 4
	SRC_TOS                      = 5
	TCP_FLAGS                    = 6
	L4_SRC_PORT                  = 7
	IPV4_SRC_ADDR                = 8
	SRC_MASK                     = 9
	INPUT_SNMP                   = 10
	L4_DST_PORT                  = 11
	IPV4_DST_ADDR                = 12
	DST_MASK                     = 13
	OUTPUT_SNMP                  = 14
	IPV4_NEXT_HOP                = 15
	SRC_AS                       = 16
	DST_AS                       = 17
	BGP_IPV4_NEXT_HOP            = 18
	MUL_DST_PKTS                 = 19
	MUL_DST_BYTES                = 20
	LAST_SWITCHED                = 21
	FIRST_SWITCHED               = 22
	OUT_BYTES                    = 23
	OUT_PKTS                     = 24
	MIN_PKT_LNGTH                = 25
	MAX_PKT_LNGTH                = 26
	IPV6_SRC_ADDR                = 27
	IPV6_DST_ADDR                = 28
	IPV6_SRC_MASK                = 29
	IPV6_DST_MASK                = 30
	IPV6_FLOW_LABEL              = 31
	ICMP_TYPE                    = 32
	MUL_IGMP_TYPE                = 33
	SAMPLING_INTERVAL            = 34
	SAMPLING_ALGORITHM           = 35
	FLOW_ACTIVE_TIMEOUT          = 36
	FLOW_INACTIVE_TIMEOUT        = 37
	ENGINE_TYPE                  = 38
	ENGINE_ID                    = 39
	TOTAL_BYTES_EXP              = 40
	TOTAL_PKTS_EXP               = 41
	TOTAL_FLOWS_EXP              = 42
	IPV4_SRC_PREFIX              = 44
	IPV4_DST_PREFIX              = 45
	MPLS_TOP_LABEL_TYPE          = 46
	MPLS_TOP_LABEL_IP_ADDR       = 47
	FLOW_SAMPLER_ID              = 48
	FLOW_SAMPLER_MODE            = 49
	FLOW_SAMPLER_RANDOM_INTERVAL = 50
	MIN_TTL                      = 52
	MAX_TTL                      = 53
	IPV4_IDENT                   = 54
	DST_TOS                      = 55
	IN_SRC_MAC                   = 56
	OUT_DST_MAC                  = 57
	SRC_VLAN                     = 58
	DST_VLAN                     = 59
	IP_PROTOCOL_VERSION          = 60
	DIRECTION                    = 61
	IPV6_NEXT_HOP                = 62
	BGP_IPV6_NEXT_HOP            = 63
	IPV6_OPTION_HEADERS          = 64
	MPLS_LABEL_1                 = 70
	MPLS_LABEL_2                 = 71
	MPLS_LABEL_3                 = 72
	MPLS_LABEL_4                 = 73
	MPLS_LABEL_5                 = 74
	MPLS_LABEL_6                 = 75
	MPLS_LABEL_7                 = 76
	MPLS_LABEL_8                 = 77
	MPLS_LABEL_9                 = 78
	MPLS_LABEL_10                = 79
	IN_DST_MAC                   = 80
	OUT_SRC_MAC                  = 81
	IF_NAME                      = 82
	IF_DESC                      = 83
	SAMPLER_NAME                 = 84
	IN_PERMANENT_BYTES           = 85
	IN_PERMANENT_PKTS            = 86
	FRAGMENT_OFFSET              = 88
	FORWARDING_STATUS            = 89
	MPLS_PAL_RD                  = 90
	MPLS_PREFIX_LEN              = 91
	SRC_TRAFFIC_INDEX            = 92
	DST_TRAFFIC_INDEX            = 93
	APPLICATION_DESCRIPTION      = 94
	APPLICATION_TAG              = 95
	APPLICATION_NAME             = 96
	postipDiffServCodePoint      = 98
	replication_factor           = 99
	layer2packetSectionOffset    = 102
	layer2packetSectionSize      = 103
	layer2packetSectionData      = 104
)

// HttpsFlow is ued to create and generate HTTPS Flows
type HttpsFlow struct {
	InBytes       uint32
	OutBytes      uint32
	InPkts        uint32
	OutPkts       uint32
	Ipv4SrcAddr   uint32
	Ipv4DstAddr   uint32
	L4SrcPort     uint16
	L4DstPort     uint16
	Protocol      uint8
	TcpFlags      uint8
	FirstSwitched uint32
	LastSwitched  uint32
	EngineType    uint8
	EngineID      uint8
}

// GetTemplateFields returns the Fields for the Template to be used.
func (hf *HttpsFlow) GetTemplateFields() []Field {
	fields := make([]Field, 14)
	fields[0] = Field{Type: IN_BYTES, Length: 4}
	fields[1] = Field{Type: OUT_BYTES, Length: 4}
	fields[2] = Field{Type: IN_PKTS, Length: 4}
	fields[3] = Field{Type: OUT_PKTS, Length: 4}
	fields[4] = Field{Type: IPV4_SRC_ADDR, Length: 4}
	fields[5] = Field{Type: IPV4_DST_ADDR, Length: 4}
	fields[6] = Field{Type: L4_SRC_PORT, Length: 2}
	fields[7] = Field{Type: L4_DST_PORT, Length: 2}
	fields[8] = Field{Type: PROTOCOL, Length: 1}
	fields[9] = Field{Type: TCP_FLAGS, Length: 1}
	fields[10] = Field{Type: FIRST_SWITCHED, Length: 4}
	fields[11] = Field{Type: LAST_SWITCHED, Length: 4}
	fields[12] = Field{Type: ENGINE_TYPE, Length: 1}
	fields[13] = Field{Type: ENGINE_ID, Length: 1}
	return fields
}

// Generate returns a HTTPS Flow with randomly generated payload
func (hf *HttpsFlow) Generate(srcIP net.IP, dstIP net.IP, flowTracker *FlowTracker) HttpsFlow {
	now := time.Now().UnixNano()
	startTime := flowTracker.GetStartTime()
	uptime := uint32((now-startTime)/int64(time.Millisecond)) + 1000
	hf.InBytes = utils.GenerateRand32(10000)
	hf.OutBytes = utils.GenerateRand32(10000)
	hf.InPkts = utils.GenerateRand32(10000)
	hf.OutPkts = utils.GenerateRand32(10000)
	hf.Ipv4SrcAddr = utils.IPToNum(srcIP)
	hf.Ipv4DstAddr = utils.IPToNum(dstIP)
	hf.L4SrcPort = utils.GenerateRand16(10000)
	hf.L4DstPort = uint16(httpsPort)
	hf.Protocol = uint8(tcpProto)
	hf.TcpFlags = uint8(utils.RandomNum(0, 32))
	hf.FirstSwitched = uptime - 100
	hf.LastSwitched = uptime - 10
	hf.EngineType = 0
	hf.EngineID = 0

	return *hf
}

// FlowTracker is used to track the start time and the flow sequence
type FlowTracker struct {
	StartTime    int64
	FlowSequence uint32
}

// Init FlowTracker starts a new counter
func (ft *FlowTracker) Init() FlowTracker {
	flowTracker := new(FlowTracker)
	flowTracker.FlowSequence = 0
	flowTracker.StartTime = time.Now().UnixNano()
	return *flowTracker
}

func (ft *FlowTracker) GetStartTime() int64 {
	return ft.StartTime
}

func (ft *FlowTracker) NextSeq() uint32 {
	ft.FlowSequence = ft.FlowSequence + 1
	return ft.FlowSequence
}

// Header NetflowHeader v9
type Header struct {
	Version      uint16
	FlowCount    uint16
	SysUptime    uint32
	UnixSec      uint32
	FlowSequence uint32
	SourceID     uint32
}

// Get the size of the Header in bytes
func (h *Header) size() int {
	size := binary.Size(h.Version)
	size += binary.Size(h.FlowCount)
	size += binary.Size(h.SysUptime)
	size += binary.Size(h.UnixSec)
	size += binary.Size(h.FlowSequence)
	size += binary.Size(h.SourceID)
	return size
}

// Get the Header in String
func (h *Header) String() string {
	return "Version: " + strconv.Itoa(int(h.Version)) +
		" Count: " + strconv.Itoa(int(h.FlowCount)) +
		" SysUptime: " + strconv.Itoa(int(h.SysUptime)) +
		" UnixSec: " + strconv.Itoa(int(h.UnixSec)) +
		" FlowSequence: " + strconv.Itoa(int(h.FlowSequence)) +
		" SourceID: " + strconv.Itoa(int(h.SourceID)) +
		" || "
}

// Generate a Header accounting for the given flowCount.  Flowcount should match the expected number of flows in the
// Netflow packet that the Header will be used for.
func (h *Header) Generate(flowSetCount int, sourceID int, flowTracker *FlowTracker) Header {
	now := time.Now().UnixNano()
	secs := now / int64(time.Second)
	startTime := flowTracker.GetStartTime()
	sysUptime = uint32((now-startTime)/int64(time.Millisecond)) + 1000

	header := new(Header)
	header.Version = 9
	header.SysUptime = sysUptime
	header.UnixSec = uint32(secs)
	header.FlowCount = uint16(flowSetCount)
	header.FlowSequence = flowTracker.NextSeq()
	header.SourceID = uint32(sourceID)

	return *header
}

// Field for Template struct
type Field struct {
	Type   uint16
	Length uint16
}

// Get the Field in String
func (f *Field) String() string {
	return "Type: " + strconv.Itoa(int(f.Type)) + "Length: " + strconv.Itoa(int(f.Length))
}

// Template for TemplateFlowSet
type Template struct {
	TemplateID uint16 // 0-255
	FieldCount uint16
	Fields     []Field
}

// Get the size of the Template in bytes
func (t *Template) size() int {
	size := binary.Size(t.TemplateID)
	size += binary.Size(t.FieldCount)
	for _, field := range t.Fields {
		size += binary.Size(field)
	}
	return size
}

// Get the size of the Fields in a given Template in bytes
func (t *Template) sizeOfFields() int {
	var size int
	for _, field := range t.Fields {
		size += int(field.Length)
	}
	return size
}

// TemplateFlowSet for Netflow
type TemplateFlowSet struct {
	FlowSetID uint16 // seems to always be 0???
	Length    uint16
	Templates []Template
}

// Generate a TemplateFlowSet.
// Per Netflow v9 spec, FlowSetID is *always* 0 for a TemplateFlow.
// Hardcoded TemplateID to 256, but could be variable as long as it is greater than 255
// TODO: Hardcoded FieldCount and Fields for HTTPS Flow.  Need to work on Generating different flows
func (t *TemplateFlowSet) Generate() TemplateFlowSet {
	templateFlowSet := new(TemplateFlowSet)
	templateFlowSet.FlowSetID = 0
	var templates []Template
	// template
	template := new(Template)
	fields := new(HttpsFlow).GetTemplateFields()
	template.TemplateID = 256
	template.FieldCount = uint16(len(fields))
	// add fields to the template
	template.Fields = fields
	templates = append(templates, *template)
	templateFlowSet.Templates = templates
	templateFlowSet.Length += uint16(templateFlowSet.size())
	return *templateFlowSet
}

// Get the size of the TemplateFlowSet in bytes
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
	return size
}

type DataItem struct {
	Fields []uint32
}
type DataAny interface {
}

// DataFlowSet for Netflow
type DataFlowSet struct {
	FlowSetID uint16
	Length    uint16
	Items     []DataAny
	Padding   int
}

// Generate a DataFlowSet.
// Per Netflow v9 spec, FlowSetID is *always* set to the TemplateID from a given TemplateFlowSet.
// Hardcoded TemplateID to 256, but could be variable as long as it is greater than 255
// Currently hardcoded to generate random src/dst IPs from 10.0.0.0/8.
// TODO: Modify src/dst IP handling to allow for passing of values
// TODO: Currently hardcoded to be a HTTPS flow.
func (d *DataFlowSet) Generate(flowCount int, srcRange string, dstRange string, flowTracker *FlowTracker) DataFlowSet {
	dataFlowSet := new(DataFlowSet)
	dataFlowSet.FlowSetID = 256
	items := make([]DataAny, flowCount)
	for i := 0; i < flowCount; i++ {
		srcIP, err := utils.RandomIP(srcRange)
		if err != nil {
			log.Printf("Issue generating IP... proceeding anyway: %v", err)
		}
		dstIP, err := utils.RandomIP(dstRange)
		if err != nil {
			log.Printf("Issue generating IP... proceeding anyway: %v", err)
		}
		hf := new(HttpsFlow)
		items[i] = hf.Generate(srcIP, dstIP, flowTracker)
	}
	dataFlowSet.Items = items
	dataFlowSet.Length = uint16(dataFlowSet.size())
	return *dataFlowSet
}

// Get the size of the DataFlowSet in bytes
func (d *DataFlowSet) size() int {
	padding := 0
	size := binary.Size(d.FlowSetID)
	size += binary.Size(d.Length)
	for _, item := range d.Items {
		size += binary.Size(item)
	}
	remainder := size % 32
	if remainder > 0 {
		padding = 32 - remainder
	}
	size += padding     // number of uint8 to pad in order to reach 32 bit boundary
	d.Padding = padding // save the padding as an int for later.
	return size
}

// Netflow complete record
type Netflow struct {
	Header           Header
	TemplateFlowSets []TemplateFlowSet
	DataFlowSets     []DataFlowSet
}

// ToBytes Converts Netflow struct to a bytes buffer than can be written to the wire
func (n *Netflow) ToBytes() bytes.Buffer {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, &n.Header)
	if err != nil {
		log.Println("[ERROR] Issue writing header: ", err)
	}
	// Write Template flow if any exists
	if len(n.TemplateFlowSets) > 0 {
		for _, tFlow := range n.TemplateFlowSets {
			// Order FlowSetID, Length, Template(s)
			err := binary.Write(&buf, binary.BigEndian, tFlow.FlowSetID)
			if err != nil {
				log.Println("[ERROR] Issue writing Template FlowSetID: ", err)
			}
			err = binary.Write(&buf, binary.BigEndian, tFlow.Length)
			if err != nil {
				log.Println("[ERROR] Issue writing Template Length: ", err)
			}
			for _, template := range tFlow.Templates {
				// Order TemplateId, Field Count, Field(s)
				err = binary.Write(&buf, binary.BigEndian, template.TemplateID)
				if err != nil {
					log.Println("[ERROR] Issue writing Template ID: ", err)
				}
				err = binary.Write(&buf, binary.BigEndian, template.FieldCount)
				if err != nil {
					log.Println("[ERROR] Issue writing Template FieldCount: ", err)
				}
				for _, field := range template.Fields {
					err = binary.Write(&buf, binary.BigEndian, field.Type)
					if err != nil {
						log.Println("[ERROR} Issue writing Field Type: ", err)
					}
					err = binary.Write(&buf, binary.BigEndian, field.Length)
					if err != nil {
						log.Println("[ERROR} Issue writing Field Length: ", err)
					}
				}
			}
		}
	}
	// Write Data flow(s) if any exists
	if len(n.DataFlowSets) > 0 {
		for _, dFlow := range n.DataFlowSets {
			// Order FlowSetID, Length, Record(s)
			err := binary.Write(&buf, binary.BigEndian, dFlow.FlowSetID)
			if err != nil {
				log.Println("[ERROR] Issue writing Data FlowSetID: ", err)
			}
			err = binary.Write(&buf, binary.BigEndian, dFlow.Length)
			if err != nil {
				log.Println("[ERROR] Issue writing Data FlowSet Length: ", err)
			}
			for _, item := range dFlow.Items {
				err = binary.Write(&buf, binary.BigEndian, item)
				if err != nil {
					log.Println("[ERROR] Issue writing Data FlowSet Field: ", err)
				}
			}
			// Padding to 32 bit boundary per Netflow v9 RFC
			if dFlow.Padding != 0 {
				padtext := bytes.Repeat([]byte{byte(0)}, dFlow.Padding)
				err = binary.Write(&buf, binary.BigEndian, padtext)
				if err != nil {
					log.Println("[ERROR] Issue writing Data Padding: ", err)
				}
			}
		}
	}
	return buf
}

// GetNetFlowSizes Gets the size of a given Netflow and returns it as a String
func GetNetFlowSizes(netFlow Netflow) string {
	output := "Header Size: " + strconv.Itoa(netFlow.Header.size()) + " bytes\n"
	tSize := 0
	dSize := 0
	for _, tFlow := range netFlow.TemplateFlowSets {
		tSize += tFlow.size()
	}
	output += "Template Size: " + strconv.Itoa(tSize) + " bytes\n"
	for _, dFlow := range netFlow.DataFlowSets {
		dSize += dFlow.size()
	}
	output += "Data Size: " + strconv.Itoa(dSize) + " bytes\n"
	return output
}

// GenerateNetflow Generates a combined Template and Data flow Netflow struct.  Not required by spec, but can be done.
func GenerateNetflow(flowCount int, sourceID int, srcRange string, dstRange string, flowTracker *FlowTracker) Netflow {
	netflow := new(Netflow)
	templateFlow := new(TemplateFlowSet).Generate()
	dataFlow := new(DataFlowSet).Generate(flowCount, srcRange, dstRange, flowTracker)
	header := new(Header).Generate(flowCount+1, sourceID, flowTracker) // always +1 of dataflow count, because we are counting the template
	netflow.Header = header
	netflow.TemplateFlowSets = append(netflow.TemplateFlowSets, templateFlow)
	netflow.DataFlowSets = append(netflow.DataFlowSets, dataFlow)
	return *netflow
}

// GenerateDataNetflow Generates a Netflow containing Data flows
func GenerateDataNetflow(flowCount int, sourceID int, srcRange string, dstRange string, flowTracker *FlowTracker) Netflow {
	netflow := new(Netflow)
	dataFlow := new(DataFlowSet).Generate(flowCount, srcRange, dstRange, flowTracker)
	header := new(Header).Generate(1, sourceID, flowTracker) // always 1 for but could be more in future
	netflow.Header = header
	netflow.DataFlowSets = append(netflow.DataFlowSets, dataFlow)
	return *netflow
}

// GenerateTemplateNetflow Generates a Netflow containing Template flow
func GenerateTemplateNetflow(sourceID int, flowTracker *FlowTracker) Netflow {
	netflow := new(Netflow)
	templateFlow := new(TemplateFlowSet).Generate()
	header := new(Header).Generate(1, sourceID, flowTracker) // always 1 counting the template only
	netflow.Header = header
	netflow.TemplateFlowSets = append(netflow.TemplateFlowSets, templateFlow)
	return *netflow
}
