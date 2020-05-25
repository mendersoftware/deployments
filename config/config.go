// Copyright 2020 Northern.tech AS
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
	"fmt"
	"os"

	"github.com/mendersoftware/go-lib-micro/config"
)

const (
	EnvProd = "prod"
	EnvDev  = "dev"

	SettingHttps            = "https"
	SettingHttpsCertificate = SettingHttps + ".certificate"
	SettingHttpsKey         = SettingHttps + ".key"

	SettingListen        = "listen"
	SettingListenDefault = ":8080"

	SettingsAws                       = "aws"
	SettingAwsS3Region                = SettingsAws + ".region"
	SettingAwsS3RegionDefault         = "us-east-1"
	SettingAwsS3Bucket                = SettingsAws + ".bucket"
	SettingAwsS3BucketDefault         = "mender-artifact-storage"
	SettingAwsS3ForcePathStyle        = SettingsAws + ".force_path_style"
	SettingAwsS3ForcePathStyleDefault = true
	SettingAwsURI                     = SettingsAws + ".uri"
	SettingsAwsTagArtifact            = SettingsAws + ".tag_artifact"
	SettingsAwsTagArtifactDefault     = false

	SettingsAwsAuth      = SettingsAws + ".auth"
	SettingAwsAuthKeyId  = SettingsAwsAuth + ".key"
	SettingAwsAuthSecret = SettingsAwsAuth + ".secret"
	SettingAwsAuthToken  = SettingsAwsAuth + ".token"

	SettingMongo        = "mongo-url"
	SettingMongoDefault = "mongodb://mongo-deployments:27017"

	SettingDbSSL        = "mongo_ssl"
	SettingDbSSLDefault = false

	SettingDbSSLSkipVerify        = "mongo_ssl_skipverify"
	SettingDbSSLSkipVerifyDefault = false

	SettingDbUsername = "mongo_username"
	SettingDbPassword = "mongo_password"

	SettingGateway        = "mender-gateway"
	SettingGatewayDefault = "localhost:9080"

	SettingWorkflows        = "mender-workflows"
	SettingWorkflowsDefault = "http://mender-workflows-server:8080"

	SettingMiddleware        = "middleware"
	SettingMiddlewareDefault = EnvProd

	SettingInventoryAddr        = "inventory_addr"
	SettingInventoryAddrDefault = "http://mender-inventory:8080"

	SettingInventoryTimeout        = "inventory_timeout"
	SettingInventoryTimeoutDefault = 10
)

// ValidateAwsAuth validates configuration of SettingsAwsAuth section if provided.
func ValidateAwsAuth(c config.Reader) error {

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
func ValidateHttps(c config.Reader) error {

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

// Generate error with missing required option message.
func MissingOptionError(option string) error {
	return fmt.Errorf("Required option: '%s'", option)
}

var (
	Validators = []config.Validator{ValidateAwsAuth, ValidateHttps}
	Defaults   = []config.Default{
		{Key: SettingListen, Value: SettingListenDefault},
		{Key: SettingAwsS3Region, Value: SettingAwsS3RegionDefault},
		{Key: SettingAwsS3Bucket, Value: SettingAwsS3BucketDefault},
		{Key: SettingAwsS3ForcePathStyle, Value: SettingAwsS3ForcePathStyleDefault},
		{Key: SettingMongo, Value: SettingMongoDefault},
		{Key: SettingDbSSL, Value: SettingDbSSLDefault},
		{Key: SettingDbSSLSkipVerify, Value: SettingDbSSLSkipVerifyDefault},
		{Key: SettingGateway, Value: SettingGatewayDefault},
		{Key: SettingWorkflows, Value: SettingWorkflowsDefault},
		{Key: SettingsAwsTagArtifact, Value: SettingsAwsTagArtifactDefault},
	}
)
