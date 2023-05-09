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

package http

import (
	"context"
	"net/url"
	"strings"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/go-lib-micro/log"

	"github.com/mendersoftware/deployments/app"
	"github.com/mendersoftware/deployments/store"
	"github.com/mendersoftware/deployments/utils/restutil"
	"github.com/mendersoftware/deployments/utils/restutil/view"
)

const (
	ApiUrlInternal   = "/api/internal/v1/deployments"
	ApiUrlManagement = "/api/management/v1/deployments"
	ApiUrlDevices    = "/api/devices/v1/deployments"

	ApiUrlManagementArtifacts               = ApiUrlManagement + "/artifacts"
	ApiUrlManagementArtifactsList           = ApiUrlManagement + "/artifacts/list"
	ApiUrlManagementArtifactsGenerate       = ApiUrlManagement + "/artifacts/generate"
	ApiUrlManagementArtifactsDirectUpload   = ApiUrlManagement + "/artifacts/directupload"
	ApiUrlManagementArtifactsCompleteUpload = ApiUrlManagementArtifactsDirectUpload +
		"/#id/complete"
	ApiUrlManagementArtifactsId         = ApiUrlManagement + "/artifacts/#id"
	ApiUrlManagementArtifactsIdDownload = ApiUrlManagement + "/artifacts/#id/download"

	ApiUrlManagementDeployments                   = ApiUrlManagement + "/deployments"
	ApiUrlManagementMultipleDeploymentsStatistics = ApiUrlManagement +
		"/deployments/statistics/list"
	ApiUrlManagementDeploymentsGroup       = ApiUrlManagement + "/deployments/group/#name"
	ApiUrlManagementDeploymentsId          = ApiUrlManagement + "/deployments/#id"
	ApiUrlManagementDeploymentsStatistics  = ApiUrlManagement + "/deployments/#id/statistics"
	ApiUrlManagementDeploymentsStatus      = ApiUrlManagement + "/deployments/#id/status"
	ApiUrlManagementDeploymentsDevices     = ApiUrlManagement + "/deployments/#id/devices"
	ApiUrlManagementDeploymentsDevicesList = ApiUrlManagement + "/deployments/#id/devices/list"
	ApiUrlManagementDeploymentsLog         = ApiUrlManagement +
		"/deployments/#id/devices/#devid/log"
	ApiUrlManagementDeploymentsDeviceId      = ApiUrlManagement + "/deployments/devices/#id"
	ApiUrlManagementDeploymentsDeviceHistory = ApiUrlManagement + "/deployments/devices/#id/history"
	ApiUrlManagementDeploymentsDeviceList    = ApiUrlManagement + "/deployments/#id/device_list"

	ApiUrlManagementReleasesList = ApiUrlManagement + "/deployments/releases/list"

	ApiUrlManagementLimitsName = ApiUrlManagement + "/limits/#name"

	ApiUrlDevicesDeploymentsNext  = ApiUrlDevices + "/device/deployments/next"
	ApiUrlDevicesDeploymentStatus = ApiUrlDevices + "/device/deployments/#id/status"
	ApiUrlDevicesDeploymentsLog   = ApiUrlDevices + "/device/deployments/#id/log"
	ApiUrlDevicesDownloadConfig   = ApiUrlDevices +
		"/download/configuration/#deployment_id/#device_type/#device_id"

	ApiUrlInternalAlive                    = ApiUrlInternal + "/alive"
	ApiUrlInternalHealth                   = ApiUrlInternal + "/health"
	ApiUrlInternalTenants                  = ApiUrlInternal + "/tenants"
	ApiUrlInternalTenantDeployments        = ApiUrlInternal + "/tenants/#tenant/deployments"
	ApiUrlInternalTenantDeploymentsDevices = ApiUrlInternal + "/tenants/#tenant/deployments/devices"
	ApiUrlInternalTenantDeploymentsDevice  = ApiUrlInternal +
		"/tenants/#tenant/deployments/devices/#id"
	ApiUrlInternalTenantArtifacts       = ApiUrlInternal + "/tenants/#tenant/artifacts"
	ApiUrlInternalTenantStorageSettings = ApiUrlInternal +
		"/tenants/#tenant/storage/settings"
	ApiUrlInternalDeviceConfigurationDeployments = ApiUrlInternal +
		"/tenants/#tenant/configuration/deployments/#deployment_id/devices/#device_id"
	ApiUrlInternalDeviceDeploymentLastStatusDeployments = ApiUrlInternal +
		"/tenants/#tenant/devices/deployments/last"
)

// NewRouter defines all REST API routes.
func NewRouter(
	ctx context.Context,
	app app.App,
	ds store.DataStore,
	cfg *Config,
) (rest.App, error) {

	// Create and configure API handlers
	//
	// Encode base64 secret in either std or URL encoding ignoring padding.
	deploymentsHandlers := NewDeploymentsApiHandlers(
		ds, new(view.RESTView), app, cfg,
	)

	// Routing
	imageRoutes := NewImagesResourceRoutes(deploymentsHandlers, cfg)
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

func NewImagesResourceRoutes(controller *DeploymentsApiHandlers, cfg *Config) []*rest.Route {

	if controller == nil {
		return []*rest.Route{}
	}

	routes := []*rest.Route{
		rest.Post(ApiUrlManagementArtifacts, controller.NewImage),
		rest.Post(ApiUrlManagementArtifactsGenerate, controller.GenerateImage),
		rest.Get(ApiUrlManagementArtifacts, controller.GetImages),
		rest.Get(ApiUrlManagementArtifactsList, controller.ListImages),

		rest.Get(ApiUrlManagementArtifactsId, controller.GetImage),
		rest.Delete(ApiUrlManagementArtifactsId, controller.DeleteImage),
		rest.Put(ApiUrlManagementArtifactsId, controller.EditImage),

		rest.Get(ApiUrlManagementArtifactsIdDownload, controller.DownloadLink),
	}
	if cfg.EnableDirectUpload {
		log.NewEmpty().Infof(
			"direct upload enabled: POST %s",
			ApiUrlManagementArtifactsDirectUpload,
		)
		routes = append(routes, rest.Post(
			ApiUrlManagementArtifactsDirectUpload,
			controller.UploadLink,
		))
		routes = append(routes, rest.Post(
			ApiUrlManagementArtifactsCompleteUpload,
			controller.CompleteUpload,
		))
	}
	return routes
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
		rest.Post(ApiUrlManagementMultipleDeploymentsStatistics,
			controller.GetDeploymentsStats),
		rest.Get(ApiUrlManagementDeploymentsStatistics, controller.GetDeploymentStats),
		rest.Put(ApiUrlManagementDeploymentsStatus, controller.AbortDeployment),
		rest.Get(ApiUrlManagementDeploymentsDevices,
			controller.GetDeviceStatusesForDeployment),
		rest.Get(ApiUrlManagementDeploymentsDevicesList,
			controller.GetDevicesListForDeployment),
		rest.Get(ApiUrlManagementDeploymentsLog,
			controller.GetDeploymentLogForDevice),
		rest.Delete(ApiUrlManagementDeploymentsDeviceId,
			controller.AbortDeviceDeployments),
		rest.Delete(ApiUrlManagementDeploymentsDeviceHistory,
			controller.DeleteDeviceDeploymentsHistory),
		rest.Get(ApiUrlManagementDeploymentsDeviceId,
			controller.ListDeviceDeployments),
		rest.Get(ApiUrlManagementDeploymentsDeviceList,
			controller.GetDeploymentDeviceList),

		// Configuration deployments (internal)
		rest.Post(ApiUrlInternalDeviceConfigurationDeployments,
			controller.PostDeviceConfigurationDeployment),

		// Last device deployment status deployments (internal)
		rest.Post(ApiUrlInternalDeviceDeploymentLastStatusDeployments,
			controller.GetDeviceDeploymentLastStatus),

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
		rest.Get(ApiUrlInternalTenantDeploymentsDevices,
			controller.ListDeviceDeploymentsByIDsInternal),
		rest.Get(ApiUrlInternalTenantDeploymentsDevice,
			controller.ListDeviceDeploymentsInternal),
		rest.Delete(ApiUrlInternalTenantDeploymentsDevice,
			controller.AbortDeviceDeploymentsInternal),
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
		rest.Get(ApiUrlManagementReleasesList, controller.ListReleases),
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
