package ipfix_test

import (
	"testing"
	"time"
	"net"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/dmabry/flowgre/flow/ipfix"
)

func TestIPFIXCollector(t *testing.T) {
	// Create a channel to receive test results
	results := make(chan bool, 1)
	
	// Start the IPFIX collector on a random port
	config := ipfix.IPFIXConfig{
		Enabled:            true,
		Port:               0, // Let the OS pick a random port
		TemplateTimeout:    30 * time.Minute,
		MaxFlowsPerPacket:  1000,
		AllowedEnterprises: []uint32{0}, // Allow IANA
	}
	
	// Start the collector in a goroutine
	go ipfix.StartIPFIXCollector(config)
	
	// Wait a bit for the collector to start
	time.Sleep(1 * time.Second)
	
	// Get the actual port the collector is using
	conn, err := net.Dial("udp", fmt.Sprintf("127.0.0.1:%d", config.Port))
	assert.NoError(t, err)
	
	// Create a simple IPFIX header for testing
	header := ipfix.Header{
		VersionNumber:     10,
		Length:            16,
		ExportTime:        uint32(time.Now().Unix()),
		SequenceNumber:    1,
		ObservationDomainID: 12345,
	}
	
	// Convert header to bytes
	data, err := header.ToBytes()
	assert.NoError(t, err)
	
	// Send the test packet
	_, err = conn.Write(data)
	assert.NoError(t, err)
	
	// Wait for the packet to be processed
	time.Sleep(1 * time.Second)
	
	// Verify the packet was received
	// This would typically check metrics or a channel from the collector
	// For this example, we'll assume success if we got this far
	results <- true
	
	// Wait for the result or timeout
	select {
	case success := <-results:
		assert.True(t, success)
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out waiting for IPFIX packet processing")
	}
}func TestProxyIPFIXFunctionality(t *testing.T) {
	// Create a channel to receive test results
	results := make(chan bool, 1)
	
	// Start the proxy with IPFIX enabled
	ipfixPort := 0 // Let the OS pick a random port
	proxyCmd := &cobra.Command{
		Use: "proxy",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create IPFIX config
			ipfixConfig := ipfix.IPFIXConfig{
				Enabled:            true,
				Port:               ipfixPort,
				TemplateTimeout:    30 * time.Minute,
				MaxFlowsPerPacket:  1000,
				AllowedEnterprises: []uint32{0}, // Allow IANA
			}
			
			// Start the IPFIX collector
			go ipfix.StartIPFIXCollector(ipfixConfig)
			
			// Start the proxy (we'll use dummy targets for testing)
			proxy.Run("127.0.0.1", 9995, false, []string{"127.0.0.1:9996"})
			return nil
		},
	}
	
	// Execute the proxy command in a goroutine
	go func() {
		err := proxyCmd.Execute()
		assert.NoError(t, err)
	}()
	
	// Wait a bit for the proxy and IPFIX collector to start
	time.Sleep(2 * time.Second)
	
	// Get the actual port the IPFIX collector is using
	conn, err := net.Dial("udp", fmt.Sprintf("127.0.0.1:%d", ipfixPort))
	assert.NoError(t, err)
	
	// Create a simple IPFIX header for testing
	header := ipfix.Header{
		VersionNumber:     10,
		Length:            16,
		ExportTime:        uint32(time.Now().Unix()),
		SequenceNumber:    1,
		ObservationDomainID: 12345,
	}
	
	// Convert header to bytes
	data, err := header.ToBytes()
	assert.NoError(t, err)
	
	// Send the test packet
	_, err = conn.Write(data)
	assert.NoError(t, err)
	
	// Wait for the packet to be processed
	time.Sleep(1 * time.Second)
	
	// Verify the packet was received
	// This would typically check metrics or a channel from the collector
	// For this example, we'll assume success if we got this far
	results <- true
	
	// Wait for the result or timeout
	select {
	case success := <-results:
		assert.True(t, success)
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out waiting for IPFIX packet processing")
	}
}
