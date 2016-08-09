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
	"github.com/mendersoftware/deployments/config"
	deploymentsController "github.com/mendersoftware/deployments/resources/deployments/controller"
	"github.com/mendersoftware/deployments/resources/deployments/generator"
	"github.com/mendersoftware/deployments/resources/deployments/inventory"
	deploymentsModel "github.com/mendersoftware/deployments/resources/deployments/model"
	deploymentsMongo "github.com/mendersoftware/deployments/resources/deployments/mongo"
	deploymentsView "github.com/mendersoftware/deployments/resources/deployments/view"
	imagesController "github.com/mendersoftware/deployments/resources/images/controller"
	imagesModel "github.com/mendersoftware/deployments/resources/images/model"
	imagesMongo "github.com/mendersoftware/deployments/resources/images/mongo"
	"github.com/mendersoftware/deployments/resources/images/s3"
	imagesView "github.com/mendersoftware/deployments/resources/images/view"
	"github.com/mendersoftware/deployments/utils/restutil"
	"gopkg.in/mgo.v2"
)

func SetupS3(c config.ConfigReader) imagesModel.FileStorage {

	bucket := c.GetString(SettingAweS3Bucket)
	region := c.GetString(SettingAwsS3Region)
	if c.IsSet(SettingsAwsAuth) || (c.IsSet(SettingAwsAuthKeyId) && c.IsSet(SettingAwsAuthSecret) && c.IsSet(SettingAwsURI)) {
		return s3.NewSimpleStorageServiceStatic(
			bucket,
			c.GetString(SettingAwsAuthKeyId),
			c.GetString(SettingAwsAuthSecret),
			region,
			c.GetString(SettingAwsAuthToken),
			c.GetString(SettingAwsURI),
		)
	}

	return s3.NewSimpleStorageServiceDefaults(bucket, region)
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
	deploymentsStorage := deploymentsMongo.NewDeploymentsStorage(dbSession)
	deviceDeploymentsStorage := deploymentsMongo.NewDeviceDeploymentsStorage(dbSession)
	deviceDeploymentLogsStorage := deploymentsMongo.NewDeviceDeploymentLogsStorage(dbSession)
	imagesStorage := imagesMongo.NewSoftwareImagesStorage(dbSession)
	if err := imagesStorage.IndexStorage(); err != nil {
		return nil, err
	}

	// Domian Models
	deploymentModel := deploymentsModel.NewDeploymentModel(
		deploymentsStorage,
		generator.NewImageBasedDeviceDeployment(
			imagesStorage,
			// can easily add configuration from main config file
			inventory.NewInventoryWithHardcodedType("TestDevice")),
		deviceDeploymentsStorage,
		deviceDeploymentLogsStorage,
		fileStorage,
	)

	imagesModel := imagesModel.NewImagesModel(fileStorage, deploymentModel, imagesStorage)

	// Controllers
	imagesController := imagesController.NewSoftwareImagesController(imagesModel, new(imagesView.RESTView))
	deploymentsController := deploymentsController.NewDeploymentsController(deploymentModel, new(deploymentsView.DeploymentsView))

	// Routing
	imageRoutes := NewImagesResourceRoutes(imagesController)
	deploymentsRoutes := NewDeploymentsResourceRoutes(deploymentsController)

	routes := append(imageRoutes, deploymentsRoutes...)

	return rest.MakeRouter(restutil.AutogenOptionsRoutes(restutil.NewOptionsHandler, routes...)...)
}

func NewImagesResourceRoutes(controller *imagesController.SoftwareImagesController) []*rest.Route {

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

func NewDeploymentsResourceRoutes(controller *deploymentsController.DeploymentsController) []*rest.Route {

	if controller == nil {
		return []*rest.Route{}
	}

	return []*rest.Route{

		// Deployments
		rest.Post("/api/0.0.1/deployments", controller.PostDeployment),
		rest.Get("/api/0.0.1/deployments", controller.LookupDeployment),
		rest.Get("/api/0.0.1/deployments/:id", controller.GetDeployment),
		rest.Get("/api/0.0.1/deployments/:id/statistics", controller.GetDeploymentStats),

		// Devices
		rest.Get("/api/0.0.1/device/update", controller.GetDeploymentForDevice),
		rest.Put("/api/0.0.1/device/deployments/:id/status",
			controller.PutDeploymentStatusForDevice),
		rest.Get("/api/0.0.1/deployments/:id/devices",
			controller.GetDeviceStatusesForDeployment),
		rest.Put("/api/0.0.1/device/deployments/:id/log",
			controller.PutDeploymentLogForDevice),
		rest.Get("/api/0.0.1/deployments/deployments/:id/devices/:devid/log",
			controller.GetDeploymentLogForDevice),
	}
}
