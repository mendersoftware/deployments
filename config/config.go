// Copyright 2017 Northern.tech AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

var (
	Config = viper.New()
)

type ConfigReader interface {
	Get(key string) interface{}
	GetBool(key string) bool
	GetFloat64(key string) float64
	GetInt(key string) int
	GetString(key string) string
	GetStringMap(key string) map[string]interface{}
	GetStringMapString(key string) map[string]string
	GetStringSlice(key string) []string
	GetTime(key string) time.Time
	GetDuration(key string) time.Duration
	IsSet(key string) bool
}

func FromConfigFile(filePath string,
	defaults []Default,
	configValidators ...Validator) error {

	// map settings such as foo.bar and foo-bar to FOO_BAR environment keys
	Config.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// Enable setting also other conig values by environment variables
	Config.SetEnvPrefix("DEPLOYMENTS")
	Config.AutomaticEnv()

	// Set default values for config
	SetDefaults(Config, defaults)

	// Find and read the config file
	if filePath != "" {
		Config.SetConfigFile(filePath)
		if err := Config.ReadInConfig(); err != nil {
			return err
		}
	}

	// Validate config
	if err := ValidateConfig(Config, configValidators...); err != nil {
		return err
	}

	return nil
}

type Validator func(c ConfigReader) error

// ValidateConfig validates conifg accroding to provided validators.
func ValidateConfig(c ConfigReader, validators ...Validator) error {

	for _, validator := range validators {
		err := validator(c)
		if err != nil {
			return err
		}
	}

	return nil
}

type Writer interface {
	SetDefault(key string, val interface{})
	Set(key string, val interface{})
}

type Default struct {
	Key   string
	Value interface{}
}

func SetDefaults(c Writer, defaults []Default) {
	for _, def := range defaults {
		c.SetDefault(def.Key, def.Value)
	}
}
