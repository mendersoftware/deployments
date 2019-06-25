// Copyright 2019 Northern.tech AS
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

package http

import (
	"crypto/tls"
	"net"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/globalsign/mgo"
	"github.com/pkg/errors"

	"github.com/mendersoftware/go-lib-micro/config"

	"github.com/mendersoftware/deployments/app"
	dconfig "github.com/mendersoftware/deployments/config"
	deploymentsController "github.com/mendersoftware/deployments/resources/deployments/controller"
	deploymentsView "github.com/mendersoftware/deployments/resources/deployments/view"
	imagesController "github.com/mendersoftware/deployments/resources/images/controller"
	tenantsController "github.com/mendersoftware/deployments/resources/tenants/controller"
	"github.com/mendersoftware/deployments/s3"
	"github.com/mendersoftware/deployments/store/mongo"
	"github.com/mendersoftware/deployments/utils/restutil"
	"github.com/mendersoftware/deployments/utils/restutil/view"
)

const (
	ApiUrlInternal   = "/api/internal/v1/deployments"
	ApiUrlManagement = "/api/management/v1/deployments"
	ApiUrlDevices    = "/api/devices/v1/deployments"

	ApiUrlManagementArtifacts = ApiUrlManagement + "/artifacts"
)

func SetupS3(c config.Reader) (s3.FileStorage, error) {

	bucket := c.GetString(dconfig.SettingAwsS3Bucket)
	region := c.GetString(dconfig.SettingAwsS3Region)

	if c.IsSet(dconfig.SettingsAwsAuth) || (c.IsSet(dconfig.SettingAwsAuthKeyId) && c.IsSet(dconfig.SettingAwsAuthSecret) && c.IsSet(dconfig.SettingAwsURI)) {
		return s3.NewSimpleStorageServiceStatic(
			bucket,
			c.GetString(dconfig.SettingAwsAuthKeyId),
			c.GetString(dconfig.SettingAwsAuthSecret),
			region,
			c.GetString(dconfig.SettingAwsAuthToken),
			c.GetString(dconfig.SettingAwsURI),
			c.GetBool(dconfig.SettingsAwsTagArtifact),
		)
	}

	return s3.NewSimpleStorageServiceDefaults(bucket, region)
}

func NewMongoSession(c config.Reader) (*mgo.Session, error) {

	dialInfo, err := mgo.ParseURL(c.GetString(dconfig.SettingMongo))
	if err != nil {
		return nil, errors.Wrap(err, "failed to open mgo session")
	}

	// Set 10s timeout - same as set by Dial
	dialInfo.Timeout = 10 * time.Second

	username := c.GetString(dconfig.SettingDbUsername)
	if username != "" {
		dialInfo.Username = username
	}

	passward := c.GetString(dconfig.SettingDbPassword)
	if passward != "" {
		dialInfo.Password = passward
	}

	if c.GetBool(dconfig.SettingDbSSL) {
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {

			// Setup TLS
			tlsConfig := &tls.Config{}
			tlsConfig.InsecureSkipVerify = c.GetBool(dconfig.SettingDbSSLSkipVerify)

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
func NewRouter(c config.Reader) (rest.App, error) {

	dbSession, err := NewMongoSession(c)
	if err != nil {
		return nil, err
	}

	// Storage Layer
	fileStorage, err := SetupS3(c)
	if err != nil {
		return nil, err
	}
	mongoStorage := mongo.NewDataStoreMongoWithSession(dbSession)

	app := app.NewDeployments(mongoStorage, fileStorage, app.ArtifactContentType)

	// Controllers
	imagesController := imagesController.NewSoftwareImagesController(app,
		new(view.RESTView))
	deploymentsController := deploymentsController.NewDeploymentsController(app,
		new(deploymentsView.DeploymentsView))

	tenantsController := tenantsController.NewController(app,
		imagesController,
		new(view.RESTView))

	deploymentsHandlers := NewDeploymentsApiHandlers(mongoStorage, new(view.RESTView), app)

	// Routing
	imageRoutes := NewImagesResourceRoutes(imagesController)
	deploymentsRoutes := NewDeploymentsResourceRoutes(deploymentsController)
	limitsRoutes := NewLimitsResourceRoutes(deploymentsHandlers)
	tenantsRoutes := TenantRoutes(tenantsController)
	releasesRoutes := ReleasesRoutes(deploymentsHandlers)

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

func NewLimitsResourceRoutes(controller *DeploymentsApiHandlers) []*rest.Route {

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

func ReleasesRoutes(controller *DeploymentsApiHandlers) []*rest.Route {
	if controller == nil {
		return []*rest.Route{}
	}

	return []*rest.Route{
		rest.Get(ApiUrlManagement+"/deployments/releases", controller.GetReleases),
	}
}
