// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
)

// TestInitViper tests that InitViper successfully loads a valid config file.
func TestInitViper(t *testing.T) {
	viper.Reset()
	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write valid config
	configContent := `
targets:
  server1:
    ip: 127.0.0.1
    port: 9995
    workers: 4
    delay: 100
`
	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// Test InitViper
	err = InitViper(tmpFile.Name())
	if err != nil {
		t.Errorf("InitViper failed: %v", err)
	}
}

// TestInitViperMissingFile tests that InitViper returns error for missing file.
func TestInitViperMissingFile(t *testing.T) {
	viper.Reset()
	err := InitViper("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error for missing file, got nil")
	}
}

// TestLoadBarrageConfigValid tests loading a valid config.
func TestLoadBarrageConfigValid(t *testing.T) {
	viper.Reset()
	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `
targets:
  server1:
    ip: 192.168.1.100
    port: 2000
    workers: 8
    delay: 50
    src-range: 172.16.0.0/12
    dst-range: 192.168.0.0/16
    web: true
    web-ip: 127.0.0.1
    web-port: 3000
`
	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// Load config
	if err := InitViper(tmpFile.Name()); err != nil {
		t.Fatalf("InitViper failed: %v", err)
	}

	cfg, err := LoadBarrageConfig()
	if err != nil {
		t.Fatalf("LoadBarrageConfig failed: %v", err)
	}

	// Verify values
	if cfg.Server != "192.168.1.100" {
		t.Errorf("Expected Server '192.168.1.100', got '%s'", cfg.Server)
	}
	if cfg.DstPort != 2000 {
		t.Errorf("Expected DstPort 2000, got %d", cfg.DstPort)
	}
	if cfg.Workers != 8 {
		t.Errorf("Expected Workers 8, got %d", cfg.Workers)
	}
	if cfg.Delay != 50 {
		t.Errorf("Expected Delay 50, got %d", cfg.Delay)
	}
	if cfg.SrcRange != "172.16.0.0/12" {
		t.Errorf("Expected SrcRange '172.16.0.0/12', got '%s'", cfg.SrcRange)
	}
	if cfg.DstRange != "192.168.0.0/16" {
		t.Errorf("Expected DstRange '192.168.0.0/16', got '%s'", cfg.DstRange)
	}
	if cfg.Web != true {
		t.Errorf("Expected Web true, got %v", cfg.Web)
	}
	if cfg.WebIP != "127.0.0.1" {
		t.Errorf("Expected WebIP '127.0.0.1', got '%s'", cfg.WebIP)
	}
	if cfg.WebPort != 3000 {
		t.Errorf("Expected WebPort 3000, got %d", cfg.WebPort)
	}
}

// TestLoadBarrageConfigMissingTargets tests that missing targets section returns error.
func TestLoadBarrageConfigMissingTargets(t *testing.T) {
	viper.Reset()
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `
other:
  key: value
`
	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	if err := InitViper(tmpFile.Name()); err != nil {
		t.Fatalf("InitViper failed: %v", err)
	}

	_, err = LoadBarrageConfig()
	if err == nil {
		t.Error("Expected error for missing targets, got nil")
	}
}

// TestLoadBarrageConfigMultipleTargets tests that multiple targets return error.
func TestLoadBarrageConfigMultipleTargets(t *testing.T) {
	viper.Reset()
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `
targets:
  server1:
    ip: 127.0.0.1
    port: 9995
    workers: 4
    delay: 100
  server2:
    ip: 127.0.0.2
    port: 9995
    workers: 4
    delay: 100
`
	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	if err := InitViper(tmpFile.Name()); err != nil {
		t.Fatalf("InitViper failed: %v", err)
	}

	_, err = LoadBarrageConfig()
	if err == nil {
		t.Error("Expected error for multiple targets, got nil")
	}
}

// TestLoadBarrageConfigDefaults tests that missing fields get defaults.
func TestLoadBarrageConfigDefaults(t *testing.T) {
	viper.Reset()
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `
targets:
  server1:
    ip: 10.0.0.1
`
	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	if err := InitViper(tmpFile.Name()); err != nil {
		t.Fatalf("InitViper failed: %v", err)
	}

	cfg, err := LoadBarrageConfig()
	if err != nil {
		t.Fatalf("LoadBarrageConfig failed: %v", err)
	}

	// Verify defaults
	if cfg.Server != "10.0.0.1" {
		t.Errorf("Expected Server '10.0.0.1', got '%s'", cfg.Server)
	}
	if cfg.DstPort != 9995 {
		t.Errorf("Expected DstPort 9995 (default), got %d", cfg.DstPort)
	}
	if cfg.Workers != 4 {
		t.Errorf("Expected Workers 4 (default), got %d", cfg.Workers)
	}
	if cfg.Delay != 100 {
		t.Errorf("Expected Delay 100 (default), got %d", cfg.Delay)
	}
	if cfg.SrcRange != "10.0.0.0/8" {
		t.Errorf("Expected SrcRange '10.0.0.0/8' (default), got '%s'", cfg.SrcRange)
	}
	if cfg.Web != false {
		t.Errorf("Expected Web false (default), got %v", cfg.Web)
	}
}

// TestLoadBarrageConfigValidation tests config validation with various inputs.
func TestLoadBarrageConfigValidation(t *testing.T) {
	viper.Reset()
	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{
			name: "valid config",
			config: `
targets:
  server1:
    ip: 127.0.0.1
    port: 9995
    workers: 4
    delay: 100
`,
			wantErr: false,
		},
		{
			name: "empty targets",
			config: `
targets: {}
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			tmpFile, err := os.CreateTemp("", "config-*.yaml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.config); err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}
			tmpFile.Close()

			if err := InitViper(tmpFile.Name()); err != nil {
				t.Fatalf("InitViper failed: %v", err)
			}

			_, err = LoadBarrageConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadBarrageConfig error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
