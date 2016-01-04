package config

import "time"

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
