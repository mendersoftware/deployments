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
	"context"

	"github.com/ant0ine/go-json-rest/rest"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mendersoftware/go-lib-micro/config"

	"github.com/mendersoftware/deployments/app"
	dconfig "github.com/mendersoftware/deployments/config"
	"github.com/mendersoftware/deployments/s3"
	mstore "github.com/mendersoftware/deployments/store/mongo"
	"github.com/mendersoftware/deployments/utils/restutil"
	"github.com/mendersoftware/deployments/utils/restutil/view"
)

const (
	ApiUrlInternal   = "/api/internal/v1/deployments"
	ApiUrlManagement = "/api/management/v1/deployments"
	ApiUrlDevices    = "/api/devices/v1/deployments"

	ApiUrlManagementArtifacts           = ApiUrlManagement + "/artifacts"
	ApiUrlManagementArtifactsGenerate   = ApiUrlManagement + "/artifacts/generate"
	ApiUrlManagementArtifactsId         = ApiUrlManagement + "/artifacts/:id"
	ApiUrlManagementArtifactsIdDownload = ApiUrlManagement + "/artifacts/:id/download"

	ApiUrlManagementDeployments           = ApiUrlManagement + "/deployments"
	ApiUrlManagementDeploymentsId         = ApiUrlManagement + "/deployments/:id"
	ApiUrlManagementDeploymentsStatistics = ApiUrlManagement + "/deployments/:id/statistics"
	ApiUrlManagementDeploymentsStatus     = ApiUrlManagement + "/deployments/:id/status"
	ApiUrlManagementDeploymentsDevices    = ApiUrlManagement + "/deployments/:id/devices"
	ApiUrlManagementDeploymentsLog        = ApiUrlManagement + "/deployments/:id/devices/:devid/log"
	ApiUrlManagementDeploymentsDeviceId   = ApiUrlManagement + "/deployments/devices/:id"

	ApiUrlManagementReleases = ApiUrlManagement + "/deployments/releases"

	ApiUrlManagementLimitsName = ApiUrlManagement + "/limits/:name"

	ApiUrlDevicesDeploymentsNext  = ApiUrlDevices + "/device/deployments/next"
	ApiUrlDevicesDeploymentStatus = ApiUrlDevices + "/device/deployments/:id/status"
	ApiUrlDevicesDeploymentsLog   = ApiUrlDevices + "/device/deployments/:id/log"

	ApiUrlInternalTenants           = ApiUrlInternal + "/tenants"
	ApiUrlInternalTenantDeployments = ApiUrlInternal + "/tenants/:tenant/deployments"
	ApiUrlInternalTenantArtifacts   = ApiUrlInternal + "/tenants/:tenant/artifacts"
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

// NewRouter defines all REST API routes.
func NewRouter(ctx context.Context, c config.Reader,
	mongoClient *mongo.Client) (rest.App, error) {

	// Storage Layer
	fileStorage, err := SetupS3(c)
	if err != nil {
		return nil, err
	}
	mongoStorage := mstore.NewDataStoreMongoWithClient(mongoClient)

	app := app.NewDeployments(mongoStorage, fileStorage, app.ArtifactContentType)

	deploymentsHandlers := NewDeploymentsApiHandlers(mongoStorage, new(view.RESTView), app)

	// Routing
	imageRoutes := NewImagesResourceRoutes(deploymentsHandlers)
	deploymentsRoutes := NewDeploymentsResourceRoutes(deploymentsHandlers)
	limitsRoutes := NewLimitsResourceRoutes(deploymentsHandlers)
	tenantsRoutes := TenantRoutes(deploymentsHandlers)
	releasesRoutes := ReleasesRoutes(deploymentsHandlers)

	routes := append(releasesRoutes, deploymentsRoutes...)
	routes = append(routes, limitsRoutes...)
	routes = append(routes, tenantsRoutes...)
	routes = append(routes, imageRoutes...)

	return rest.MakeRouter(restutil.AutogenOptionsRoutes(restutil.NewOptionsHandler, routes...)...)
}

func NewImagesResourceRoutes(controller *DeploymentsApiHandlers) []*rest.Route {

	if controller == nil {
		return []*rest.Route{}
	}

	return []*rest.Route{
		rest.Post(ApiUrlManagementArtifacts, controller.NewImage),
		rest.Post(ApiUrlManagementArtifactsGenerate, controller.GenerateImage),
		rest.Get(ApiUrlManagementArtifacts, controller.ListImages),

		rest.Get(ApiUrlManagementArtifactsId, controller.GetImage),
		rest.Delete(ApiUrlManagementArtifactsId, controller.DeleteImage),
		rest.Put(ApiUrlManagementArtifactsId, controller.EditImage),

		rest.Get(ApiUrlManagementArtifactsIdDownload, controller.DownloadLink),
	}
}

func NewDeploymentsResourceRoutes(controller *DeploymentsApiHandlers) []*rest.Route {

	if controller == nil {
		return []*rest.Route{}
	}

	return []*rest.Route{

		// Deployments
		rest.Post(ApiUrlManagementDeployments, controller.PostDeployment),
		rest.Get(ApiUrlManagementDeployments, controller.LookupDeployment),
		rest.Get(ApiUrlManagementDeploymentsId, controller.GetDeployment),
		rest.Get(ApiUrlManagementDeploymentsStatistics, controller.GetDeploymentStats),
		rest.Put(ApiUrlManagementDeploymentsStatus, controller.AbortDeployment),
		rest.Get(ApiUrlManagementDeploymentsDevices,
			controller.GetDeviceStatusesForDeployment),
		rest.Get(ApiUrlManagementDeploymentsLog,
			controller.GetDeploymentLogForDevice),
		rest.Delete(ApiUrlManagementDeploymentsDeviceId,
			controller.DecommissionDevice),

		// Devices
		rest.Get(ApiUrlDevicesDeploymentsNext, controller.GetDeploymentForDevice),
		rest.Put(ApiUrlDevicesDeploymentStatus,
			controller.PutDeploymentStatusForDevice),
		rest.Put(ApiUrlDevicesDeploymentsLog,
			controller.PutDeploymentLogForDevice),
	}
}

func NewLimitsResourceRoutes(controller *DeploymentsApiHandlers) []*rest.Route {

	if controller == nil {
		return []*rest.Route{}
	}

	return []*rest.Route{
		// limits
		rest.Get(ApiUrlManagementLimitsName, controller.GetLimit),
	}
}

func TenantRoutes(controller *DeploymentsApiHandlers) []*rest.Route {
	if controller == nil {
		return []*rest.Route{}
	}

	return []*rest.Route{
		rest.Post(ApiUrlInternalTenants, controller.ProvisionTenantsHandler),
		rest.Get(ApiUrlInternalTenantDeployments, controller.DeploymentsPerTenantHandler),
		rest.Post(ApiUrlInternalTenantArtifacts, controller.NewImageForTenantHandler),
	}
}

func ReleasesRoutes(controller *DeploymentsApiHandlers) []*rest.Route {
	if controller == nil {
		return []*rest.Route{}
	}

	return []*rest.Route{
		rest.Get(ApiUrlManagementReleases, controller.GetReleases),
	}
}
