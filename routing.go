// Copyright 2018 Northern.tech AS
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
	"crypto/tls"
	"net"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"

	"github.com/mendersoftware/deployments/config"
	deploymentsController "github.com/mendersoftware/deployments/resources/deployments/controller"
	deploymentsModel "github.com/mendersoftware/deployments/resources/deployments/model"
	deploymentsMongo "github.com/mendersoftware/deployments/resources/deployments/mongo"
	deploymentsView "github.com/mendersoftware/deployments/resources/deployments/view"
	imagesController "github.com/mendersoftware/deployments/resources/images/controller"
	imagesModel "github.com/mendersoftware/deployments/resources/images/model"
	imagesMongo "github.com/mendersoftware/deployments/resources/images/mongo"
	"github.com/mendersoftware/deployments/resources/images/s3"
	limitsController "github.com/mendersoftware/deployments/resources/limits/controller"
	limitsModel "github.com/mendersoftware/deployments/resources/limits/model"
	limitsMongo "github.com/mendersoftware/deployments/resources/limits/mongo"
	releasesController "github.com/mendersoftware/deployments/resources/releases/controller"
	releasesStore "github.com/mendersoftware/deployments/resources/releases/store"
	tenantsController "github.com/mendersoftware/deployments/resources/tenants/controller"
	tenantsModel "github.com/mendersoftware/deployments/resources/tenants/model"
	tenantsStore "github.com/mendersoftware/deployments/resources/tenants/store"
	"github.com/mendersoftware/deployments/utils/restutil"
	"github.com/mendersoftware/deployments/utils/restutil/view"
)

const (
	ApiUrlInternal   = "/api/internal/v1/deployments"
	ApiUrlManagement = "/api/management/v1/deployments"
	ApiUrlDevices    = "/api/devices/v1/deployments"

	ApiUrlManagementArtifacts = ApiUrlManagement + "/artifacts"
)

func SetupS3(c config.ConfigReader) (imagesModel.FileStorage, error) {

	bucket := c.GetString(SettingAwsS3Bucket)
	region := c.GetString(SettingAwsS3Region)

	if c.IsSet(SettingsAwsAuth) || (c.IsSet(SettingAwsAuthKeyId) && c.IsSet(SettingAwsAuthSecret) && c.IsSet(SettingAwsURI)) {
		return s3.NewSimpleStorageServiceStatic(
			bucket,
			c.GetString(SettingAwsAuthKeyId),
			c.GetString(SettingAwsAuthSecret),
			region,
			c.GetString(SettingAwsAuthToken),
			c.GetString(SettingAwsURI),
			c.GetBool(SettingsAwsTagArtifact),
		)
	}

	return s3.NewSimpleStorageServiceDefaults(bucket, region)
}

func NewMongoSession(c config.ConfigReader) (*mgo.Session, error) {

	dialInfo, err := mgo.ParseURL(c.GetString(SettingMongo))
	if err != nil {
		return nil, errors.Wrap(err, "failed to open mgo session")
	}

	// Set 10s timeout - same as set by Dial
	dialInfo.Timeout = 10 * time.Second

	username := c.GetString(SettingDbUsername)
	if username != "" {
		dialInfo.Username = username
	}

	passward := c.GetString(SettingDbPassword)
	if passward != "" {
		dialInfo.Password = passward
	}

	if c.GetBool(SettingDbSSL) {
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {

			// Setup TLS
			tlsConfig := &tls.Config{}
			tlsConfig.InsecureSkipVerify = c.GetBool(SettingDbSSLSkipVerify)

			conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
			return conn, err
		}
	}

	masterSession, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open mgo session")
	}

	// Validate connection
	if err := masterSession.Ping(); err != nil {
		return nil, errors.Wrap(err, "failed to open mgo session")
	}

	// force write ack with immediate journal file fsync
	masterSession.SetSafe(&mgo.Safe{
		W: 1,
		J: true,
	})

	return masterSession, nil
}

// NewRouter defines all REST API routes.
func NewRouter(c config.ConfigReader) (rest.App, error) {

	dbSession, err := NewMongoSession(c)
	if err != nil {
		return nil, err
	}

	// Storage Layer
	fileStorage, err := SetupS3(c)
	if err != nil {
		return nil, err
	}
	deploymentsStorage := deploymentsMongo.NewDeploymentsStorage(dbSession)
	deviceDeploymentsStorage := deploymentsMongo.NewDeviceDeploymentsStorage(dbSession)
	deviceDeploymentLogsStorage := deploymentsMongo.NewDeviceDeploymentLogsStorage(dbSession)
	imagesStorage := imagesMongo.NewSoftwareImagesStorage(dbSession)
	limitsStorage := limitsMongo.NewLimitsStorage(dbSession)
	tenantsStorage := tenantsStore.NewStore(dbSession)
	releasesStorage := releasesStore.NewStore(dbSession)

	// Domain Models
	deploymentModel := deploymentsModel.NewDeploymentModel(deploymentsModel.DeploymentsModelConfig{
		DeploymentsStorage:          deploymentsStorage,
		DeviceDeploymentsStorage:    deviceDeploymentsStorage,
		DeviceDeploymentLogsStorage: deviceDeploymentLogsStorage,
		ImageLinker:                 fileStorage,
		ArtifactGetter:              imagesStorage,
		ImageContentType:            imagesModel.ArtifactContentType,
	})

	imagesModel := imagesModel.NewImagesModel(fileStorage, deploymentModel, imagesStorage)
	limitsModel := limitsModel.NewLimitsModel(limitsStorage)
	tenantsModel := tenantsModel.NewModel(tenantsStorage)

	// Controllers
	imagesController := imagesController.NewSoftwareImagesController(imagesModel,
		new(view.RESTView))
	deploymentsController := deploymentsController.NewDeploymentsController(deploymentModel,
		new(deploymentsView.DeploymentsView))
	limitsController := limitsController.NewLimitsController(limitsModel,
		new(view.RESTView))

	tenantsController := tenantsController.NewController(tenantsModel,
		deploymentModel,
		imagesModel,
		imagesController,
		new(view.RESTView))

	releasesController := releasesController.NewReleasesController(releasesStorage, new(view.RESTView))

	// Routing
	imageRoutes := NewImagesResourceRoutes(imagesController)
	deploymentsRoutes := NewDeploymentsResourceRoutes(deploymentsController)
	limitsRoutes := NewLimitsResourceRoutes(limitsController)
	tenantsRoutes := TenantRoutes(tenantsController)
	releasesRoutes := ReleasesRoutes(releasesController)

	routes := append(releasesRoutes, deploymentsRoutes...)
	routes = append(routes, limitsRoutes...)
	routes = append(routes, tenantsRoutes...)
	routes = append(routes, imageRoutes...)

	return rest.MakeRouter(restutil.AutogenOptionsRoutes(restutil.NewOptionsHandler, routes...)...)
}

func NewImagesResourceRoutes(controller *imagesController.SoftwareImagesController) []*rest.Route {

	if controller == nil {
		return []*rest.Route{}
	}

	return []*rest.Route{
		rest.Post(ApiUrlManagementArtifacts, controller.NewImage),
		rest.Get(ApiUrlManagementArtifacts, controller.ListImages),

		rest.Get(ApiUrlManagement+"/artifacts/:id", controller.GetImage),
		rest.Delete(ApiUrlManagement+"/artifacts/:id", controller.DeleteImage),
		rest.Put(ApiUrlManagement+"/artifacts/:id", controller.EditImage),

		rest.Get(ApiUrlManagement+"/artifacts/:id/download", controller.DownloadLink),
	}
}

func NewDeploymentsResourceRoutes(controller *deploymentsController.DeploymentsController) []*rest.Route {

	if controller == nil {
		return []*rest.Route{}
	}

	return []*rest.Route{

		// Deployments
		rest.Post(ApiUrlManagement+"/deployments", controller.PostDeployment),
		rest.Get(ApiUrlManagement+"/deployments", controller.LookupDeployment),
		rest.Get(ApiUrlManagement+"/deployments/:id", controller.GetDeployment),
		rest.Get(ApiUrlManagement+"/deployments/:id/statistics", controller.GetDeploymentStats),
		rest.Put(ApiUrlManagement+"/deployments/:id/status", controller.AbortDeployment),
		rest.Get(ApiUrlManagement+"/deployments/:id/devices",
			controller.GetDeviceStatusesForDeployment),
		rest.Get(ApiUrlManagement+"/deployments/:id/devices/:devid/log",
			controller.GetDeploymentLogForDevice),
		rest.Delete(ApiUrlManagement+"/deployments/devices/:id",
			controller.DecommissionDevice),

		// Devices
		rest.Get(ApiUrlDevices+"/device/deployments/next", controller.GetDeploymentForDevice),
		rest.Put(ApiUrlDevices+"/device/deployments/:id/status",
			controller.PutDeploymentStatusForDevice),
		rest.Put(ApiUrlDevices+"/device/deployments/:id/log",
			controller.PutDeploymentLogForDevice),
	}
}

func NewLimitsResourceRoutes(controller *limitsController.LimitsController) []*rest.Route {

	if controller == nil {
		return []*rest.Route{}
	}

	return []*rest.Route{
		// limits
		rest.Get(ApiUrlManagement+"/limits/:name", controller.GetLimit),
	}
}

func TenantRoutes(controller *tenantsController.Controller) []*rest.Route {
	if controller == nil {
		return []*rest.Route{}
	}

	return []*rest.Route{
		rest.Post(ApiUrlInternal+"/tenants", controller.ProvisionTenantsHandler),
		rest.Get(ApiUrlInternal+"/tenants/:tenant/deployments", controller.DeploymentsPerTenantHandler),
		rest.Post(ApiUrlInternal+"/tenants/:tenant/artifacts", controller.NewImageForTenantHandler),
	}
}

func ReleasesRoutes(controller *releasesController.ReleasesController) []*rest.Route {
	if controller == nil {
		return []*rest.Route{}
	}

	return []*rest.Route{
		rest.Get(ApiUrlManagement+"/deployments/releases", controller.GetReleases),
	}
}
