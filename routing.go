package main

import (
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/artifacts/config"
	"github.com/mendersoftware/artifacts/controllers"
	"github.com/mendersoftware/artifacts/handlers"
	"github.com/mendersoftware/artifacts/models/fileservice"
	"github.com/mendersoftware/artifacts/models/fileservice/s3"
	"github.com/mendersoftware/artifacts/models/images/memmap"
	"github.com/mendersoftware/artifacts/utils/safemap"
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

	images := memmap.NewImagesInMem(safemap.NewStringMap())
	meta := handlers.NewImageMeta(controllers.NewImagesController(images, SetupFileStorage(c)))

	// Define routers and autogen OPTIONS method for each route.
	routes := []*rest.Route{
		rest.Get("/api/0.0.1/images", meta.Lookup),
		rest.Post("/api/0.0.1/images", meta.Create),
		// rest.Options("/api/0.0.1/images", handlers.NewOptionsHandler(handlers.HttpMethodGet,
		// 	handlers.HttpMethodPost)),

		rest.Get("/api/0.0.1/images/:id", meta.Get),
		rest.Put("/api/0.0.1/images/:id", meta.Edit),
		rest.Delete("/api/0.0.1/images/:id", meta.Delete),
		// rest.Options("/api/0.0.1/images/:id", handlers.NewOptionsHandler(handlers.HttpMethodGet,
		// 	handlers.HttpMethodPut, handlers.HttpMethodDelete)),

		rest.Get("/api/0.0.1/images/:id/upload", meta.UploadLink),
		// rest.Options("/api/0.0.1/images/:id/upload", handlers.NewOptionsHandler(handlers.HttpMethodGet)),

		rest.Get("/api/0.0.1/images/:id/download", meta.DownloadLink),
		// rest.Options("/api/0.0.1/images/:id/download", handlers.NewOptionsHandler(handlers.HttpMethodGet)),
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
