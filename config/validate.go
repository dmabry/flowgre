// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package config provides centralized validation for command-line and YAML
// configuration before any goroutine, socket, or database is started.
package config

import (
	"fmt"
	"net"
	"strconv"
)

// ValidateRecord validates record command configuration.
func ValidateRecord(ip string, port int, dbdir string) error {
	if err := validateListenerIP(ip); err != nil {
		return fmt.Errorf("record listener IP: %w", err)
	}
	if err := validatePort(port, false); err != nil {
		return fmt.Errorf("record listener port: %w", err)
	}
	if dbdir == "" {
		return fmt.Errorf("record database directory is required")
	}
	return nil
}

// ValidateReplay validates replay command configuration.
func ValidateReplay(server string, port int, delay int, dbdir string, workers int) error {
	if err := validateDestIP(server); err != nil {
		return fmt.Errorf("replay destination IP: %w", err)
	}
	if err := validatePort(port, false); err != nil {
		return fmt.Errorf("replay destination port: %w", err)
	}
	if delay <= 0 {
		return fmt.Errorf("replay delay must be positive, got %d", delay)
	}
	if dbdir == "" {
		return fmt.Errorf("replay database directory is required")
	}
	if workers < 1 {
		return fmt.Errorf("replay workers must be at least 1, got %d", workers)
	}
	return nil
}

// ValidateBarrage validates barrage command configuration.
func ValidateBarrage(server string, port int, srcRange, dstRange string, workers, delay, templateInterval int) error {
	if err := validateDestIP(server); err != nil {
		return fmt.Errorf("barrage destination IP: %w", err)
	}
	if err := validatePort(port, false); err != nil {
		return fmt.Errorf("barrage destination port: %w", err)
	}
	if err := validateCIDR(srcRange, "src-range"); err != nil {
		return err
	}
	if err := validateCIDR(dstRange, "dst-range"); err != nil {
		return err
	}
	if workers < 1 {
		return fmt.Errorf("barrage workers must be at least 1, got %d", workers)
	}
	if delay <= 0 {
		return fmt.Errorf("barrage delay must be positive, got %d", delay)
	}
	if templateInterval < 0 {
		return fmt.Errorf("barrage template-interval must be 0 (disabled) or positive, got %d", templateInterval)
	}
	return nil
}

// ValidateProxy validates proxy command configuration.
func ValidateProxy(ip string, port int, targets []string) error {
	if err := validateListenerIP(ip); err != nil {
		return fmt.Errorf("proxy listener IP: %w", err)
	}
	if err := validatePort(port, false); err != nil {
		return fmt.Errorf("proxy listener port: %w", err)
	}
	if len(targets) == 0 {
		return fmt.Errorf("proxy requires at least one target")
	}
	if len(targets) > 10 {
		return fmt.Errorf("proxy supports at most 10 targets, got %d", len(targets))
	}
	for _, target := range targets {
		host, p, err := net.SplitHostPort(target)
		if err != nil {
			return fmt.Errorf("proxy target %q: %w", target, err)
		}
		if err := validateDestIP(host); err != nil {
			return fmt.Errorf("proxy target %q IP: %w", target, err)
		}
		portInt, err := strconv.Atoi(p)
		if err != nil {
			return fmt.Errorf("proxy target %q port is not a valid integer: %q", target, p)
		}
		if portInt < 1 || portInt > 65535 {
			return fmt.Errorf("proxy target %q port %d out of range 1-65535", target, portInt)
		}
	}
	return nil
}

// ValidateWeb validates web server configuration.
func ValidateWeb(ip string, port int) error {
	if err := validateListenerIP(ip); err != nil {
		return fmt.Errorf("web listener IP: %w", err)
	}
	if err := validatePort(port, false); err != nil {
		return fmt.Errorf("web listener port: %w", err)
	}
	return nil
}

// validateListenerIP validates an IP address used for listening.
func validateListenerIP(ip string) error {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return fmt.Errorf("invalid IP address: %q", ip)
	}
	return nil
}

// validateDestIP validates an IP address used as a destination.
func validateDestIP(ip string) error {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return fmt.Errorf("invalid IP address: %q", ip)
	}
	return nil
}

// validatePort validates a port number. ephemeral allows 0 for source ports.
func validatePort(port int, ephemeral bool) error {
	if ephemeral {
		if port < 0 || port > 65535 {
			return fmt.Errorf("port %d out of range 0-65535", port)
		}
		return nil
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("port %d out of range 1-65535", port)
	}
	return nil
}

// validateCIDR validates a CIDR notation string.
func validateCIDR(cidr, name string) error {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid %s CIDR %q: %w", name, cidr, err)
	}
	return nil
}
