// Package cmd provides per-mode command implementations for flowgre.
package cmd

import (
	"testing"
)

// TestSingleCommandParseFlags tests flag parsing for single mode.
func TestSingleCommandParseFlags(t *testing.T) {
	t.Parallel()
	c := &SingleCommand{}

	args := []string{"-server", "10.0.0.1", "-port", "9996", "-count", "50"}
	err := c.ParseFlags(args)
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if *c.server != "10.0.0.1" {
		t.Errorf("server = %q, want %q", *c.server, "10.0.0.1")
	}
	if *c.port != 9996 {
		t.Errorf("port = %d, want %d", *c.port, 9996)
	}
	if *c.count != 50 {
		t.Errorf("count = %d, want %d", *c.count, 50)
	}
}

// TestSingleCommandParseFlagsDefaults tests default values for single mode.
func TestSingleCommandParseFlagsDefaults(t *testing.T) {
	t.Parallel()
	c := &SingleCommand{}

	err := c.ParseFlags([]string{})
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if *c.server != "127.0.0.1" {
		t.Errorf("default server = %q, want %q", *c.server, "127.0.0.1")
	}
	if *c.port != 9995 {
		t.Errorf("default port = %d, want %d", *c.port, 9995)
	}
	if *c.srcPort != 0 {
		t.Errorf("default srcPort = %d, want %d", *c.srcPort, 0)
	}
	if *c.count != 1 {
		t.Errorf("default count = %d, want %d", *c.count, 1)
	}
	if *c.hexDump != false {
		t.Errorf("default hexDump = %v, want false", *c.hexDump)
	}
	if *c.srcRange != "10.0.0.0/8" {
		t.Errorf("default srcRange = %q, want %q", *c.srcRange, "10.0.0.0/8")
	}
	if *c.dstRange != "10.0.0.0/8" {
		t.Errorf("default dstRange = %q, want %q", *c.dstRange, "10.0.0.0/8")
	}
}

// TestSingleCommandParseFlagsHexDump tests hexdump flag.
func TestSingleCommandParseFlagsHexDump(t *testing.T) {
	t.Parallel()
	c := &SingleCommand{}

	args := []string{"-hexdump"}
	err := c.ParseFlags(args)
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if *c.hexDump != true {
		t.Errorf("hexDump = %v, want true", *c.hexDump)
	}
}

// TestProxyCommandParseFlags tests flag parsing for proxy mode.
func TestProxyCommandParseFlags(t *testing.T) {
	t.Parallel()
	c := &ProxyCommand{}

	args := []string{"-ip", "0.0.0.0", "-port", "19995", "-target", "10.0.0.1:9995", "-target", "10.0.0.2:9996"}
	err := c.ParseFlags(args)
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if *c.ip != "0.0.0.0" {
		t.Errorf("ip = %q, want %q", *c.ip, "0.0.0.0")
	}
	if *c.port != 19995 {
		t.Errorf("port = %d, want %d", *c.port, 19995)
	}
	if len(c.targets) != 2 {
		t.Errorf("targets length = %d, want %d", len(c.targets), 2)
	}
	if c.targets[0] != "10.0.0.1:9995" {
		t.Errorf("targets[0] = %q, want %q", c.targets[0], "10.0.0.1:9995")
	}
	if c.targets[1] != "10.0.0.2:9996" {
		t.Errorf("targets[1] = %q, want %q", c.targets[1], "10.0.0.2:9996")
	}
}

// TestProxyCommandParseFlagsDefaults tests default values for proxy mode.
func TestProxyCommandParseFlagsDefaults(t *testing.T) {
	t.Parallel()
	c := &ProxyCommand{}

	err := c.ParseFlags([]string{})
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if *c.ip != "127.0.0.1" {
		t.Errorf("default ip = %q, want %q", *c.ip, "127.0.0.1")
	}
	if *c.port != 9995 {
		t.Errorf("default port = %d, want %d", *c.port, 9995)
	}
	if len(c.targets) != 0 {
		t.Errorf("default targets length = %d, want 0", len(c.targets))
	}
	if *c.verbose != false {
		t.Errorf("default verbose = %v, want false", *c.verbose)
	}
}

// TestProxyCommandParseFlagsVerbose tests verbose flag.
func TestProxyCommandParseFlagsVerbose(t *testing.T) {
	t.Parallel()
	c := &ProxyCommand{}

	args := []string{"-verbose"}
	err := c.ParseFlags(args)
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if *c.verbose != true {
		t.Errorf("verbose = %v, want true", *c.verbose)
	}
}

// TestTargetFlagsSet tests the targetFlags Set method.
func TestTargetFlagsSet(t *testing.T) {
	t.Parallel()
	var tf targetFlags

	err := tf.Set("10.0.0.1:9995")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if len(tf) != 1 || tf[0] != "10.0.0.1:9995" {
		t.Errorf("tf = %v, want [10.0.0.1:9995]", tf)
	}

	err = tf.Set("10.0.0.2:9996")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if len(tf) != 2 || tf[1] != "10.0.0.2:9996" {
		t.Errorf("tf = %v, want [10.0.0.1:9995 10.0.0.2:9996]", tf)
	}
}

// TestTargetFlagsString tests the targetFlags String method.
func TestTargetFlagsString(t *testing.T) {
	t.Parallel()
	var tf targetFlags

	result := tf.String()
	if result != "<multiple>" {
		t.Errorf("String() = %q, want %q", result, "<multiple>")
	}
}

// TestRecordCommandParseFlags tests flag parsing for record mode.
func TestRecordCommandParseFlags(t *testing.T) {
	t.Parallel()
	c := &RecordCommand{}

	args := []string{"-ip", "0.0.0.0", "-port", "29995", "-db", "/tmp/test_flows"}
	err := c.ParseFlags(args)
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if *c.ip != "0.0.0.0" {
		t.Errorf("ip = %q, want %q", *c.ip, "0.0.0.0")
	}
	if *c.port != 29995 {
		t.Errorf("port = %d, want %d", *c.port, 29995)
	}
	if *c.dbDir != "/tmp/test_flows" {
		t.Errorf("dbDir = %q, want %q", *c.dbDir, "/tmp/test_flows")
	}
}

// TestRecordCommandParseFlagsDefaults tests default values for record mode.
func TestRecordCommandParseFlagsDefaults(t *testing.T) {
	t.Parallel()
	c := &RecordCommand{}

	err := c.ParseFlags([]string{})
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if *c.ip != "127.0.0.1" {
		t.Errorf("default ip = %q, want %q", *c.ip, "127.0.0.1")
	}
	if *c.port != 9995 {
		t.Errorf("default port = %d, want %d", *c.port, 9995)
	}
	if *c.dbDir != "recorded_flows" {
		t.Errorf("default dbDir = %q, want %q", *c.dbDir, "recorded_flows")
	}
	if *c.verbose != false {
		t.Errorf("default verbose = %v, want false", *c.verbose)
	}
}

// TestReplayCommandParseFlags tests flag parsing for replay mode.
func TestReplayCommandParseFlags(t *testing.T) {
	t.Parallel()
	c := &ReplayCommand{}

	args := []string{"-server", "10.0.0.1", "-port", "39995", "-delay", "200", "-loop", "-workers", "4"}
	err := c.ParseFlags(args)
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if *c.server != "10.0.0.1" {
		t.Errorf("server = %q, want %q", *c.server, "10.0.0.1")
	}
	if *c.port != 39995 {
		t.Errorf("port = %d, want %d", *c.port, 39995)
	}
	if *c.delay != 200 {
		t.Errorf("delay = %d, want %d", *c.delay, 200)
	}
	if *c.loop != true {
		t.Errorf("loop = %v, want true", *c.loop)
	}
	if *c.workers != 4 {
		t.Errorf("workers = %d, want %d", *c.workers, 4)
	}
}

// TestReplayCommandParseFlagsDefaults tests default values for replay mode.
func TestReplayCommandParseFlagsDefaults(t *testing.T) {
	t.Parallel()
	c := &ReplayCommand{}

	err := c.ParseFlags([]string{})
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if *c.server != "127.0.0.1" {
		t.Errorf("default server = %q, want %q", *c.server, "127.0.0.1")
	}
	if *c.port != 9995 {
		t.Errorf("default port = %d, want %d", *c.port, 9995)
	}
	if *c.delay != 100 {
		t.Errorf("default delay = %d, want %d", *c.delay, 100)
	}
	if *c.dbDir != "recorded_flows" {
		t.Errorf("default dbDir = %q, want %q", *c.dbDir, "recorded_flows")
	}
	if *c.loop != false {
		t.Errorf("default loop = %v, want false", *c.loop)
	}
	if *c.workers != 1 {
		t.Errorf("default workers = %d, want %d", *c.workers, 1)
	}
	if *c.updateTS != false {
		t.Errorf("default updateTS = %v, want false", *c.updateTS)
	}
	if *c.verbose != false {
		t.Errorf("default verbose = %v, want false", *c.verbose)
	}
}

// TestReplayCommandParseFlagsUpdateTS tests updatets flag.
func TestReplayCommandParseFlagsUpdateTS(t *testing.T) {
	t.Parallel()
	c := &ReplayCommand{}

	args := []string{"-updatets"}
	err := c.ParseFlags(args)
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if *c.updateTS != true {
		t.Errorf("updateTS = %v, want true", *c.updateTS)
	}
}

// TestBarrageCommandParseFlags tests flag parsing for barrage mode.
func TestBarrageCommandParseFlags(t *testing.T) {
	t.Parallel()
	c := &BarrageCommand{}

	args := []string{"-server", "10.0.0.1", "-port", "49995", "-workers", "8", "-delay", "50"}
	err := c.ParseFlags(args)
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if *c.server != "10.0.0.1" {
		t.Errorf("server = %q, want %q", *c.server, "10.0.0.1")
	}
	if *c.port != 49995 {
		t.Errorf("port = %d, want %d", *c.port, 49995)
	}
	if *c.workers != 8 {
		t.Errorf("workers = %d, want %d", *c.workers, 8)
	}
	if *c.delay != 50 {
		t.Errorf("delay = %d, want %d", *c.delay, 50)
	}
}

// TestBarrageCommandParseFlagsDefaults tests default values for barrage mode.
func TestBarrageCommandParseFlagsDefaults(t *testing.T) {
	t.Parallel()
	c := &BarrageCommand{}

	err := c.ParseFlags([]string{})
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if *c.server != "127.0.0.1" {
		t.Errorf("default server = %q, want %q", *c.server, "127.0.0.1")
	}
	if *c.port != 9995 {
		t.Errorf("default port = %d, want %d", *c.port, 9995)
	}
	if *c.srcRange != "10.0.0.0/8" {
		t.Errorf("default srcRange = %q, want %q", *c.srcRange, "10.0.0.0/8")
	}
	if *c.dstRange != "10.0.0.0/8" {
		t.Errorf("default dstRange = %q, want %q", *c.dstRange, "10.0.0.0/8")
	}
	if *c.workers != 4 {
		t.Errorf("default workers = %d, want %d", *c.workers, 4)
	}
	if *c.delay != 100 {
		t.Errorf("default delay = %d, want %d", *c.delay, 100)
	}
	if *c.webPort != 8080 {
		t.Errorf("default webPort = %d, want %d", *c.webPort, 8080)
	}
	if *c.webIP != "0.0.0.0" {
		t.Errorf("default webIP = %q, want %q", *c.webIP, "0.0.0.0")
	}
	if *c.web != false {
		t.Errorf("default web = %v, want false", *c.web)
	}
}

// TestBarrageCommandParseFlagsWeb tests web flags.
func TestBarrageCommandParseFlagsWeb(t *testing.T) {
	t.Parallel()
	c := &BarrageCommand{}

	args := []string{"-web", "-web-port", "9090", "-web-ip", "127.0.0.1"}
	err := c.ParseFlags(args)
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if *c.web != true {
		t.Errorf("web = %v, want true", *c.web)
	}
	if *c.webPort != 9090 {
		t.Errorf("webPort = %d, want %d", *c.webPort, 9090)
	}
	if *c.webIP != "127.0.0.1" {
		t.Errorf("webIP = %q, want %q", *c.webIP, "127.0.0.1")
	}
}

// TestBarrageCommandParseFlagsConfig tests config flag.
func TestBarrageCommandParseFlagsConfig(t *testing.T) {
	t.Parallel()
	c := &BarrageCommand{}

	args := []string{"-config", "/tmp/test_config.yaml"}
	err := c.ParseFlags(args)
	if err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if *c.configFile != "/tmp/test_config.yaml" {
		t.Errorf("configFile = %q, want %q", *c.configFile, "/tmp/test_config.yaml")
	}
}
