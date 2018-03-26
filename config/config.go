package config

import (
	"github.com/spf13/viper"
	"fmt"
)


const (
	// Configuration file default path
	DefaultConfigPath     = "$HOME/.gobalance/"
	DefaultConfigFilename = "conf"
)


// ConfigError()
type ConfigError struct {
	s string
}

func (e *ConfigError) Error() string {
	return e.s
}

func NewConfigError(text string) error {
	return &ConfigError{s: text}
}

// Parse configuration options and check types.
func ParseConfig(vip *viper.Viper) (map[string]interface{}, error){

	conf := make(map[string]interface{})
	
	// Load options into a map of supportedOptions
	supportedOptions := make(map[string]Option)
	for _, op := range Options {
		supportedOptions[op.name] = op
	}

	// Check all options in the file are supported
	allOptions := vip.AllKeys()

	for _, opName := range allOptions {
		if _, ok := supportedOptions[opName]; !ok {
			return nil, NewConfigError(fmt.Sprintf("%v - Unsupported option", opName))
		}
	}

	// Check options type

	for _, op := range Options {
		value := vip.Get(op.name)
		
		// Validate value
		if err := op.val(value); err != nil {
			msg := fmt.Sprintf("Config Error: %v -> %v", op.name, err.Error())
			return nil, NewConfigError(msg)
		}

		conf[op.name] = value
	}

	return conf, nil
}

// InitViper initializes options and flags
func InitOptions(vip *viper.Viper) error {
	// Add default values for all the options
	for _, op := range Options {
		vip.SetDefault(op.name, op.def) 
	}

	return nil
}

// Read and parser configuration from file
func ReadConfig(filepath string, filename string) (conf map[string]interface{}, err error){

	v := viper.New()

	v.SetConfigType("toml")

	// Initialize 
	if err = InitOptions(v); err != nil {
		return nil, err
	}

	// Use default filename if one wasn't provided
	if filename != "" {
		v.SetConfigName(filename)
	} else {
		v.SetConfigName(DefaultConfigFilename)
	}

	// Use default filepath if it wasn't provided 
	if filepath != "" {
		v.AddConfigPath(filepath)
	} else {
		v.AddConfigPath(DefaultConfigPath)
	}

	// Try to read configuration file
	err = v.ReadInConfig()
	if err != nil {
		return nil, err
	}

	// Extract configuration
	return ParseConfig(v)
}

// LoadConfig usign default config file path
func LoadConfig() (map[string]interface{}, error) {
	return ReadConfig("", "")
}
