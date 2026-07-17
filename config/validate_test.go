// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

package config

import "testing"

func TestValidateRecord(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		port    int
		dbdir   string
		wantErr bool
	}{
		{"valid", "127.0.0.1", 9995, "/tmp/db", false},
		{"valid IPv6", "::1", 9995, "/tmp/db", false},
		{"invalid IP", "localhos", 9995, "/tmp/db", true},
		{"port zero", "127.0.0.1", 0, "/tmp/db", true},
		{"port overflow", "127.0.0.1", 65536, "/tmp/db", true},
		{"empty dbdir", "127.0.0.1", 9995, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecord(tt.ip, tt.port, tt.dbdir)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRecord() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateReplay(t *testing.T) {
	tests := []struct {
		name    string
		server  string
		port    int
		delay   int
		dbdir   string
		workers int
		wantErr bool
	}{
		{"valid", "127.0.0.1", 9995, 100, "/tmp/db", 1, false},
		{"valid IPv6", "::1", 9995, 100, "/tmp/db", 4, false},
		{"invalid server IP", "bad", 9995, 100, "/tmp/db", 1, true},
		{"port zero", "127.0.0.1", 0, 100, "/tmp/db", 1, true},
		{"delay zero", "127.0.0.1", 9995, 0, "/tmp/db", 1, true},
		{"delay negative", "127.0.0.1", 9995, -1, "/tmp/db", 1, true},
		{"workers zero", "127.0.0.1", 9995, 100, "/tmp/db", 0, true},
		{"workers negative", "127.0.0.1", 9995, 100, "/tmp/db", -1, true},
		{"empty dbdir", "127.0.0.1", 9995, 100, "", 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReplay(tt.server, tt.port, tt.delay, tt.dbdir, tt.workers)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateReplay() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateBarrage(t *testing.T) {
	tests := []struct {
		name             string
		server           string
		port             int
		srcRange         string
		dstRange         string
		workers          int
		delay            int
		templateInterval int
		wantErr          bool
	}{
		{"valid", "127.0.0.1", 9995, "10.0.0.0/8", "10.0.0.0/8", 4, 100, 30, false},
		{"valid IPv6 server", "::1", 9995, "::/64", "::/64", 1, 50, 0, false},
		{"invalid server", "bad", 9995, "10.0.0.0/8", "10.0.0.0/8", 4, 100, 30, true},
		{"invalid srcRange", "127.0.0.1", 9995, "bad", "10.0.0.0/8", 4, 100, 30, true},
		{"invalid dstRange", "127.0.0.1", 9995, "10.0.0.0/8", "bad", 4, 100, 30, true},
		{"workers zero", "127.0.0.1", 9995, "10.0.0.0/8", "10.0.0.0/8", 0, 100, 30, true},
		{"workers negative", "127.0.0.1", 9995, "10.0.0.0/8", "10.0.0.0/8", -1, 100, 30, true},
		{"delay zero", "127.0.0.1", 9995, "10.0.0.0/8", "10.0.0.0/8", 4, 0, 30, true},
		{"delay negative", "127.0.0.1", 9995, "10.0.0.0/8", "10.0.0.0/8", 4, -1, 30, true},
		{"templateInterval negative", "127.0.0.1", 9995, "10.0.0.0/8", "10.0.0.0/8", 4, 100, -1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBarrage(tt.server, tt.port, tt.srcRange, tt.dstRange, tt.workers, tt.delay, tt.templateInterval)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBarrage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateProxy(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		port    int
		targets []string
		wantErr bool
	}{
		{"valid", "127.0.0.1", 9995, []string{"127.0.0.1:9996"}, false},
		{"valid multiple", "127.0.0.1", 9995, []string{"127.0.0.1:9996", "127.0.0.1:9997"}, false},
		{"invalid listener IP", "bad", 9995, []string{"127.0.0.1:9996"}, true},
		{"port zero", "127.0.0.1", 0, []string{"127.0.0.1:9996"}, true},
		{"no targets", "127.0.0.1", 9995, nil, true},
		{"bad target format", "127.0.0.1", 9995, []string{"bad"}, true},
		{"bad target port", "127.0.0.1", 9995, []string{"127.0.0.1:0"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProxy(tt.ip, tt.port, tt.targets)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProxy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateWeb(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		port    int
		wantErr bool
	}{
		{"valid loopback", "127.0.0.1", 8080, false},
		{"valid wildcard", "0.0.0.0", 8080, false},
		{"valid IPv6", "::1", 8080, false},
		{"invalid IP", "bad", 8080, true},
		{"port zero", "127.0.0.1", 0, true},
		{"port overflow", "127.0.0.1", 65536, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWeb(tt.ip, tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWeb() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
