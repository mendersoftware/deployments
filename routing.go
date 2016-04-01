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
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/artifacts/config"
	"github.com/mendersoftware/artifacts/controllers"
	"github.com/mendersoftware/artifacts/handlers"
	"github.com/mendersoftware/artifacts/models"
	"github.com/mendersoftware/artifacts/models/fileservice"
	"github.com/mendersoftware/artifacts/models/fileservice/s3"
	"github.com/mendersoftware/artifacts/models/images/memmap"
	"github.com/mendersoftware/artifacts/mvc"
	"github.com/mendersoftware/artifacts/utils/safemap"
	"gopkg.in/mgo.v2"
)

func SetupFileStorage(c config.ConfigReader) fileservice.FileServiceModelI {

	bucket := c.GetString(SettingAweS3Bucket)
	region := c.GetString(SettingAwsS3Region)

	if c.IsSet(SettingsAwsAuth) {
		return s3.NewSimpleStorageServiceStatic(
			bucket,
			c.GetString(SettingAwsAuthKeyId),
			c.GetString(SettingAwsAuthSecret),
			region,
			c.GetString(SettingAwsAuthToken),
		)
	}

	return s3.NewSimpleStorageServiceDefaults(bucket, region)
}

// NewRouter defines all REST API routes.
func NewRouter(c config.ConfigReader) (rest.App, error) {
	version := handlers.NewVersion(CreateVersionString(), BuildNumber)

	// Image management
	imagesStorage := memmap.NewImagesInMem(safemap.NewStringMap())
	fileStorage := SetupFileStorage(c)
	imagesController := controllers.NewImagesController(imagesStorage, fileStorage)
	meta := handlers.NewImageMeta(imagesController)

	// Software deployments
	dbSession, err := mgo.Dial(c.GetString(SettingMongo))
	if err != nil {
		return nil, err
	}
	deploymentModel := models.NewDeploymentModel(dbSession, imagesController)

	// Define routers and autogen OPTIONS method for each route.
	routes := []*rest.Route{

		rest.Get("/api", version.Get),

		// Images
		rest.Get("/api/0.0.1/images", meta.Lookup),
		rest.Post("/api/0.0.1/images", meta.Create),
		rest.Get("/api/0.0.1/images/:id", meta.Get),
		rest.Put("/api/0.0.1/images/:id", meta.Edit),
		rest.Delete("/api/0.0.1/images/:id", meta.Delete),
		rest.Get("/api/0.0.1/images/:id/upload", meta.UploadLink),
		rest.Get("/api/0.0.1/images/:id/download", meta.DownloadLink),

		// Deployments
		rest.Post("/api/0.0.1/deployments", mvc.NewCreateController(deploymentModel, mvc.NewViewRestPost("/api/0.0.1/deployments"))),
		rest.Get("/api/0.0.1/deployments/:id", mvc.NewGetObjectController(deploymentModel, mvc.NewViewRestGetOne())),

		// Devices
		rest.Get("/api/0.0.1/devices/:id/update", mvc.NewGetObjectController(models.NewDeviceUpdateModel(dbSession, fileStorage), mvc.NewViewRestGetOne())),
	}

	return rest.MakeRouter(AutogenOptionsRoutes(handlers.NewOptionsHandler, routes...)...)
}

// Automatically add OPTIONS method support for each defined route.
func AutogenOptionsRoutes(createHandler handlers.CreateOptionsHandler, routes ...*rest.Route) []*rest.Route {

	methodGroups := make(map[string][]string, len(routes))

	for _, route := range routes {
		methods, ok := methodGroups[route.PathExp]
		if !ok {
			methods = make([]string, 0, 0)
		}

		methodGroups[route.PathExp] = append(methods, route.HttpMethod)
	}

	options := make([]*rest.Route, 0, len(methodGroups))
	for route, methods := range methodGroups {
		options = append(options, rest.Options(route, createHandler(methods...)))
	}

	return append(routes, options...)
}
