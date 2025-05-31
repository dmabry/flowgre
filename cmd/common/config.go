// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Package common provides shared functionality for all commands
package common

import (
	"log"

	"github.com/spf13/viper"
)

// InitConfig initializes the configuration management using viper
func InitConfig(configFile string) {
	if configFile != "" {
		log.Printf("Reading config file... ignoring any other given arguments\n\n")
		viper.SetConfigFile(configFile)

		if err := viper.ReadInConfig(); err != nil {
			FatalError("error reading config file", err)
		}
	}
}

// GetString gets a string value from the configuration
func GetString(key string, defaultValue string) string {
	return viper.GetString(key, defaultValue)
}

// GetInt gets an int value from the configuration
func GetInt(key string, defaultValue int) int {
	return viper.GetInt(key, defaultValue)
}

// GetBool gets a bool value from the configuration
func GetBool(key string, defaultValue bool) bool {
	return viper.GetBool(key, defaultValue)
}