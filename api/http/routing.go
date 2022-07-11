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

package http

import (
	"context"
	"encoding/base64"
	"net/url"
	"strings"
	"time"

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
	ApiUrlManagementArtifactsId         = ApiUrlManagement + "/artifacts/#id"
	ApiUrlManagementArtifactsIdDownload = ApiUrlManagement + "/artifacts/#id/download"

	ApiUrlManagementDeployments            = ApiUrlManagement + "/deployments"
	ApiUrlManagementDeploymentsGroup       = ApiUrlManagement + "/deployments/group/#name"
	ApiUrlManagementDeploymentsId          = ApiUrlManagement + "/deployments/#id"
	ApiUrlManagementDeploymentsStatistics  = ApiUrlManagement + "/deployments/#id/statistics"
	ApiUrlManagementDeploymentsStatus      = ApiUrlManagement + "/deployments/#id/status"
	ApiUrlManagementDeploymentsDevices     = ApiUrlManagement + "/deployments/#id/devices"
	ApiUrlManagementDeploymentsDevicesList = ApiUrlManagement + "/deployments/#id/devices/list"
	ApiUrlManagementDeploymentsLog         = ApiUrlManagement +
		"/deployments/#id/devices/#devid/log"
	ApiUrlManagementDeploymentsDeviceId   = ApiUrlManagement + "/deployments/devices/#id"
	ApiUrlManagementDeploymentsDeviceList = ApiUrlManagement + "/deployments/#id/device_list"

	ApiUrlManagementReleases = ApiUrlManagement + "/deployments/releases"

	ApiUrlManagementLimitsName = ApiUrlManagement + "/limits/#name"

	ApiUrlDevicesDeploymentsNext  = ApiUrlDevices + "/device/deployments/next"
	ApiUrlDevicesDeploymentStatus = ApiUrlDevices + "/device/deployments/#id/status"
	ApiUrlDevicesDeploymentsLog   = ApiUrlDevices + "/device/deployments/#id/log"
	ApiUrlDevicesDownloadConfig   = ApiUrlDevices +
		"/download/configuration/#deployment_id/#device_type/#device_id"

	ApiUrlInternalAlive                   = ApiUrlInternal + "/alive"
	ApiUrlInternalHealth                  = ApiUrlInternal + "/health"
	ApiUrlInternalTenants                 = ApiUrlInternal + "/tenants"
	ApiUrlInternalTenantDeployments       = ApiUrlInternal + "/tenants/#tenant/deployments"
	ApiUrlInternalTenantDeploymentsDevice = ApiUrlInternal +
		"/tenants/#tenant/deployments/devices/#id"
	ApiUrlInternalTenantArtifacts       = ApiUrlInternal + "/tenants/#tenant/artifacts"
	ApiUrlInternalTenantStorageSettings = ApiUrlInternal +
		"/tenants/#tenant/storage/settings"
	ApiUrlInternalDeviceConfigurationDeployments = ApiUrlInternal +
		"/tenants/#tenant/configuration/deployments/#deployment_id/devices/#device_id"
)

func SetupS3(c config.Reader) (s3.FileStorage, error) {

	bucket := c.GetString(dconfig.SettingAwsS3Bucket)
	region := c.GetString(dconfig.SettingAwsS3Region)

	if c.IsSet(dconfig.SettingsAwsAuth) ||
		(c.IsSet(dconfig.SettingAwsAuthKeyId) &&
			c.IsSet(dconfig.SettingAwsAuthSecret) &&
			c.IsSet(dconfig.SettingAwsURI)) {
		return s3.NewSimpleStorageServiceStatic(
			bucket,
			c.GetString(dconfig.SettingAwsAuthKeyId),
			c.GetString(dconfig.SettingAwsAuthSecret),
			region,
			c.GetString(dconfig.SettingAwsAuthToken),
			c.GetString(dconfig.SettingAwsURI),
			c.GetBool(dconfig.SettingsAwsTagArtifact),
			c.GetBool(dconfig.SettingAwsS3ForcePathStyle),
			c.GetBool(dconfig.SettingAwsS3UseAccelerate),
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

	// Initialise a bucket, which is needed by Minio
	bucket := c.GetString(dconfig.SettingAwsS3Bucket)
	err = fileStorage.InitBucket(ctx, bucket)
	if err != nil {
		return nil, err
	}

	mongoStorage := mstore.NewDataStoreMongoWithClient(mongoClient)

	app := app.NewDeployments(mongoStorage, fileStorage, app.ArtifactContentType)

	// Create and configure API handlers
	//
	// Encode base64 secret in either std or URL encoding ignoring padding.
	base64Repl := strings.NewReplacer("-", "+", "_", "/", "=", "")
	expireSec := c.GetDuration(dconfig.SettingPresignExpireSeconds)
	apiConf := NewConfig().
		SetPresignExpire(time.Second * expireSec).
		SetPresignHostname(c.GetString(dconfig.SettingPresignHost)).
		SetPresignScheme(c.GetString(dconfig.SettingPresignScheme))
	// TODO: When adding support for different signing algorithm,
	//       conditionally decode this one:
	if key, err := base64.RawStdEncoding.DecodeString(
		base64Repl.Replace(
			c.GetString(dconfig.SettingPresignSecret),
		),
	); err == nil {
		apiConf.SetPresignSecret(key)
	}
	deploymentsHandlers := NewDeploymentsApiHandlers(
		mongoStorage, new(view.RESTView), app, apiConf,
	)

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
		rest.Post(ApiUrlManagementDeploymentsGroup, controller.DeployToGroup),
		rest.Get(ApiUrlManagementDeployments, controller.LookupDeployment),
		rest.Get(ApiUrlManagementDeploymentsId, controller.GetDeployment),
		rest.Get(ApiUrlManagementDeploymentsStatistics, controller.GetDeploymentStats),
		rest.Put(ApiUrlManagementDeploymentsStatus, controller.AbortDeployment),
		rest.Get(ApiUrlManagementDeploymentsDevices,
			controller.GetDeviceStatusesForDeployment),
		rest.Get(ApiUrlManagementDeploymentsDevicesList,
			controller.GetDevicesListForDeployment),
		rest.Get(ApiUrlManagementDeploymentsLog,
			controller.GetDeploymentLogForDevice),
		rest.Get(ApiUrlManagementDeploymentsDeviceList,
			controller.GetDeploymentDeviceList),

		// Configuration deployments (internal)
		rest.Post(ApiUrlInternalDeviceConfigurationDeployments,
			controller.PostDeviceConfigurationDeployment),

		// Devices
		rest.Get(ApiUrlDevicesDeploymentsNext, controller.GetDeploymentForDevice),
		rest.Post(ApiUrlDevicesDeploymentsNext, controller.GetDeploymentForDevice),
		rest.Put(ApiUrlDevicesDeploymentStatus,
			controller.PutDeploymentStatusForDevice),
		rest.Put(ApiUrlDevicesDeploymentsLog,
			controller.PutDeploymentLogForDevice),
		rest.Get(ApiUrlDevicesDownloadConfig,
			controller.DownloadConfiguration),

		// Health Check
		rest.Get(ApiUrlInternalAlive, controller.AliveHandler),
		rest.Get(ApiUrlInternalHealth, controller.HealthHandler),
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
		rest.Delete(ApiUrlInternalTenantDeploymentsDevice, controller.DecommissionDevice),
		rest.Post(ApiUrlInternalTenantArtifacts, controller.NewImageForTenantHandler),

		// per-tenant storage settings
		rest.Get(ApiUrlInternalTenantStorageSettings, controller.GetTenantStorageSettingsHandler),
		rest.Put(ApiUrlInternalTenantStorageSettings, controller.PutTenantStorageSettingsHandler),
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

func FMTConfigURL(scheme, hostname, deploymentID, deviceType, deviceID string) string {
	repl := strings.NewReplacer(
		"#"+ParamDeploymentID, url.PathEscape(deploymentID),
		"#"+ParamDeviceType, url.PathEscape(deviceType),
		"#"+ParamDeviceID, url.PathEscape(deviceID),
	)
	return scheme + "://" + hostname + repl.Replace(ApiUrlDevicesDownloadConfig)
}
