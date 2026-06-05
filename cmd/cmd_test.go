// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"testing"
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
