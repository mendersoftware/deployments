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
	"os"

	"github.com/mendersoftware/deployments/config"
)

const (
	SettingHttps            = "https"
	SettingHttpsCertificate = SettingHttps + ".certificate"
	SettingHttpsKey         = SettingHttps + ".key"

	SettingListen        = "listen"
	SettingListenDefault = ":8080"

	SettingsAws               = "aws"
	SettingAwsS3Region        = SettingsAws + ".region"
	SettingAwsS3RegionDefault = "us-east-1"
	SettingAwsS3Bucket        = SettingsAws + ".bucket"
	SettingAwsS3BucketDefault = "mender-artifact-storage"
	SettingAwsURI             = SettingsAws + ".uri"

	SettingsAwsAuth      = SettingsAws + ".auth"
	SettingAwsAuthKeyId  = SettingsAwsAuth + ".key"
	SettingAwsAuthSecret = SettingsAwsAuth + ".secret"
	SettingAwsAuthToken  = SettingsAwsAuth + ".token"

	SettingMongo        = "mongo-url"
	SettingMongoDefault = "mongo-deployments"

	SettingGateway        = "mender-gateway"
	SettingGatewayDefault = "localhost:9080"
)

// ValidateAwsAuth validates configuration of SettingsAwsAuth section if provided.
func ValidateAwsAuth(c config.ConfigReader) error {

	if c.IsSet(SettingsAwsAuth) {
		required := []string{SettingAwsAuthKeyId, SettingAwsAuthSecret}
		for _, key := range required {
			if !c.IsSet(key) {
				return MissingOptionError(key)
			}

			if c.GetString(key) == "" {
				return MissingOptionError(key)
			}
		}
	}

	return nil
}

// ValidateHttps validates configuration of SettingHttps section if provided.
func ValidateHttps(c config.ConfigReader) error {

	if c.IsSet(SettingHttps) {
		required := []string{SettingHttpsCertificate, SettingHttpsKey}
		for _, key := range required {
			if !c.IsSet(key) {
				return MissingOptionError(key)
			}

			value := c.GetString(key)
			if value == "" {
				return MissingOptionError(key)
			}

			if _, err := os.Stat(value); err != nil {
				return err
			}
		}
	}

	return nil
}

// Generate error with missing reuired option message.
func MissingOptionError(option string) error {
	return fmt.Errorf("Required option: '%s'", option)
}
