// netflow funcs and structs

package netflow

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"time"
)

// StartTime Start time for this instance, used to compute sysUptime
var StartTime = time.Now().UnixNano()

// current sysUptime in msec - recalculated in CreateNFlowHeader()
var sysUptime uint32 = 0

// Counter of flow packets that have been sent
var flowSequence uint32 = 0

// Constants for ports
const (
	ftpPort          = 21
	sshPort          = 22
	dnsPort          = 53
	httpPort         = 80
	httpsPort        = 443
	ntpPort          = 123
	snmpPort         = 161
	imapsPort        = 993
	mysqlPort        = 3306
	httpsAltPort     = 8080
	p2pPort          = 6681
	btPort           = 6682
	uint16Max        = 65535
	payloadAvgMedium = 1024
	payloadAvgSmall  = 256
)

// Constants for Fields
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

// Header NetflowHeader v9
type Header struct {
	Version      uint16
	FlowCount    uint16
	SysUptime    uint32
	UnixSec      uint32
	FlowSequence uint32
	SourceID     uint32
}

func (h *Header) size() int {
	size := binary.Size(h.Version)
	size += binary.Size(h.FlowCount)
	size += binary.Size(h.SysUptime)
	size += binary.Size(h.UnixSec)
	size += binary.Size(h.FlowSequence)
	size += binary.Size(h.SourceID)
	return size
}

func (h *Header) String() string {
	return "Version: " + strconv.Itoa(int(h.Version)) +
		" Count: " + strconv.Itoa(int(h.FlowCount)) +
		" SysUptime: " + strconv.Itoa(int(h.SysUptime)) +
		" UnixSec: " + strconv.Itoa(int(h.UnixSec)) +
		" FlowSequence: " + strconv.Itoa(int(h.FlowSequence)) +
		" SourceID: " + strconv.Itoa(int(h.SourceID)) +
		" || "
}

func (h *Header) Generate(flowCount int) Header {
	now := time.Now().UnixNano()
	secs := now / int64(time.Second)
	sysUptime = uint32((now-StartTime)/int64(time.Millisecond)) + 1000
	flowSequence++

	header := new(Header)
	header.Version = 9
	header.SysUptime = sysUptime
	header.UnixSec = uint32(secs)
	header.FlowCount = uint16(flowCount)
	header.FlowSequence = flowSequence
	header.SourceID = uint32(618)

	return *header
}

type Field struct {
	Type   uint16
	Length uint16
}

func (f *Field) String() string {
	return "Type: " + strconv.Itoa(int(f.Type)) + "Length: " + strconv.Itoa(int(f.Length))
}

type Template struct {
	TemplateID uint16 // 0-255
	FieldCount uint16
	Fields     []Field
}

func (t *Template) size() int {
	size := binary.Size(t.TemplateID)
	size += binary.Size(t.FieldCount)
	for _, field := range t.Fields {
		size += binary.Size(field)
	}
	return size
}

func (t *Template) sizeOfFields() int {
	var size int
	for _, field := range t.Fields {
		size += int(field.Length)
	}
	return size
}

type TemplateFlowSet struct {
	FlowSetID uint16 // seems to always be 0???
	Length    uint16
	Templates []Template
}

func (t *TemplateFlowSet) Generate() TemplateFlowSet {
	templateFlowSet := new(TemplateFlowSet)
	templateFlowSet.FlowSetID = 0
	var templates []Template
	// template
	template := new(Template)
	template.TemplateID = 256
	template.FieldCount = 6
	// fields
	fields := make([]Field, template.FieldCount)
	fields[0] = Field{Type: IN_BYTES, Length: 4}
	fields[1] = Field{Type: IN_PKTS, Length: 4}
	fields[2] = Field{Type: IPV4_SRC_ADDR, Length: 4}
	fields[3] = Field{Type: IPV4_DST_ADDR, Length: 4}
	fields[4] = Field{Type: L4_SRC_PORT, Length: 4}
	fields[5] = Field{Type: L4_DST_PORT, Length: 4}
	// add them to the template
	template.Fields = fields
	// chicken and egg.... need to rethink this TODO: Solve this differently
	templates = append(templates, *template)
	templateFlowSet.Templates = templates
	templateFlowSet.Length += uint16(templateFlowSet.size())
	return *templateFlowSet
}

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

type DataFlowSet struct {
	FlowSetID uint16 // should equal template id previously passed... for generation maybe always use 256?
	Length    uint16
	Items     []DataItem
	Padding   int //used to calculate "pad" the flowset to 32 bit
}

func (d *DataFlowSet) Generate(flowCount int) DataFlowSet {
	dataFlowSet := new(DataFlowSet)
	dataFlowSet.FlowSetID = 256
	// dataFlowSet.Length = 0 // need to figure out how to calculate this
	items := make([]DataItem, flowCount)
	for i := 0; i < flowCount; i++ {
		srcIP, _ := RandomIP("10.0.0.0/8")
		dstIP, _ := RandomIP("10.0.0.0/8")
		fields := make([]uint32, 6)
		//IN_BYTES
		fields[0] = genRand32(10000)
		//IN_PKTS
		fields[1] = genRand32(10000)
		//IPV4_SRC_ADDR
		//fields[2] = IPto32("10.0.0.32")
		fields[2] = ipToNum(srcIP)
		//IPV4_DST_ADDR
		//fields[3] = IPto32("10.0.0.42")
		fields[3] = ipToNum(dstIP)
		//L4_SRC_PORT
		fields[4] = genRand32(10000)
		//L4_DST_PORT
		fields[5] = uint32(httpsPort)
		//add fields to the item
		items[i].Fields = fields
	}
	dataFlowSet.Items = items
	dataFlowSet.Length = uint16(dataFlowSet.size())
	return *dataFlowSet
}

func (d *DataFlowSet) size() int {
	size := binary.Size(d.FlowSetID)
	size += binary.Size(d.Length)
	for _, item := range d.Items {
		for _, field := range item.Fields {
			size += binary.Size(field)
		}
	}
	remainder := size % 32
	padding := 32 - remainder
	size += padding
	d.Padding = padding // save the padding as an int for later.
	return size
}

// Netflow Complete netflow records
type Netflow struct {
	Header           Header
	TemplateFlowSets []TemplateFlowSet
	DataFlowSets     []DataFlowSet
}

func (n *Netflow) ToBytes() bytes.Buffer {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, &n.Header)
	if err != nil {
		log.Println("[ERROR] Issue writing header: ", err)
	}
	//write template flow if any exists
	if len(n.TemplateFlowSets) > 0 {
		for _, tFlow := range n.TemplateFlowSets {
			//order flowsetid, length, template(s)
			err := binary.Write(&buf, binary.BigEndian, tFlow.FlowSetID)
			if err != nil {
				log.Println("[ERROR] Issue writing Template FlowSetID: ", err)
			}
			err = binary.Write(&buf, binary.BigEndian, tFlow.Length)
			if err != nil {
				log.Println("[ERROR] Issue writing Template Length: ", err)
			}
			for _, template := range tFlow.Templates {
				// templateId, Field Count, Field(s)
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
	//write dataflow(s)
	if len(n.DataFlowSets) > 0 {
		for _, dFlow := range n.DataFlowSets {
			// order FlowSetID, Length, Record(s)
			err := binary.Write(&buf, binary.BigEndian, dFlow.FlowSetID)
			if err != nil {
				log.Println("[ERROR] Issue writing Data FlowSetID: ", err)
			}
			err = binary.Write(&buf, binary.BigEndian, dFlow.Length)
			if err != nil {
				log.Println("[ERROR] Issue writing Data FlowSet Length: ", err)
			}
			for _, item := range dFlow.Items {
				for _, field := range item.Fields {
					err = binary.Write(&buf, binary.BigEndian, field)
					if err != nil {
						log.Println("[ERROR] Issue writing Data FlowSet Field: ", err)
					}
				}
			}
			//padding to 32 bit boundary
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

func GenerateNetflow(flowCount int) Netflow {
	netflow := new(Netflow)
	templateFlow := new(TemplateFlowSet).Generate()
	dataFlow := new(DataFlowSet).Generate(flowCount)
	header := new(Header).Generate(flowCount + 1) // always +1 of dataflow count, because we are counting the template
	netflow.Header = header
	netflow.TemplateFlowSets = append(netflow.TemplateFlowSets, templateFlow)
	netflow.DataFlowSets = append(netflow.DataFlowSets, dataFlow)
	return *netflow
}

func GenerateDataNetflow(flowCount int) Netflow {
	netflow := new(Netflow)
	dataFlow := new(DataFlowSet).Generate(flowCount)
	header := new(Header).Generate(flowCount) // always +1 of dataflow count, because we are counting the template
	netflow.Header = header
	netflow.DataFlowSets = append(netflow.DataFlowSets, dataFlow)
	return *netflow
}

func GenerateTemplateNetflow() Netflow {
	netflow := new(Netflow)
	templateFlow := new(TemplateFlowSet).Generate()
	header := new(Header).Generate(1) // always 1 counting the template only
	netflow.Header = header
	netflow.TemplateFlowSets = append(netflow.TemplateFlowSets, templateFlow)
	return *netflow
}

func genRand16(max int) uint16 {
	return uint16(rand.Intn(max))
}

func IPto32(s string) uint32 {
	ip := net.ParseIP(s)
	return binary.BigEndian.Uint32(ip.To4())
}

func genRand32(max int) uint32 {
	return uint32(rand.Intn(max))
}

func randomNum(min, max int) int {
	return rand.Intn(max-min) + min
}

// Not used currently, but handy to have for later maybe
func ToBytes(key interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(key)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func ipToNum(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func numToIP(num uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, num)
	return ip
}

func RandomIP(cidr string) (net.IP, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		fmt.Println("[ERROR] Parsing CIDR", cidr, " failed. error: ", err)
	}
	ipMin := ipNet.IP
	ipMax, _ := getLastIP(ipNet)
	ipMinNum := ipToNum(ipMin)
	ipMaxNum := ipToNum(ipMax)
	rand.Seed(time.Now().UnixNano())
	randIPNum := uint32(rand.Int31n(int32(ipMaxNum-ipMinNum)) + int32(ipMinNum))
	randIP := numToIP(randIPNum)

	//check if in range

	if ipNet.Contains(randIP) {
		return randIP, nil
	}
	return nil, errors.New("random IP broken")

	//fmt.Println("ipMin: ", ipMin.String(), " ipMax: ", ipMax.String())
}

func getLastIP(ipNet *net.IPNet) (net.IP, error) {
	ip := make(net.IP, len(ipNet.IP.To4()))
	binary.BigEndian.PutUint32(ip, binary.BigEndian.Uint32(ipNet.IP.To4())|^binary.BigEndian.Uint32(ipNet.Mask))
	return ip, nil
}
