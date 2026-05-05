// Package config handles YAML configuration file loading for flowgre modes.
package config

import (
	"fmt"
	"log"

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
func LoadBarrageConfig() (*models.Config, error) {
	if !viper.InConfig("targets") {
		return nil, fmt.Errorf("couldn't find targets section in config file")
	}

	targets := viper.AllSettings()
	if len(targets) > 1 {
		return nil, fmt.Errorf("found more than 1 target in config file, only 1 is allowed")
	}

	for _, value := range targets {
		v, ok := value.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("unexpected type returned by viper: %T", value)
		}
		for targetName, targetValues := range v {
			t, ok := targetValues.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("unexpected nested type for target %s", targetName)
			}

		ip, _ := t["ip"].(string)
		port, _ := t["port"].(float64)
		workers, _ := t["workers"].(float64)
		delay, _ := t["delay"].(float64)

		log.Printf("target: %s ip: %s port: %d workers: %d delay: %d\n",
			targetName, ip, int(port), int(workers), int(delay))

		return &models.Config{
			Server:  ip,
			DstPort: int(port),
			Workers: int(workers),
			Delay:   int(delay),
		}, nil
		}
	}

	return nil, fmt.Errorf("no targets found in config")
}

// InitViper sets up Viper to read from the given config file path.
func InitViper(configPath string) error {
	viper.SetConfigFile(configPath)
	return viper.ReadInConfig()
}
