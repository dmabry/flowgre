// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Single is used to send a single stream of a given number of netflow packets towards a single collector

package single

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/dmabry/flowgre/lifecycle"
	"github.com/dmabry/flowgre/netflow"
	"github.com/dmabry/flowgre/utils"
)

const (
	// sourcePortMin/Max define the range for random source ports
	sourcePortMin = 10000
	sourcePortMax = 15000
	// sourceIDMin/Max define the range for random source IDs
	sourceIDMin = 100
	sourceIDMax = 10000
)

// RunCtx creates the given number of Netflow packets, including the required
// Template, for a Single run with an external context. Cancelling ctx stops
// packet generation cleanly. Use Run() for CLI usage where OS signal handling
// is desired.
func RunCtx(ctx context.Context, collectorIP string, destPort int, srcPort int, count int, srcRange string, dstRange string, hexDump bool) error {
	// Configure connection to use. It looks like a listener, but it will be used to send packet. Allows setting the source port.
	if srcPort == 0 {
		// Pick random source port between 10000 and 15000
		var err error
		srcPort, err = utils.RandomNum(sourcePortMin, sourcePortMax)
		if err != nil {
			return fmt.Errorf("generate source port: %w", err)
		}
	} // else use the given srcPort number
	// Generate random sourceID for all Netflow headers. This is essentially a virtual ID.
	sourceID, err := utils.RandomNum(sourceIDMin, sourceIDMax)
	if err != nil {
		return fmt.Errorf("generate source ID: %w", err)
	}

	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: srcPort})
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer conn.Close()
	// Convert given IP String to net.IP type
	destIP := net.ParseIP(collectorIP)
	if destIP == nil {
		return fmt.Errorf("failed to parse destination IP %s", collectorIP)
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
	_, err = utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: destPort}, tBuf.Bytes(), true)
	if err != nil {
		return fmt.Errorf("flowgre had an issue sending packet: %w", err)
	}

	// Generate and send Data Flow(s)
	fmt.Printf("\nSending Data Flows\n\n")
	for i := 1; i <= count; i++ {
		select {
		case <-ctx.Done():
			log.Printf("Single run cancelled after %d/%d packets\n", i-1, count)
			return nil
		default:
		}
		flow, err := netflow.GenerateDataNetflow(10, sourceID, srcRange, dstRange, 0, session)
		if err != nil {
			return fmt.Errorf("GenerateDataNetflow failed: %w", err)
		}
		buf := flow.ToBytes()
		fmt.Println(netflow.GetNetFlowSizes(flow))
		if hexDump {
			fmt.Printf("%s", hex.Dump(buf.Bytes()))
		}
		_, err = utils.SendPacket(conn, &net.UDPAddr{IP: destIP, Port: destPort}, buf.Bytes(), true)
		if err != nil {
			return fmt.Errorf("flowgre had an issue sending packet: %w", err)
		}
	}
	return nil
}

// Run Creates the given number of Netflow packets, including the required
// Template, for a Single run. Creates the packets and puts them on the wire to
// the targeted host. Sets up OS signal handling (SIGINT/SIGTERM) for clean
// shutdown. Use RunCtx() when you need to control the lifecycle via context.
func Run(collectorIP string, destPort int, srcPort int, count int, srcRange string, dstRange string, hexDump bool) {
	mgr := lifecycle.New()
	defer mgr.Cancel()

	// Setup signal handling via lifecycle manager
	_ = mgr.SetupSignalHandler()

	if err := RunCtx(mgr.Context(), collectorIP, destPort, srcPort, count, srcRange, dstRange, hexDump); err != nil {
		fmt.Fprintf(os.Stderr, "single error: %v\n", err)
		os.Exit(1)
	}
	mgr.Wait()
}
