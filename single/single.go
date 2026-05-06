// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Single is used to send a single stream of a given number of netflow packets towards a single collector

package single

import (
	"encoding/hex"
	"fmt"
	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/utils"
	"log"
	"net"
)

const (
	// sourcePortMin/Max define the range for random source ports
	sourcePortMin = 10000
	sourcePortMax = 15000
	// sourceIDMin/Max define the range for random source IDs
	sourceIDMin = 100
	sourceIDMax = 10000
)

// Run Creates the given number of Netflow packets, including the required
// Template, for a Single run. Creates the packets and puts them on the wire to
// the targeted host.
func Run(collectorIP string, destPort int, srcPort int, count int, srcRange string, dstRange string, hexDump bool) {
	// Configure connection to use. It looks like a listener, but it will be used to send packet. Allows setting the source port.
	if srcPort == 0 {
		// Pick random source port between 10000 and 15000
		srcPort = utils.RandomNum(sourcePortMin, sourcePortMax)
	} // else use the given srcPort number
	// Generate random sourceID for all Netflow headers. This is essentially a virtual ID.
	sourceID := utils.RandomNum(sourceIDMin, sourceIDMax)

	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: srcPort})
	if err != nil {
		log.Fatal("Listen:", err)
	}
	// Convert given IP String to net.IP type
	destIP := net.ParseIP(collectorIP)
	if destIP == nil {
		log.Fatalf("Failed to parse destination IP %s", collectorIP)
	}
	// Create new session for flow generation
	session := netflow.NewSession()

	// Generate and send first Template Flow(s)
	tFlow := netflow.GenerateTemplateNetflow(sourceID, session)
	tBuf := tFlow.ToBytes()
	fmt.Printf("\nSending Template Flow\n\n")
	fmt.Println(netflow.GetNetFlowSizes(tFlow))
	if hexDump {
		fmt.Printf("%s", hex.Dump(tBuf.Bytes()))
	}
	_, err = utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: destPort}, tBuf, true)
	if err != nil {
		log.Fatalf("Flowgre had an issue sending packet %v\n", err)
	}

	// Generate and send Data Flow(s)
	fmt.Printf("\nSending Data Flows\n\n")
	for i := 1; i <= count; i++ {
		flow := netflow.GenerateDataNetflow(10, sourceID, srcRange, dstRange, 0, session)
		buf := flow.ToBytes()
		fmt.Println(netflow.GetNetFlowSizes(flow))
		if hexDump {
			fmt.Printf("%s", hex.Dump(buf.Bytes()))
		}
		_, err = utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: destPort}, buf, true)
		if err != nil {
			log.Fatalf("Flowgre had an issue sending packet %v\n", err)
		}
	}
}
