// Copyright 2016 Mender Software AS
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
package main

import (
	"fmt"
	"testing"
	"time"
)

type MockConfigReader struct {
	settings map[string]string
}

func NewMockConfigReader() *MockConfigReader {
	return &MockConfigReader{
		settings: make(map[string]string),
	}
}

func (m *MockConfigReader) Get(key string) interface{}                      { return nil }
func (m *MockConfigReader) GetBool(key string) bool                         { return true }
func (m *MockConfigReader) GetFloat64(key string) float64                   { return 1.1 }
func (m *MockConfigReader) GetInt(key string) int                           { return 1 }
func (m *MockConfigReader) GetStringMap(key string) map[string]interface{}  { return nil }
func (m *MockConfigReader) GetStringMapString(key string) map[string]string { return nil }
func (m *MockConfigReader) GetStringSlice(key string) []string              { return []string{} }
func (m *MockConfigReader) GetTime(key string) time.Time                    { return time.Now() }
func (m *MockConfigReader) GetDuration(key string) time.Duration            { return time.Second }

func (m *MockConfigReader) GetString(key string) string {
	val, _ := m.settings[key]
	return val
}

func (m *MockConfigReader) IsSet(key string) bool {
	_, found := m.settings[key]
	return found
}

func (m *MockConfigReader) SetString(key, value string) {
	m.settings[key] = value
}

func TestMissingOptionErrror(t *testing.T) {
	if err := MissingOptionError("FIELD 1"); err == nil {
		t.FailNow()
	}
}

func TestValidateHttps(t *testing.T) {

	testList := []struct {
		out    error
		conifg *MockConfigReader
	}{
		{nil, NewMockConfigReader()},
		{MissingOptionError(SettingHttpsCertificate),
			func() *MockConfigReader {
				conf := NewMockConfigReader()
				conf.SetString(SettingHttps, "")
				return conf
			}()},
		{MissingOptionError(SettingHttpsCertificate),
			func() *MockConfigReader {
				conf := NewMockConfigReader()
				conf.SetString(SettingHttps, "")
				conf.SetString(SettingHttpsCertificate, "")
				return conf
			}()},
		{MissingOptionError(SettingHttpsKey),
			func() *MockConfigReader {
				conf := NewMockConfigReader()
				conf.SetString(SettingHttps, "")
				conf.SetString(SettingHttpsCertificate, "./config_test.go")
				return conf
			}()},
		{MissingOptionError(SettingHttpsKey),
			func() *MockConfigReader {
				conf := NewMockConfigReader()
				conf.SetString(SettingHttps, "")
				conf.SetString(SettingHttpsCertificate, "./config_test.go")
				conf.SetString(SettingHttpsKey, "")
				return conf
			}()},
		{nil,
			func() *MockConfigReader {
				conf := NewMockConfigReader()
				conf.SetString(SettingHttps, "")
				conf.SetString(SettingHttpsCertificate, "./config_test.go")
				conf.SetString(SettingHttpsKey, "./config_test.go")
				return conf
			}()},
	}

	for _, test := range testList {
		if test.out == nil {
			if err := ValidateHttps(test.conifg); err != test.out {
				t.FailNow()
			}
		} else if err := ValidateHttps(test.conifg); err.Error() != test.out.Error() {
			t.FailNow()
		}
	}
}

func TestValidateAwsAuth(t *testing.T) {

	testList := []struct {
		out    error
		conifg *MockConfigReader
	}{
		{nil, NewMockConfigReader()},
		{MissingOptionError(SettingAwsAuthKeyId),
			func() *MockConfigReader {
				conf := NewMockConfigReader()
				conf.SetString(SettingsAwsAuth, "")
				return conf
			}()},
		{MissingOptionError(SettingAwsAuthKeyId),
			func() *MockConfigReader {
				conf := NewMockConfigReader()
				conf.SetString(SettingsAwsAuth, "")
				conf.SetString(SettingAwsAuthKeyId, "")
				return conf
			}()},
		{MissingOptionError(SettingAwsAuthSecret),
			func() *MockConfigReader {
				conf := NewMockConfigReader()
				conf.SetString(SettingsAwsAuth, "")
				conf.SetString(SettingAwsAuthKeyId, "lala")
				return conf
			}()},
		{MissingOptionError(SettingAwsAuthSecret),
			func() *MockConfigReader {
				conf := NewMockConfigReader()
				conf.SetString(SettingsAwsAuth, "")
				conf.SetString(SettingAwsAuthKeyId, "lala")
				conf.SetString(SettingAwsAuthSecret, "")
				return conf
			}()},
		{nil,
			func() *MockConfigReader {
				conf := NewMockConfigReader()
				conf.SetString(SettingsAwsAuth, "")
				conf.SetString(SettingAwsAuthKeyId, "lala")
				conf.SetString(SettingAwsAuthSecret, "lala")
				return conf
			}()},
	}

	for _, test := range testList {
		if test.out == nil {
			if err := ValidateAwsAuth(test.conifg); err != test.out {
				fmt.Println(err, test.out)
				t.FailNow()
			}
		} else if err := ValidateAwsAuth(test.conifg); err.Error() != test.out.Error() {
			fmt.Println(err, test.out)
			t.FailNow()
		}
	}
}
