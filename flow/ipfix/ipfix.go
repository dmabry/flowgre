package ipfix

import (
	"encoding/binary"
	"bytes"
	"time"
	"fmt"
	"net"
	"github.com/prometheus/client_golang/prometheus"
)

// Header represents the IPFIX protocol header (RFC 7011)
type Header struct {
	VersionNumber       uint16 // Must be 10 for IPFIX
	Length              uint16
	ExportTime          uint32
	SequenceNumber      uint32 // Incremented per packet
	ObservationDomainID uint32
}

// ToBytes converts the header to its byte representation
func (h *Header) ToBytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, h); err != nil {
		return nil, fmt.Errorf("failed to write IPFIX header: %v", err)
	}
	return buf.Bytes(), nil
}

// IsValid checks if the header contains valid IPFIX values
func (h *Header) IsValid() bool {
	return h.VersionNumber == 10 && h.Length >= 16
}

// FromBytes populates the header from byte data
func (h *Header) FromBytes(data []byte) error {
	if len(data) < 16 {
		return fmt.Errorf("insufficient data for IPFIX header")
	}
	buf := bytes.NewBuffer(data)
	if err := binary.Read(buf, binary.BigEndian, h); err != nil {
		return fmt.Errorf("failed to read IPFIX header: %v", err)
	}
	return nil
}// IPFIXConfig contains configuration for the IPFIX collector
type IPFIXConfig struct {
	Enabled            bool
	Port               int
	TemplateTimeout    time.Duration
	MaxFlowsPerPacket  int
	AllowedEnterprises []uint32
}

// StartIPFIXCollector starts the IPFIX UDP collector
func StartIPFIXCollector(config IPFIXConfig) {
	// Set up UDP address
	addr := fmt.Sprintf(":%d", config.Port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		fmt.Printf("Error resolving UDP address: %v\n", err)
		return
	}

	// Create UDP connection
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Printf("Error creating UDP listener: %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Printf("IPFIX collector started on port %d\n", config.Port)

	// Buffer for incoming packets
	buf := make([]byte, 65535) // Max UDP packet size

	for {
		// Read packet
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Printf("Error reading UDP packet: %v\n", err)
			continue
		}

		// Increment packet counter
		IPFIXPacketsReceived.Inc()

		// Process packet in a goroutine to avoid blocking
		go func(data []byte, addr *net.UDPAddr) {
			// Parse IPFIX header
			var header Header
			if err := header.FromBytes(data[:n]); err != nil {
				fmt.Printf("Error parsing IPFIX header: %v\n", err)
				return
			}

			// Validate enterprise IDs in template records
			if !header.IsValid() {
				fmt.Printf("Invalid IPFIX header from %v\n", addr)
				return
			}

			// Process template records
			// This is a simplified example - actual implementation would parse the entire packet
			// and check all enterprise IDs in template records
			// For this example, we'll assume the template record starts after the header
			if n > 16 {
				// Skip header (16 bytes)
				templateData := data[16:n]
				
				// Parse template records
				// This is a simplified example - actual implementation would parse all records
				var template TemplateRecord
				if err := template.FromBytes(templateData); err != nil {
					fmt.Printf("Error parsing IPFIX template: %v\n", err)
					return
				}

				// Check if enterprise ID is allowed
				allowed := false
				for _, allowedID := range config.AllowedEnterprises {
					for _, field := range template.Fields {
						if field.EnterpriseFlag && field.EnterpriseID == allowedID {
							allowed = true
							break
						}
					}
					if allowed {
						break
					}
				}

				if !allowed && len(config.AllowedEnterprises) > 0 {
					IPFIXInvalidEnterprise.Inc()
					fmt.Printf("Packet from %v contains disallowed enterprise ID\n", addr)
					return
				}
			}

			// Process data flow sets (simplified)
			// In a real implementation, this would parse and process the actual flow data
			// For this example, we'll just acknowledge receipt
			fmt.Printf("Received valid IPFIX packet from %v, length: %d\n", addr, n)
		}(buf[:n], remoteAddr)
	}
}
