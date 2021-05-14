// Copyright 2021 Northern.tech AS
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
	SettingAwsS3UseAccelerate         = SettingsAws + ".use_accelerate"
	SettingAwsS3UseAccelerateDefault  = false
	SettingAwsURI                     = SettingsAws + ".uri"
	SettingsAwsTagArtifact            = SettingsAws + ".tag_artifact"
	SettingsAwsTagArtifactDefault     = false

	SettingsAwsDownloadExpireSeconds        = SettingsAws + ".download_expire_seconds"
	SettingsAwsDownloadExpireSecondsDefault = 900
	SettingsAwsUploadExpireSeconds          = SettingsAws + ".upload_expire_seconds"
	SettingsAwsUploadExpireSecondsDefault   = 3600

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

	// SettingPresignAlgorithm sets the algorithm used for signing
	// downloadable URLs. This option is currently ignored.
	SettingPresignAlgorithm        = "presign.algorithm"
	SettingPresignAlgorithmDefault = "HMAC256"

	// SettingPresignSecret sets the secret for generating signed url.
	// For HMAC type of algorithms the value must be a base64 encoded
	// secret. For public key signatures, the value must be a path to
	// the private key (not yet supported).
	SettingPresignSecret        = "presign.secret"
	SettingPresignSecretDefault = ""

	// SettingPresignExpireSeconds sets the amount of seconds it takes for
	// the signed URL to expire.
	SettingPresignExpireSeconds        = "presign.expire_seconds"
	SettingPresignExpireSecondsDefault = 900

	// SettingPresignHost sets the URL hostname (pointing to the gateway)
	// for the generated URL. If the configuration option is left blank
	// (default), it will try to use the X-Forwarded-Host header forwarded
	// by the proxy.
	SettingPresignHost        = "presign.url_hostname"
	SettingPresignHostDefault = ""

	// SettingPresignURLScheme sets the URL scheme used for generating the
	// pre-signed url.
	SettingPresignScheme        = "presign.url_scheme"
	SettingPresignSchemeDefault = "https"
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
		{Key: SettingAwsS3UseAccelerate, Value: SettingAwsS3UseAccelerateDefault},
		{Key: SettingsAwsDownloadExpireSeconds, Value: SettingsAwsDownloadExpireSecondsDefault},
		{Key: SettingsAwsUploadExpireSeconds, Value: SettingsAwsUploadExpireSecondsDefault},
		{Key: SettingMongo, Value: SettingMongoDefault},
		{Key: SettingDbSSL, Value: SettingDbSSLDefault},
		{Key: SettingDbSSLSkipVerify, Value: SettingDbSSLSkipVerifyDefault},
		{Key: SettingGateway, Value: SettingGatewayDefault},
		{Key: SettingWorkflows, Value: SettingWorkflowsDefault},
		{Key: SettingsAwsTagArtifact, Value: SettingsAwsTagArtifactDefault},
		{Key: SettingInventoryAddr, Value: SettingInventoryAddrDefault},
		{Key: SettingInventoryTimeout, Value: SettingInventoryTimeoutDefault},
		{Key: SettingPresignAlgorithm, Value: SettingPresignAlgorithmDefault},
		{Key: SettingPresignSecret, Value: SettingPresignSecretDefault},
		{Key: SettingPresignExpireSeconds, Value: SettingPresignExpireSecondsDefault},
		{Key: SettingPresignHost, Value: SettingPresignHostDefault},
		{Key: SettingPresignScheme, Value: SettingPresignSchemeDefault},
	}
)
