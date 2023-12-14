// Copyright 2023 Northern.tech AS
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
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/mendersoftware/go-lib-micro/log"

	api "github.com/mendersoftware/deployments/api/http"
	"github.com/mendersoftware/deployments/app"
	"github.com/mendersoftware/deployments/client/reporting"
	dconfig "github.com/mendersoftware/deployments/config"
	"github.com/mendersoftware/deployments/storage"
	"github.com/mendersoftware/deployments/storage/azblob"
	"github.com/mendersoftware/deployments/storage/manager"
	"github.com/mendersoftware/deployments/storage/s3"
	mstore "github.com/mendersoftware/deployments/store/mongo"
)

func SetupS3(ctx context.Context, defaultOptions *s3.Options) (storage.ObjectStorage, error) {
	c := config.Config

	bucket := c.GetString(dconfig.SettingStorageBucket)

	// Copy / merge defaultOptions
	options := s3.NewOptions(defaultOptions).
		SetBucketName(bucket).
		SetForcePathStyle(c.GetBool(dconfig.SettingAwsS3ForcePathStyle))

	// The following parameters falls back on AWS_* environment if not set
	if c.IsSet(dconfig.SettingAwsS3Region) {
		options.SetRegion(c.GetString(dconfig.SettingAwsS3Region))
	}
	if c.IsSet(dconfig.SettingsAwsAuth) ||
		(c.IsSet(dconfig.SettingAwsAuthKeyId) &&
			c.IsSet(dconfig.SettingAwsAuthSecret)) {
		options.SetStaticCredentials(
			c.GetString(dconfig.SettingAwsAuthKeyId),
			c.GetString(dconfig.SettingAwsAuthSecret),
			c.GetString(dconfig.SettingAwsAuthToken),
		)
	}
	useAccelerate := c.GetBool(dconfig.SettingAwsS3UseAccelerate)
	if c.IsSet(dconfig.SettingAwsURI) {
		options.SetURI(c.GetString(dconfig.SettingAwsURI))
		if useAccelerate {
			log.FromContext(ctx).
				Warn(`cannot use s3 transfer acceleration with custom "uri": ` +
					"disabling transfer acceleration")
		}
	} else {
		options.SetUseAccelerate(c.GetBool(dconfig.SettingAwsS3UseAccelerate))
	}
	if c.IsSet(dconfig.SettingAwsExternalURI) {
		options.SetExternalURI(c.GetString(dconfig.SettingAwsExternalURI))
	}
	if c.IsSet(dconfig.SettingStorageProxyURI) {
		rawURL := c.GetString(dconfig.SettingStorageProxyURI)
		proxyURL, err := url.Parse(rawURL)
		if err != nil {
			return nil, errors.WithMessage(err, "invalid setting `storage.proxy_uri`")
		}
		options.SetProxyURI(proxyURL)
	}
	if c.IsSet(dconfig.SettingAwsUnsignedHeaders) {
		options.SetUnsignedHeaders(c.GetStringSlice(dconfig.SettingAwsUnsignedHeaders))
	}

	storage, err := s3.New(ctx, options)
	return storage, err
}

func SetupBlobStorage(
	ctx context.Context,
	defaultOptions *azblob.Options,
) (storage.ObjectStorage, error) {
	c := config.Config

	// Copy / merge options
	options := azblob.NewOptions(defaultOptions)

	if c.IsSet(dconfig.SettingAzureConnectionString) {
		options.SetConnectionString(c.GetString(dconfig.SettingAzureConnectionString))
	} else if c.IsSet(dconfig.SettingAzureSharedKeyAccount) &&
		c.IsSet(dconfig.SettingAzureSharedKeyAccountKey) {
		creds := azblob.SharedKeyCredentials{
			AccountName: c.GetString(dconfig.SettingAzureSharedKeyAccount),
			AccountKey:  c.GetString(dconfig.SettingAzureSharedKeyAccountKey),
		}
		if c.IsSet(dconfig.SettingAzureSharedKeyURI) {
			uri := c.GetString(dconfig.SettingAzureSharedKeyURI)
			creds.URI = &uri
		}
		options.SetSharedKey(creds)
	}
	if c.IsSet(dconfig.SettingStorageProxyURI) {
		rawURL := c.GetString(dconfig.SettingStorageProxyURI)
		proxyURL, err := url.Parse(rawURL)
		if err != nil {
			return nil, errors.WithMessage(err, `invalid setting "storage.proxy_uri"`)
		}
		if !strings.HasPrefix(strings.ToLower(proxyURL.Scheme), "https") {
			log.FromContext(ctx).
				Warnf(`setting "storage.proxy_uri" (%s) is not using https`, rawURL)
		}
		options.SetProxyURI(proxyURL)
	}
	return azblob.New(ctx, c.GetString(dconfig.SettingStorageBucket), options)
}

func SetupObjectStorage(ctx context.Context) (objManager storage.ObjectStorage, err error) {
	c := config.Config

	// Calculate s3 multipart buffer size: the minimum buffer size that
	// covers the maximum image size aligned to multiple of 5MiB.
	maxImageSize := c.GetInt64(dconfig.SettingStorageMaxImageSize)
	bufferSize := (((maxImageSize - 1) /
		(s3.MultipartMaxParts * s3.MultipartMinSize)) + 1) *
		s3.MultipartMinSize
	var (
		s3Options = s3.NewOptions().
				SetContentType(app.ArtifactContentType).
				SetBufferSize(int(bufferSize))
		azOptions = azblob.NewOptions().
				SetContentType(app.ArtifactContentType)
	)
	var defaultStorage storage.ObjectStorage
	switch defType := c.GetString(dconfig.SettingDefaultStorage); defType {
	case dconfig.StorageTypeAWS:
		defaultStorage, err = SetupS3(ctx, s3Options)
	case dconfig.StorageTypeAzure:
		defaultStorage, err = SetupBlobStorage(ctx, azOptions)
	default:
		err = errors.Errorf(
			`storage type must be one of %q or %q, received value %q`,
			dconfig.StorageTypeAWS, dconfig.StorageTypeAzure, defType,
		)
	}
	if err != nil {
		return nil, err
	}
	return manager.New(ctx, defaultStorage, s3Options, azOptions)
}

func RunServer(ctx context.Context) error {
	c := config.Config
	dbClient, err := mstore.NewMongoClient(ctx, c)
	if err != nil {
		return err
	}
	defer func() {
		_ = dbClient.Disconnect(context.Background())
	}()

	ds := mstore.NewDataStoreMongoWithClient(dbClient)

	// Storage Layer
	objStore, err := SetupObjectStorage(ctx)
	if err != nil {
		return errors.WithMessage(err, "main: failed to setup storage client")
	}

	app := app.NewDeployments(ds, objStore, 0, false)
	if addr := c.GetString(dconfig.SettingReportingAddr); addr != "" {
		c := reporting.NewClient(addr)
		app = app.WithReporting(c)
	}

	// Setup API Router configuration
	base64Repl := strings.NewReplacer("-", "+", "_", "/", "=", "")
	expireSec := c.GetDuration(dconfig.SettingPresignExpireSeconds)
	apiConf := api.NewConfig().
		SetPresignExpire(time.Second * expireSec).
		SetPresignHostname(c.GetString(dconfig.SettingPresignHost)).
		SetPresignScheme(c.GetString(dconfig.SettingPresignScheme)).
		SetMaxImageSize(c.GetInt64(dconfig.SettingStorageMaxImageSize)).
		SetEnableDirectUpload(c.GetBool(dconfig.SettingStorageEnableDirectUpload)).
		SetEnableDirectUploadSkipVerify(c.GetBool(dconfig.SettingStorageDirectUploadSkipVerify)).
		SetDisableNewReleasesFeature(c.GetBool(dconfig.SettingDisableNewReleasesFeature))
	if key, err := base64.RawStdEncoding.DecodeString(
		base64Repl.Replace(
			c.GetString(dconfig.SettingPresignSecret),
		),
	); err == nil {
		apiConf.SetPresignSecret(key)
	}
	handler, err := api.NewHandler(ctx, app, ds, apiConf)
	if err != nil {
		return err
	}

	listen := c.GetString(dconfig.SettingListen)

	if c.IsSet(dconfig.SettingHttps) {

		cert := c.GetString(dconfig.SettingHttpsCertificate)
		key := c.GetString(dconfig.SettingHttpsKey)

		return http.ListenAndServeTLS(listen, cert, key, handler)
	}

	return http.ListenAndServe(listen, handler)
}
