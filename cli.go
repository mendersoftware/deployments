package main

import (
	"errors"
	"fmt"

	"github.com/mendersoftware/artifacts/Godeps/_workspace/src/github.com/codegangsta/cli"
)

const (
	EC2Flag        = "ec2"
	EC2Description = "Executing in EC2. Automatically load IAM credentials."
	EC2Var         = "MENDER_EC2"

	TLSCertificateFlag        = "certificate"
	TLSCertificateDescription = "HTTPS certificate filename."
	TLSCertificateVar         = "MENDER_ARTIFACT_CERT"

	TLSKeyFlag        = "key"
	TLSKeyDescription = "HTTPS private key filename."
	TLSKeyVar         = "MENDER_ARTIFACT_CERT_KEY"

	HTTPSFlag        = "https"
	HTTPSDescription = "Serve under HTTPS. Requires key and cerfiticate."
	HTTPSVar         = "MENDER_ARTIFACT_HTTPS"

	ListenFlag        = "listen"
	ListenDescription = "TCP network address."
	ListenDefault     = "localhost:8080"
	ListenVar         = "MENDER_ARTIFACT_LISTEN"

	EnvFlag        = "env"
	EnvDescription = "Environment " + EnvProd + "|" + EnvDev
	EnvDefault     = EnvDev
	EnvVar         = "MENDER_ARTIFACT_ENV"

	AwsAccessKeyIdFlag        = "aws-id"
	AwsAccessKeyIdDescription = "AWS access id key with S3 read/write permissions for specified bucket (required if now ec2)."
	AwsAccessKeyIdVar         = "AWS_ACCESS_KEY_ID"

	AwsAccessKeySecretFlag        = "aws_secret"
	AwsAccessKeySecretDescription = "AWS secret key with S3 read/write permissions for specified bucket (required if not ec2)."
	AwsAccessKeySecretVar         = "AWS_SECRET_ACCESS_KEY"

	AwsS3RegionFlag        = "aws-region"
	AwsS3RegionDescription = "AWS region."
	AwsS3RegionDefault     = "eu-west-1"
	AwsS3RegionVar         = "AWS_REGION"

	S3BucketFlag        = "bucket"
	S3BucketDescription = "S3 bucket name for image storage."
	S3BucketDefault     = "mender-artifact-storage"
	S3BucketVar         = "MENDER_S3_BUCKET"
)

func SetupGlobalFlags(app *cli.App) {

	app.Name = "artifacts"
	app.Usage = "Archifact management service for mender.io"
	app.Authors = []cli.Author{
		{"Maciej Mrowiec", "maciej.mrowiec@mender.io"},
	}
	app.Email = "contact@mender.io"

	if Tag != "" {
		app.Version = Tag
	} else {
		app.Version = BuildNumber
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   TLSCertificateFlag,
			Usage:  TLSCertificateDescription,
			EnvVar: TLSCertificateVar,
		},
		cli.StringFlag{
			Name:   TLSKeyFlag,
			Usage:  TLSKeyDescription,
			EnvVar: TLSKeyVar,
		},
		cli.BoolFlag{
			Name:   HTTPSFlag,
			Usage:  HTTPSDescription,
			EnvVar: HTTPSVar,
		},
		cli.StringFlag{
			Name:   ListenFlag,
			Usage:  ListenDescription,
			Value:  ListenDefault,
			EnvVar: ListenVar,
		},
		cli.StringFlag{
			Name:   EnvFlag,
			Usage:  EnvDescription,
			Value:  EnvDefault,
			EnvVar: EnvVar,
		},

		cli.StringFlag{
			Name:   AwsAccessKeyIdFlag,
			Usage:  AwsAccessKeyIdDescription,
			EnvVar: AwsAccessKeyIdVar,
		},
		cli.StringFlag{
			Name:   AwsAccessKeySecretFlag,
			Usage:  AwsAccessKeySecretDescription,
			EnvVar: AwsAccessKeySecretVar,
		},
		cli.StringFlag{
			Name:   S3BucketFlag,
			Usage:  S3BucketDescription,
			Value:  S3BucketDefault,
			EnvVar: S3BucketVar,
		},
		cli.StringFlag{
			Name:   AwsS3RegionFlag,
			Usage:  AwsS3RegionDescription,
			Value:  AwsS3RegionDefault,
			EnvVar: AwsS3RegionVar,
		},
		cli.BoolFlag{
			Name:   EC2Flag,
			Usage:  EC2Description,
			EnvVar: EC2Var,
		},
	}
}

func ValidateGlobalFlags(c *cli.Context) error {

	fns := []func(c *cli.Context) error{
		validateAWSFlags,
		validateEnvFlags,
		validateHttpFlags,
	}

	for _, validateFn := range fns {
		err := validateFn(c)
		if err != nil {
			cli.ShowAppHelp(c)
			return err
		}
	}

	return nil
}

func validateAWSFlags(c *cli.Context) error {

	key := c.String(AwsAccessKeyIdFlag)
	secret := c.String(AwsAccessKeySecretFlag)
	region := c.String(AwsS3RegionFlag)
	ec2 := c.Bool(EC2Flag)

	if !ec2 {
		if key == "" {
			return MissingOptionError(AwsAccessKeyIdFlag)
		}

		if secret == "" {
			return MissingOptionError(AwsAccessKeySecretFlag)
		}
	}

	if region == "" {
		return MissingOptionError(AwsS3RegionFlag)
	}

	return nil
}

func validateEnvFlags(c *cli.Context) error {

	env := c.String(EnvFlag)

	switch env {
	case EnvProd:
		return nil
	case EnvDev:
		return nil

	default:
		return InvalidValueError(EnvFlag, env)
	}
}

func validateHttpFlags(c *cli.Context) error {

	isHttps := c.Bool(HTTPSFlag)
	cert := c.String(TLSCertificateFlag)
	key := c.String(TLSKeyFlag)

	if isHttps {
		if cert == "" {
			return MissingOptionError(TLSCertificateFlag)
		}

		if key == "" {
			return MissingOptionError(TLSKeyFlag)
		}
	}

	return nil
}

func InvalidValueError(option string, value interface{}) error {
	return errors.New(fmt.Sprintf("Invalid value '%s' = '%v'.", option, value))
}

func MissingOptionError(option string) error {
	return errors.New(fmt.Sprintf("Required option: '%s'", option))
}
