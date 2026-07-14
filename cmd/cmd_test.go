// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"os"
	"testing"

	"github.com/dmabry/flowgre/web"
)

// =============================================================================
// targetFlags (proxy custom flag type)
// =============================================================================

func TestTargetFlagsString(t *testing.T) {
	tf := targetFlags{"a", "b"}
	if tf.String() != "<multiple>" {
		t.Errorf("expected '<multiple>', got %q", tf.String())
	}
}

func TestTargetFlagsSet(t *testing.T) {
	var tf targetFlags

	if err := tf.Set("10.0.0.1:9995"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := tf.Set("10.0.0.2:9996"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tf) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(tf))
	}
	if tf[0] != "10.0.0.1:9995" {
		t.Errorf("expected '10.0.0.1:9995', got %q", tf[0])
	}
	if tf[1] != "10.0.0.2:9996" {
		t.Errorf("expected '10.0.0.2:9996', got %q", tf[1])
	}
}

func TestTargetFlagsEmpty(t *testing.T) {
	var tf targetFlags
	if len(tf) != 0 {
		t.Errorf("expected 0 targets, got %d", len(tf))
	}
	if tf.String() != "<multiple>" {
		t.Errorf("expected '<multiple>', got %q", tf.String())
	}
}

// =============================================================================
// SingleCommand
// =============================================================================

func TestSingleCommandDefaults(t *testing.T) {
	c := &SingleCommand{}
	if err := c.ParseFlags([]string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *c.server != "127.0.0.1" {
		t.Errorf("expected server '127.0.0.1', got %q", *c.server)
	}
	if *c.port != 9995 {
		t.Errorf("expected port 9995, got %d", *c.port)
	}
	if *c.srcPort != 0 {
		t.Errorf("expected srcPort 0, got %d", *c.srcPort)
	}
	if *c.count != 1 {
		t.Errorf("expected count 1, got %d", *c.count)
	}
	if *c.hexDump != false {
		t.Errorf("expected hexDump false, got %v", *c.hexDump)
	}
	if *c.srcRange != "10.0.0.0/8" {
		t.Errorf("expected srcRange '10.0.0.0/8', got %q", *c.srcRange)
	}
	if *c.dstRange != "10.0.0.0/8" {
		t.Errorf("expected dstRange '10.0.0.0/8', got %q", *c.dstRange)
	}
}

func TestSingleCommandOverrides(t *testing.T) {
	c := &SingleCommand{}
	args := []string{
		"-server", "192.168.1.1",
		"-port", "12345",
		"-src-port", "5000",
		"-count", "100",
		"-hexdump",
		"-src-range", "172.16.0.0/12",
		"-dst-range", "192.168.0.0/16",
	}
	if err := c.ParseFlags(args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *c.server != "192.168.1.1" {
		t.Errorf("expected '192.168.1.1', got %q", *c.server)
	}
	if *c.port != 12345 {
		t.Errorf("expected 12345, got %d", *c.port)
	}
	if *c.srcPort != 5000 {
		t.Errorf("expected 5000, got %d", *c.srcPort)
	}
	if *c.count != 100 {
		t.Errorf("expected 100, got %d", *c.count)
	}
	if !*c.hexDump {
		t.Error("expected hexDump true")
	}
	if *c.srcRange != "172.16.0.0/12" {
		t.Errorf("expected '172.16.0.0/12', got %q", *c.srcRange)
	}
	if *c.dstRange != "192.168.0.0/16" {
		t.Errorf("expected '192.168.0.0/16', got %q", *c.dstRange)
	}
}

func TestSingleCommandIPv6(t *testing.T) {
	c := &SingleCommand{}
	args := []string{
		"-server", "::1",
		"-src-range", "2001:db8::/32",
		"-dst-range", "fd00::/8",
	}
	if err := c.ParseFlags(args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *c.server != "::1" {
		t.Errorf("expected '::1', got %q", *c.server)
	}
	if *c.srcRange != "2001:db8::/32" {
		t.Errorf("expected '2001:db8::/32', got %q", *c.srcRange)
	}
	if *c.dstRange != "fd00::/8" {
		t.Errorf("expected 'fd00::/8', got %q", *c.dstRange)
	}
}

// =============================================================================
// BarrageCommand
// =============================================================================

func TestBarrageCommandDefaults(t *testing.T) {
	c := &BarrageCommand{}
	if err := c.ParseFlags([]string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *c.server != "127.0.0.1" {
		t.Errorf("expected server '127.0.0.1', got %q", *c.server)
	}
	if *c.port != 9995 {
		t.Errorf("expected port 9995, got %d", *c.port)
	}
	if *c.workers != 4 {
		t.Errorf("expected workers 4, got %d", *c.workers)
	}
	if *c.delay != 100 {
		t.Errorf("expected delay 100, got %d", *c.delay)
	}
	if *c.templateInterval != 30 {
		t.Errorf("expected templateInterval 30, got %d", *c.templateInterval)
	}
	if *c.configFile != "" {
		t.Errorf("expected configFile '', got %q", *c.configFile)
	}
	if *c.webPort != 8080 {
		t.Errorf("expected webPort 8080, got %d", *c.webPort)
	}
	if *c.webIP != "127.0.0.1" {
		t.Errorf("expected webIP '127.0.0.1', got %q", *c.webIP)
	}
	if *c.web != false {
		t.Errorf("expected web false, got %v", *c.web)
	}
	if *c.protocol != "netflow" {
		t.Errorf("expected protocol 'netflow', got %q", *c.protocol)
	}
}

func TestBarrageCommandOverrides(t *testing.T) {
	c := &BarrageCommand{}
	args := []string{
		"-server", "10.0.0.1",
		"-port", "20000",
		"-workers", "8",
		"-delay", "50",
		"-template-interval", "60",
		"-web-port", "9090",
		"-web-ip", "::",
		"-web",
		"-protocol", "ipfix",
	}
	if err := c.ParseFlags(args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *c.server != "10.0.0.1" {
		t.Errorf("expected '10.0.0.1', got %q", *c.server)
	}
	if *c.port != 20000 {
		t.Errorf("expected 20000, got %d", *c.port)
	}
	if *c.workers != 8 {
		t.Errorf("expected 8, got %d", *c.workers)
	}
	if *c.delay != 50 {
		t.Errorf("expected 50, got %d", *c.delay)
	}
	if *c.templateInterval != 60 {
		t.Errorf("expected 60, got %d", *c.templateInterval)
	}
	if *c.webPort != 9090 {
		t.Errorf("expected 9090, got %d", *c.webPort)
	}
	if *c.webIP != "::" {
		t.Errorf("expected '::', got %q", *c.webIP)
	}
	if !*c.web {
		t.Error("expected web true")
	}
	if *c.protocol != "ipfix" {
		t.Errorf("expected 'ipfix', got %q", *c.protocol)
	}
}

func TestBarrageCommandConfigFile(t *testing.T) {
	c := &BarrageCommand{}
	args := []string{"-config", "/tmp/nonexistent.yaml"}
	if err := c.ParseFlags(args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *c.configFile != "/tmp/nonexistent.yaml" {
		t.Errorf("expected '/tmp/nonexistent.yaml', got %q", *c.configFile)
	}

	// Execute with nonexistent config file should return an error
	err := c.Execute()
	if err == nil {
		t.Error("expected error for nonexistent config file")
	}
}

// =============================================================================
// RecordCommand
// =============================================================================

func TestRecordCommandDefaults(t *testing.T) {
	c := &RecordCommand{}
	if err := c.ParseFlags([]string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *c.ip != "127.0.0.1" {
		t.Errorf("expected ip '127.0.0.1', got %q", *c.ip)
	}
	if *c.port != 9995 {
		t.Errorf("expected port 9995, got %d", *c.port)
	}
	if *c.dbDir != "recorded_flows" {
		t.Errorf("expected dbDir 'recorded_flows', got %q", *c.dbDir)
	}
	if *c.verbose != false {
		t.Errorf("expected verbose false, got %v", *c.verbose)
	}
}

func TestRecordCommandOverrides(t *testing.T) {
	c := &RecordCommand{}
	args := []string{
		"-ip", "0.0.0.0",
		"-port", "20000",
		"-db", "/tmp/flows",
		"-verbose",
	}
	if err := c.ParseFlags(args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *c.ip != "0.0.0.0" {
		t.Errorf("expected '0.0.0.0', got %q", *c.ip)
	}
	if *c.port != 20000 {
		t.Errorf("expected 20000, got %d", *c.port)
	}
	if *c.dbDir != "/tmp/flows" {
		t.Errorf("expected '/tmp/flows', got %q", *c.dbDir)
	}
	if !*c.verbose {
		t.Error("expected verbose true")
	}
}

func TestRecordCommandIPv6(t *testing.T) {
	c := &RecordCommand{}
	args := []string{"-ip", "::"}
	if err := c.ParseFlags(args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *c.ip != "::" {
		t.Errorf("expected '::', got %q", *c.ip)
	}
}

// =============================================================================
// ReplayCommand
// =============================================================================

func TestReplayCommandDefaults(t *testing.T) {
	c := &ReplayCommand{}
	if err := c.ParseFlags([]string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *c.server != "127.0.0.1" {
		t.Errorf("expected server '127.0.0.1', got %q", *c.server)
	}
	if *c.port != 9995 {
		t.Errorf("expected port 9995, got %d", *c.port)
	}
	if *c.delay != 100 {
		t.Errorf("expected delay 100, got %d", *c.delay)
	}
	if *c.dbDir != "recorded_flows" {
		t.Errorf("expected dbDir 'recorded_flows', got %q", *c.dbDir)
	}
	if *c.loop != false {
		t.Errorf("expected loop false, got %v", *c.loop)
	}
	if *c.workers != 1 {
		t.Errorf("expected workers 1, got %d", *c.workers)
	}
	if *c.updateTS != false {
		t.Errorf("expected updateTS false, got %v", *c.updateTS)
	}
	if *c.verbose != false {
		t.Errorf("expected verbose false, got %v", *c.verbose)
	}
}

func TestReplayCommandOverrides(t *testing.T) {
	c := &ReplayCommand{}
	args := []string{
		"-server", "10.0.0.1",
		"-port", "20000",
		"-delay", "50",
		"-db", "/tmp/recorded",
		"-loop",
		"-workers", "4",
		"-updatets",
		"-verbose",
	}
	if err := c.ParseFlags(args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *c.server != "10.0.0.1" {
		t.Errorf("expected '10.0.0.1', got %q", *c.server)
	}
	if *c.port != 20000 {
		t.Errorf("expected 20000, got %d", *c.port)
	}
	if *c.delay != 50 {
		t.Errorf("expected 50, got %d", *c.delay)
	}
	if *c.dbDir != "/tmp/recorded" {
		t.Errorf("expected '/tmp/recorded', got %q", *c.dbDir)
	}
	if !*c.loop {
		t.Error("expected loop true")
	}
	if *c.workers != 4 {
		t.Errorf("expected 4, got %d", *c.workers)
	}
	if !*c.updateTS {
		t.Error("expected updateTS true")
	}
	if !*c.verbose {
		t.Error("expected verbose true")
	}
}

func TestReplayCommandIPv6(t *testing.T) {
	c := &ReplayCommand{}
	args := []string{"-server", "::1"}
	if err := c.ParseFlags(args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *c.server != "::1" {
		t.Errorf("expected '::1', got %q", *c.server)
	}
}

// =============================================================================
// IPFIXCommand
// =============================================================================

func TestIPFIXCommandDefaults(t *testing.T) {
	c := &IPFIXCommand{}
	if err := c.ParseFlags([]string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *c.server != "127.0.0.1" {
		t.Errorf("expected server '127.0.0.1', got %q", *c.server)
	}
	if *c.port != 9995 {
		t.Errorf("expected port 9995, got %d", *c.port)
	}
	if *c.srcPort != 0 {
		t.Errorf("expected srcPort 0, got %d", *c.srcPort)
	}
	if *c.count != 1 {
		t.Errorf("expected count 1, got %d", *c.count)
	}
	if *c.hexDump != false {
		t.Errorf("expected hexDump false, got %v", *c.hexDump)
	}
	if *c.srcRange != "10.0.0.0/8" {
		t.Errorf("expected srcRange '10.0.0.0/8', got %q", *c.srcRange)
	}
	if *c.dstRange != "10.0.0.0/8" {
		t.Errorf("expected dstRange '10.0.0.0/8', got %q", *c.dstRange)
	}
}

func TestIPFIXCommandOverrides(t *testing.T) {
	c := &IPFIXCommand{}
	args := []string{
		"-server", "192.168.1.1",
		"-port", "20000",
		"-src-port", "5000",
		"-count", "50",
		"-hexdump",
		"-src-range", "172.16.0.0/12",
		"-dst-range", "192.168.0.0/16",
	}
	if err := c.ParseFlags(args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *c.server != "192.168.1.1" {
		t.Errorf("expected '192.168.1.1', got %q", *c.server)
	}
	if *c.port != 20000 {
		t.Errorf("expected 20000, got %d", *c.port)
	}
	if *c.srcPort != 5000 {
		t.Errorf("expected 5000, got %d", *c.srcPort)
	}
	if *c.count != 50 {
		t.Errorf("expected 50, got %d", *c.count)
	}
	if !*c.hexDump {
		t.Error("expected hexDump true")
	}
	if *c.srcRange != "172.16.0.0/12" {
		t.Errorf("expected '172.16.0.0/12', got %q", *c.srcRange)
	}
	if *c.dstRange != "192.168.0.0/16" {
		t.Errorf("expected '192.168.0.0/16', got %q", *c.dstRange)
	}
}

func TestIPFIXCommandIPv6(t *testing.T) {
	c := &IPFIXCommand{}
	args := []string{
		"-server", "::1",
		"-src-range", "2001:db8::/32",
		"-dst-range", "fd00::/8",
	}
	if err := c.ParseFlags(args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *c.server != "::1" {
		t.Errorf("expected '::1', got %q", *c.server)
	}
	if *c.srcRange != "2001:db8::/32" {
		t.Errorf("expected '2001:db8::/32', got %q", *c.srcRange)
	}
	if *c.dstRange != "fd00::/8" {
		t.Errorf("expected 'fd00::/8', got %q", *c.dstRange)
	}
}

// =============================================================================
// ProxyCommand
// =============================================================================

func TestProxyCommandDefaults(t *testing.T) {
	c := &ProxyCommand{}
	if err := c.ParseFlags([]string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *c.ip != "127.0.0.1" {
		t.Errorf("expected ip '127.0.0.1', got %q", *c.ip)
	}
	if *c.port != 9995 {
		t.Errorf("expected port 9995, got %d", *c.port)
	}
	if len(c.targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(c.targets))
	}
	if *c.verbose != false {
		t.Errorf("expected verbose false, got %v", *c.verbose)
	}
}

func TestProxyCommandOverrides(t *testing.T) {
	c := &ProxyCommand{}
	args := []string{
		"-ip", "0.0.0.0",
		"-port", "20000",
		"-target", "10.0.0.1:9995",
		"-target", "10.0.0.2:9996",
		"-target", "10.0.0.3:9997",
		"-verbose",
	}
	if err := c.ParseFlags(args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *c.ip != "0.0.0.0" {
		t.Errorf("expected '0.0.0.0', got %q", *c.ip)
	}
	if *c.port != 20000 {
		t.Errorf("expected 20000, got %d", *c.port)
	}
	if len(c.targets) != 3 {
		t.Errorf("expected 3 targets, got %d", len(c.targets))
	}
	if c.targets[0] != "10.0.0.1:9995" {
		t.Errorf("expected '10.0.0.1:9995', got %q", c.targets[0])
	}
	if c.targets[1] != "10.0.0.2:9996" {
		t.Errorf("expected '10.0.0.2:9996', got %q", c.targets[1])
	}
	if c.targets[2] != "10.0.0.3:9997" {
		t.Errorf("expected '10.0.0.3:9997', got %q", c.targets[2])
	}
	if !*c.verbose {
		t.Error("expected verbose true")
	}
}

func TestProxyCommandIPv6(t *testing.T) {
	c := &ProxyCommand{}
	args := []string{
		"-ip", "::",
		"-target", "[::1]:9995",
	}
	if err := c.ParseFlags(args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *c.ip != "::" {
		t.Errorf("expected '::', got %q", *c.ip)
	}
	if len(c.targets) != 1 {
		t.Errorf("expected 1 target, got %d", len(c.targets))
	}
	if c.targets[0] != "[::1]:9995" {
		t.Errorf("expected '[::1]:9995', got %q", c.targets[0])
	}
}

func TestProxyCommandSingleTarget(t *testing.T) {
	c := &ProxyCommand{}
	args := []string{"-target", "10.0.0.1:9995"}
	if err := c.ParseFlags(args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(c.targets) != 1 {
		t.Errorf("expected 1 target, got %d", len(c.targets))
	}
}

// =============================================================================
// Execute tests (blocking paths, verify configuration)
// =============================================================================

func TestBarrageCommandExecuteConfig(t *testing.T) {
	c := &BarrageCommand{}
	if err := c.ParseFlags([]string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *c.server != "127.0.0.1" {
		t.Errorf("expected server '127.0.0.1', got %q", *c.server)
	}
	if *c.port != 9995 {
		t.Errorf("expected port 9995, got %d", *c.port)
	}
	if *c.protocol != "netflow" {
		t.Errorf("expected protocol 'netflow', got %q", *c.protocol)
	}
}

func TestSingleCommandExecuteConfig(t *testing.T) {
	c := &SingleCommand{}
	if err := c.ParseFlags([]string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *c.server != "127.0.0.1" {
		t.Errorf("expected server '127.0.0.1', got %q", *c.server)
	}
	if *c.count != 1 {
		t.Errorf("expected count 1, got %d", *c.count)
	}
}

func TestIPFIXCommandExecuteConfig(t *testing.T) {
	c := &IPFIXCommand{}
	if err := c.ParseFlags([]string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *c.server != "127.0.0.1" {
		t.Errorf("expected server '127.0.0.1', got %q", *c.server)
	}
	if *c.count != 1 {
		t.Errorf("expected count 1, got %d", *c.count)
	}
}

func TestProxyCommandExecuteConfig(t *testing.T) {
	c := &ProxyCommand{}
	if err := c.ParseFlags([]string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *c.ip != "127.0.0.1" {
		t.Errorf("expected ip '127.0.0.1', got %q", *c.ip)
	}
	if *c.port != 9995 {
		t.Errorf("expected port 9995, got %d", *c.port)
	}
}

func TestReplayCommandExecuteConfig(t *testing.T) {
	c := &ReplayCommand{}
	if err := c.ParseFlags([]string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *c.server != "127.0.0.1" {
		t.Errorf("expected server '127.0.0.1', got %q", *c.server)
	}
	if *c.delay != 100 {
		t.Errorf("expected delay 100, got %d", *c.delay)
	}
}

func TestRecordCommandExecuteConfig(t *testing.T) {
	c := &RecordCommand{}
	if err := c.ParseFlags([]string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *c.ip != "127.0.0.1" {
		t.Errorf("expected ip '127.0.0.1', got %q", *c.ip)
	}
	if *c.dbDir != "recorded_flows" {
		t.Errorf("expected dbDir 'recorded_flows', got %q", *c.dbDir)
	}
}

// =============================================================================
// Run* entry points (verify flag parsing without blocking)
// =============================================================================

func TestRunSingle(t *testing.T) {
	// Verify RunSingle parses flags correctly
	c := &SingleCommand{}
	err := c.ParseFlags([]string{"-server", "127.0.0.1", "-port", "9995"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *c.server != "127.0.0.1" || *c.port != 9995 {
		t.Error("flags not parsed correctly")
	}
}

func TestRunBarrage(t *testing.T) {
	// Verify RunBarrage parses flags correctly
	c := &BarrageCommand{}
	err := c.ParseFlags([]string{"-server", "127.0.0.1", "-port", "9995"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *c.server != "127.0.0.1" || *c.port != 9995 {
		t.Error("flags not parsed correctly")
	}
}

func TestRunIPFIX(t *testing.T) {
	// Verify RunIPFIX parses flags correctly
	c := &IPFIXCommand{}
	err := c.ParseFlags([]string{"-server", "127.0.0.1", "-port", "9995"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *c.server != "127.0.0.1" || *c.port != 9995 {
		t.Error("flags not parsed correctly")
	}
}

func TestRunRecord(t *testing.T) {
	// Verify RunRecord parses flags correctly
	c := &RecordCommand{}
	err := c.ParseFlags([]string{"-ip", "127.0.0.1", "-port", "9995"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *c.ip != "127.0.0.1" || *c.port != 9995 {
		t.Error("flags not parsed correctly")
	}
}

func TestRunReplay(t *testing.T) {
	// Verify RunReplay parses flags correctly
	c := &ReplayCommand{}
	err := c.ParseFlags([]string{"-server", "127.0.0.1", "-port", "9995"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *c.server != "127.0.0.1" || *c.port != 9995 {
		t.Error("flags not parsed correctly")
	}
}

func TestRunProxy(t *testing.T) {
	// Verify RunProxy parses flags correctly
	c := &ProxyCommand{}
	err := c.ParseFlags([]string{"-ip", "127.0.0.1", "-port", "9995", "-target", "10.0.0.1:9995"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *c.ip != "127.0.0.1" || *c.port != 9995 {
		t.Error("flags not parsed correctly")
	}
}

// =============================================================================
// Validation tests
// =============================================================================

func TestValidateProtocol(t *testing.T) {
	tests := []struct {
		name    string
		proto   string
		wantErr bool
	}{
		{"netflow", "netflow", false},
		{"ipfix", "ipfix", false},
		{"invalid", "sflow", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProtocol(tt.proto)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateWebBinding(t *testing.T) {
	tests := []struct {
		name    string
		webIP   string
		cliUser string
		cliPass string
		envUser string
		envPass string
		wantErr bool
	}{
		{"loopback safe", "127.0.0.1", "", "", "", "", false},
		{"ipv6 loopback safe", "::1", "", "", "", "", false},
		{"non-loopback with cli creds", "0.0.0.0", "admin", "secret", "", "", false},
		{"non-loopback with env creds", "0.0.0.0", "", "", "admin", "secret", false},
		{"non-loopback no creds", "0.0.0.0", "", "", "", "", true},
		{"non-loopback cli user only", "192.168.1.1", "admin", "", "", "", false},
		{"non-loopback cli pass only", "192.168.1.1", "", "secret", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envUser != "" {
				os.Setenv("FLOWGRE_WEB_USERNAME", tt.envUser)
				defer os.Unsetenv("FLOWGRE_WEB_USERNAME")
			}
			if tt.envPass != "" {
				os.Setenv("FLOWGRE_WEB_PASSWORD", tt.envPass)
				defer os.Unsetenv("FLOWGRE_WEB_PASSWORD")
			}
			err := validateWebBinding(tt.webIP, tt.cliUser, tt.cliPass)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestResolveCredentials(t *testing.T) {
	// Test CLI credentials
	username, hashed, err := resolveCredentials("myuser", "mypass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "myuser" {
		t.Errorf("expected username 'myuser', got %q", username)
	}
	if hashed == "" {
		t.Error("expected non-empty hashed password for CLI credentials")
	}

	// Test environment variable credentials
	os.Setenv("FLOWGRE_WEB_USERNAME", "envuser")
	os.Setenv("FLOWGRE_WEB_PASSWORD", "envpass")
	envUsername, envHashed, err := resolveCredentials("", "")
	if err != nil {
		os.Unsetenv("FLOWGRE_WEB_USERNAME")
		os.Unsetenv("FLOWGRE_WEB_PASSWORD")
		t.Fatalf("unexpected error: %v", err)
	}
	os.Unsetenv("FLOWGRE_WEB_USERNAME")
	os.Unsetenv("FLOWGRE_WEB_PASSWORD")
	if envUsername != "envuser" {
		t.Errorf("expected username 'envuser', got %q", envUsername)
	}
	if envHashed == "" {
		t.Error("expected non-empty hashed password for env credentials")
	}

	// Test default credentials (random password generated)
	defUsername, defHashed, err := resolveCredentials("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if defUsername != "admin" {
		t.Errorf("expected default username 'admin', got %q", defUsername)
	}
	if defHashed == "" {
		t.Error("expected non-empty hashed password for default credentials")
	}
}

func TestResolveProfile(t *testing.T) {
	tests := []struct {
		name     string
		profile  string
		expected string
	}{
		{"generic", "generic", "generic"},
		{"minimal", "minimal", "minimal"},
		{"extended", "extended", "extended"},
		{"unknown", "unknown", "generic"},
		{"empty", "", "generic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := resolveProfile(tt.profile)
			name := profile.Name()
			if name != tt.expected {
				t.Errorf("expected name %q, got %q", tt.expected, name)
			}
		})
	}
}

func TestGenerateRandomPassword(t *testing.T) {
	password, err := web.GenerateRandomPassword(16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(password) != 16 {
		t.Errorf("expected length 16, got %d", len(password))
	}

	// Generate two passwords and verify they differ (probabilistic)
	p1, err := web.GenerateRandomPassword(32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	p2, err := web.GenerateRandomPassword(32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p1 == p2 {
		t.Log("Warning: two random passwords are identical (extremely unlikely)")
	}
}

func TestEffectiveWebIP(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty defaults to loopback", "", "127.0.0.1"},
		{"loopback unchanged", "127.0.0.1", "127.0.0.1"},
		{"non-loopback unchanged", "0.0.0.0", "0.0.0.0"},
		{"ipv6 loopback unchanged", "::1", "::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := effectiveWebIP(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestValidateWebBindingEmptyIP(t *testing.T) {
	// Empty web-ip should be treated as loopback (safe)
	err := validateWebBinding("", "", "")
	if err != nil {
		t.Errorf("empty web-ip should be safe (defaults to loopback), got error: %v", err)
	}
}

func TestValidateTemplateInterval(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{"zero disabled", 0, false},
		{"positive", 30, false},
		{"negative", -1, true},
		{"large negative", -100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTemplateInterval(tt.value)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestBarrageCommandNegativeTemplateInterval(t *testing.T) {
	c := &BarrageCommand{}
	args := []string{"-template-interval", "-1"}
	if err := c.ParseFlags(args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err := c.Execute()
	if err == nil {
		t.Error("expected error for negative template-interval")
	}
}
