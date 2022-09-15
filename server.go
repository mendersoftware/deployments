// Copyright 2022 Northern.tech AS
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
	"strings"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/pkg/errors"

	"github.com/mendersoftware/go-lib-micro/config"

	api "github.com/mendersoftware/deployments/api/http"
	"github.com/mendersoftware/deployments/app"
	"github.com/mendersoftware/deployments/client/reporting"
	dconfig "github.com/mendersoftware/deployments/config"
	"github.com/mendersoftware/deployments/storage"
	"github.com/mendersoftware/deployments/storage/s3"
	mstore "github.com/mendersoftware/deployments/store/mongo"
)

func SetupS3(ctx context.Context) (storage.ObjectStorage, error) {
	c := config.Config

	// Calculate multipart buffer size: the minimum buffer size that covers
	// the maximum image size aligned to multiple of 5MiB.
	maxImageSize := c.GetInt64(dconfig.SettingAwsS3MaxImageSize)
	bufferSize := (((maxImageSize - 1) /
		(s3.MultipartMaxParts * s3.MultipartMinSize)) + 1) *
		s3.MultipartMinSize

	// Compute the buffer size
	bucket := c.GetString(dconfig.SettingAwsS3Bucket)
	options := s3.NewOptions().
		SetContentType(app.ArtifactContentType).
		SetForcePathStyle(c.GetBool(dconfig.SettingAwsS3ForcePathStyle)).
		SetUseAccelerate(c.GetBool(dconfig.SettingAwsS3UseAccelerate)).
		SetBufferSize(int(bufferSize))

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
	if c.IsSet(dconfig.SettingAwsURI) {
		options.SetURI(c.GetString(dconfig.SettingAwsURI))
	}
	if c.IsSet(dconfig.SettingAwsExternalURI) {
		options.SetExternalURI(c.GetString(dconfig.SettingAwsExternalURI))
	}

	return s3.New(ctx, bucket, options)
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
	objStore, err := SetupS3(ctx)
	if err != nil {
		return errors.WithMessage(err, "main: failed to setup s3 client")
	}

	app := app.NewDeployments(ds, objStore)
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
		SetMaxImageSize(c.GetInt64(dconfig.SettingAwsS3MaxImageSize))
	if key, err := base64.RawStdEncoding.DecodeString(
		base64Repl.Replace(
			c.GetString(dconfig.SettingPresignSecret),
		),
	); err == nil {
		apiConf.SetPresignSecret(key)
	}
	router, err := api.NewRouter(ctx, app, ds, apiConf)
	if err != nil {
		return err
	}

	api := rest.NewApi()
	SetupMiddleware(c, api)
	api.SetApp(router)

	listen := c.GetString(dconfig.SettingListen)

	if c.IsSet(dconfig.SettingHttps) {

		cert := c.GetString(dconfig.SettingHttpsCertificate)
		key := c.GetString(dconfig.SettingHttpsKey)

		return http.ListenAndServeTLS(listen, cert, key, api.MakeHandler())
	}

	return http.ListenAndServe(listen, api.MakeHandler())
}
