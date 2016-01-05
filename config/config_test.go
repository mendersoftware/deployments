package config

import (
	"errors"
	"testing"
	"time"
)

type MockConfigReader struct{}

func (m *MockConfigReader) Get(key string) interface{}                      { return nil }
func (m *MockConfigReader) GetBool(key string) bool                         { return true }
func (m *MockConfigReader) GetFloat64(key string) float64                   { return 1.1 }
func (m *MockConfigReader) GetInt(key string) int                           { return 1 }
func (m *MockConfigReader) GetString(key string) string                     { return "some string" }
func (m *MockConfigReader) GetStringMap(key string) map[string]interface{}  { return nil }
func (m *MockConfigReader) GetStringMapString(key string) map[string]string { return nil }
func (m *MockConfigReader) GetStringSlice(key string) []string              { return []string{} }
func (m *MockConfigReader) GetTime(key string) time.Time                    { return time.Now() }
func (m *MockConfigReader) GetDuration(key string) time.Duration            { return time.Second }
func (m *MockConfigReader) IsSet(key string) bool                           { return true }

func TestValidateConfig(t *testing.T) {

	err := errors.New("test error")

	testList := []struct {
		out        error
		c          ConfigReader
		validators []Validator
	}{
		{nil, &MockConfigReader{}, []Validator{}},
		{err, &MockConfigReader{}, []Validator{func(c ConfigReader) error { return err }}},
	}

	for _, test := range testList {
		if ValidateConfig(test.c, test.validators...) != test.out {
			t.FailNow()
		}
	}
}
