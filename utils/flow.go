// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package utils

// Well-known port constants used for generating simulated flow traffic.
const (
	FTPPort      = 21
	SSHPort      = 22
	DNSPort      = 53
	HTTPPort     = 80
	HTTPSPort    = 443
	NTPPort      = 123
	SNMPPort     = 161
	IMAPSPort    = 993
	MySQLPort    = 3306
	HTTPAltPort  = 8080
	HTTPSAltPort = 8443
	P2PPort      = 6681
	BTPort       = 6682
)

// Well-known IP protocol constants (RFC 1700 / IANA).
const (
	TCPProto   = 6
	UDPProto   = 17
	ICMPProto  = 1
	SCTPProto  = 132
	IGMPProto  = 2
	EGPProto   = 8
	IGPProto   = 9
	GREProto   = 47
	ESPProto   = 50
	EIGRPProto = 88
)

// ProtoPorts is the default set of destination ports used for generating
// simulated flow traffic. Each entry maps to a well-known service.
var ProtoPorts = []int{21, 22, 53, 80, 443, 123, 161, 993, 3306, 8080, 8443, 6681, 6682}

// ResolvePortProtocol maps a destination port to its well-known port and
// associated IP protocol number. Returns (dstPort, protocol) where protocol
// is one of TCPProto, UDPProto, etc.
//
// This centralizes the port→protocol mapping that was duplicated across
// GenericFlow, MinimalFlow, ExtendedFlow, and IPFIX GenericFlow.
func ResolvePortProtocol(flowPort int) (dstPort uint16, protocol uint8) {
	switch flowPort {
	case FTPPort:
		return uint16(FTPPort), TCPProto
	case SSHPort:
		return uint16(SSHPort), TCPProto
	case DNSPort:
		return uint16(DNSPort), UDPProto
	case HTTPPort:
		return uint16(HTTPPort), TCPProto
	case HTTPSPort:
		return uint16(HTTPSPort), TCPProto
	case NTPPort:
		return uint16(NTPPort), UDPProto
	case SNMPPort:
		return uint16(SNMPPort), UDPProto
	case IMAPSPort:
		return uint16(IMAPSPort), TCPProto
	case MySQLPort:
		return uint16(MySQLPort), TCPProto
	case HTTPAltPort:
		return uint16(HTTPAltPort), TCPProto
	case HTTPSAltPort:
		return uint16(HTTPSAltPort), TCPProto
	case P2PPort:
		return uint16(P2PPort), TCPProto
	case BTPort:
		return uint16(BTPort), TCPProto
	default:
		return uint16(HTTPSPort), TCPProto
	}
}
