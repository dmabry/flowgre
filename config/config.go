// Package config handles YAML configuration file loading for flowgre modes.
package config

import (
	"fmt"
	"log"
	"strconv"

	"github.com/dmabry/flowgre/models"
	"github.com/spf13/viper"
)

// LoadBarrageConfig reads a Viper-loaded YAML config and returns a models.Config.
// The expected format is:
//
//	targets:
//	  server1:
//	    ip: 127.0.0.1
//	    port: 9995
//	    workers: 4
//	    delay: 100
//	    src-range: 10.0.0.0/8
//	    dst-range: 10.0.0.0/8
//	    web: false
//	    web-ip: 0.0.0.0
//	    web-port: 8080
func LoadBarrageConfig() (*models.Config, error) {
	if !viper.IsSet("targets") {
		return nil, fmt.Errorf("couldn't find targets section in config file")
	}

	targets := viper.Get("targets")
	targetMap, ok := targets.(map[string]interface{})
	if !ok || len(targetMap) == 0 {
		return nil, fmt.Errorf("no targets found in config")
	}

	if len(targetMap) > 1 {
		return nil, fmt.Errorf("found more than 1 target in config file, only 1 is allowed")
	}

	// Get the single target
	var targetName string
	var targetValues map[string]interface{}
	for name, vals := range targetMap {
		targetName = name
		tv, ok := vals.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("unexpected type for target %s: %T", name, vals)
		}
		targetValues = tv
	}

	// Extract values with defaults
	ip := getString(targetValues, "ip", "127.0.0.1")
	port := getInt(targetValues, "port", 9995)
	workers := getInt(targetValues, "workers", 4)
	delay := getInt(targetValues, "delay", 100)
	srcRange := getString(targetValues, "src-range", "10.0.0.0/8")
	dstRange := getString(targetValues, "dst-range", "10.0.0.0/8")
	webIP := getString(targetValues, "web-ip", "0.0.0.0")
	webPort := getInt(targetValues, "web-port", 8080)
	web := getBool(targetValues, "web", false)

	log.Printf("target: %s ip: %s port: %d workers: %d delay: %d src-range: %s dst-range: %s web: %v web-ip: %s web-port: %d\n",
		targetName, ip, port, workers, delay, srcRange, dstRange, web, webIP, webPort)

	return &models.Config{
		Server:   ip,
		DstPort:  port,
		Workers:  workers,
		Delay:    delay,
		SrcRange: srcRange,
		DstRange: dstRange,
		WebIP:    webIP,
		WebPort:  webPort,
		Web:      web,
	}, nil
}

// getString safely gets a string value from a map with a default.
func getString(m map[string]interface{}, key, def string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return def
}

// getInt safely gets an int value from a map with a default.
// Viper returns float64 for numbers, so we handle that.
func getInt(m map[string]interface{}, key string, def int) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		case string:
			// Try to parse as int
			result, err := strconv.Atoi(val)
			if err != nil {
				return def
			}
			return result
		}
	}
	return def
}

// getBool safely gets a bool value from a map with a default.
func getBool(m map[string]interface{}, key string, def bool) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
		// Handle string "true"/"false"
		if s, ok := v.(string); ok {
			return s == "true" || s == "1" || s == "yes"
		}
	}
	return def
}

// InitViper sets up Viper to read from the given config file path.
func InitViper(configPath string) error {
	viper.SetConfigFile(configPath)
	return viper.ReadInConfig()
}
