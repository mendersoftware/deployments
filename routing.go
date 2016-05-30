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
	"github.com/mendersoftware/artifacts/deployments"
	"github.com/mendersoftware/artifacts/images"
	"github.com/mendersoftware/artifacts/mvc"
	"gopkg.in/mgo.v2"
)

func SetupS3(c config.ConfigReader) images.FileStorager {

	bucket := c.GetString(SettingAweS3Bucket)
	region := c.GetString(SettingAwsS3Region)

	if c.IsSet(SettingsAwsAuth) {
		return images.NewSimpleStorageServiceStatic(
			bucket,
			c.GetString(SettingAwsAuthKeyId),
			c.GetString(SettingAwsAuthSecret),
			region,
			c.GetString(SettingAwsAuthToken),
		)
	}

	return images.NewSimpleStorageServiceDefaults(bucket, region)
}

// NewRouter defines all REST API routes.
func NewRouter(c config.ConfigReader) (rest.App, error) {

	dbSession, err := mgo.Dial(c.GetString(SettingMongo))
	if err != nil {
		return nil, err
	}
	dbSession.SetSafe(&mgo.Safe{})

	// Storage Layer
	fileStorage := SetupS3(c)
	deploymentsStorage := deployments.NewDeploymentsStorage(dbSession)
	deviceDeploymentsStorage := deployments.NewDeviceDeploymentsStorage(dbSession)
	imagesStorage := images.NewSoftwareImagesStorage(dbSession)
	if err := imagesStorage.IndexStorage(); err != nil {
		return nil, err
	}

	// Domian Models
	deploymentModel := deployments.NewDeploymentModel(deploymentsStorage, imagesStorage, deviceDeploymentsStorage, fileStorage)
	imagesModel := images.NewImagesModel(fileStorage, deploymentModel, imagesStorage)

	// Controllers
	imagesController := images.NewSoftwareImagesController(imagesModel, mvc.RESTViewDefaults{})
	deploymentsController := deployments.NewDeploymentsController(deploymentModel, deployments.DeploymentsViews{})

	// Routing
	imageRoutes := NewImagesResourceRoutes(imagesController)
	deploymentsRoutes := NewDeploymentsResourceRoutes(deploymentsController)

	routes := append(imageRoutes, deploymentsRoutes...)

	return rest.MakeRouter(mvc.AutogenOptionsRoutes(mvc.NewOptionsHandler, routes...)...)
}

func NewImagesResourceRoutes(controller *images.SoftwareImagesController) []*rest.Route {

	if controller == nil {
		return []*rest.Route{}
	}

	return []*rest.Route{
		rest.Post("/api/0.0.1/images", controller.NewImage),
		rest.Get("/api/0.0.1/images", controller.ListImages),

		rest.Get("/api/0.0.1/images/:id", controller.GetImage),
		rest.Delete("/api/0.0.1/images/:id", controller.DeleteImage),
		rest.Put("/api/0.0.1/images/:id", controller.EditImage),

		rest.Get("/api/0.0.1/images/:id/upload", controller.UploadLink),
		rest.Get("/api/0.0.1/images/:id/download", controller.DownloadLink),
	}
}

func NewDeploymentsResourceRoutes(controller *deployments.DeploymentsController) []*rest.Route {

	if controller == nil {
		return []*rest.Route{}
	}

	return []*rest.Route{

		// Deployments
		rest.Post("/api/0.0.1/deployments", controller.PostDeployment),
		rest.Get("/api/0.0.1/deployments/:id", controller.GetDeployment),

		// Devices
		rest.Get("/api/0.0.1/devices/:id/update", controller.GetDeploymentForDevice),
	}
}
