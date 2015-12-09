package main

import (
	"github.com/mendersoftware/artifacts/Godeps/_workspace/src/github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/artifacts/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/mendersoftware/artifacts/controllers"
	"github.com/mendersoftware/artifacts/handlers"
	"github.com/mendersoftware/artifacts/models/fileservice/s3"
	"github.com/mendersoftware/artifacts/models/images/memmap"
	"github.com/mendersoftware/artifacts/utils/safemap"
)

// NewRouter defines all REST API routes.
func NewRouter(c *cli.Context) (rest.App, error) {

	images := memmap.NewImagesInMem(safemap.NewStringMap())

	bucket := c.String(S3BucketFlag)
	awsKey := c.String(AwsAccessKeyIdFlag)
	awsSecret := c.String(AwsAccessKeySecretFlag)
	region := c.String(AwsS3RegionFlag)
	ec2 := c.Bool(EC2Flag)

	var fileStorage *s3.SimpleStorageService
	if ec2 {
		fileStorage = s3.NewSimpleStorageServiceDefaults(bucket, region)
	} else {
		fileStorage = s3.NewSimpleStorageServiceStatic(bucket, awsKey, awsSecret, region, "")
	}

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
