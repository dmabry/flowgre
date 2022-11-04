// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Single is used to send a single stream of a given number of netflow packets towards a single collector

package single

import (
	"encoding/hex"
	"fmt"
	"github.com/dmabry/flowgre/flow/netflow"
	"github.com/dmabry/flowgre/utils"
	"log"
	"net"
)

// Run Creates the given number of Netflow packets, including the required
// Template, for a Single run. Creates the packets and puts the on the wire to
// the targeted host.
func Run(collectorIP string, destPort int, srcPort int, count int, srcRange string, dstRange string, hexDump bool) {
	// Configure connection to use.  It looks like a listener, but it will be used to send packet.  Allows me to set the source port
	if srcPort == 0 {
		//Pick random source port between 10000 and 15000
		srcPort = utils.RandomNum(10000, 15000)
	} // else use the given srcPort number
	// Generate random sourceID for All Netflow headers.  This is essentially a virtual ID.
	sourceID := utils.RandomNum(100, 10000)

	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: srcPort})
	if err != nil {
		log.Fatal("Listen:", err)
	}
	// Convert given IP String to net.IP type
	destIP := net.ParseIP(collectorIP)
	// start new FlowTracker
	ft := new(netflow.FlowTracker).Init()

	// Generate and send Template Flow(s)
	tFlow := netflow.GenerateTemplateNetflow(sourceID, &ft)
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
		flow := netflow.GenerateDataNetflow(10, sourceID, srcRange, dstRange, 0, &ft)
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
