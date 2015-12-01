package main

import (
	"github.com/mendersoftware/services/Godeps/_workspace/src/github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/services/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/mendersoftware/services/controllers"
	"github.com/mendersoftware/services/handlers"
	"github.com/mendersoftware/services/models/fileservice/s3"
	"github.com/mendersoftware/services/models/images/memmap"
	"github.com/mendersoftware/services/utils/safemap"
)

// NewRouter defines all REST API routes.
func NewRouter(c *cli.Context) (rest.App, error) {

	images := memmap.NewImagesInMem(safemap.NewStringMap())

	bucket := c.String(S3BucketFlag)
	awsKey := c.String(AwsAccessKeyIdFlag)
	awsSecret := c.String(AwsAccessKeySecretFlag)
	region := c.String(AwsS3RegionFlag)

	fileStorage := s3.NewSimpleStorageService(bucket, awsKey, awsSecret, region, "")

	meta := handlers.NewImageMeta(controllers.NewImagesController(images, fileStorage))

	app, err := rest.MakeRouter(
		rest.Get("/api/0.0.1/images/", meta.Lookup),
		rest.Post("/api/0.0.1/images/", meta.Create),

		rest.Get("/api/0.0.1/images/:id", meta.Get),
		rest.Put("/api/0.0.1/images/:id", meta.Edit),
		rest.Delete("/api/0.0.1/images/:id", meta.Delete),

		rest.Get("/api/0.0.1/images/:id/upload", meta.UploadLink),
		rest.Get("/api/0.0.1/images/:id/download", meta.DownloadLink),

		// rest.Post("/api/0.0.1/images/:id/verify", verifier.CreateJob),
		// rest.Get("/api/0.0.1/images/:id/verify/:job", verifier.GetJob),
		// rest.Delete("/api/0.0.1/images/:id/verify/:job", verifier.DeleteJob),
	)

	return app, err
}
