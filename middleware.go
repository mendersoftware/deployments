package main

import (
	"log"
	"os"

	"github.com/mendersoftware/services/Godeps/_workspace/src/github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/services/Godeps/_workspace/src/github.com/codegangsta/cli"
)

const (
	EnvProd = "prod"
	EnvDev  = "dev"
)

var DefaultDevStack = []rest.Middleware{

	// logging
	&rest.AccessLogApacheMiddleware{},
	&rest.TimerMiddleware{},
	&rest.RecorderMiddleware{},

	// catches the panic errors that occur with stack trace
	&rest.RecoverMiddleware{
		EnableResponseStackTrace: true,
	},

	// json pretty print
	&rest.JsonIndentMiddleware{},

	// verifies the request Content-Type header
	// The expected Content-Type is 'application/json'
	// if the content is non-null
	&rest.ContentTypeCheckerMiddleware{},
}

var DefaultProdStack = []rest.Middleware{

	// logging
	&rest.AccessLogJsonMiddleware{
		// No prefix or other fields, entire output is JSON encoded and could break it.
		Logger: log.New(os.Stdout, "", 0),
	},
	&rest.TimerMiddleware{},
	&rest.RecorderMiddleware{},

	// catches the panic errorsx
	&rest.RecoverMiddleware{},

	// response compression
	&rest.GzipMiddleware{},

	// verifies the request Content-Type header
	// The expected Content-Type is 'application/json'
	// if the content is non-null
	&rest.ContentTypeCheckerMiddleware{},
}

func InstallMiddleware(c *cli.Context, api *rest.Api) {

	env := c.String(EnvFlag)

	switch env {
	case EnvProd:
		api.Use(DefaultProdStack...)
	case EnvDev:
		api.Use(DefaultDevStack...)
	default:
		log.Fatal(InvalidValueError(EnvFlag, env))
	}

	api.Use(&rest.AuthBasicMiddleware{
		Realm: "artifacts",
		Authenticator: func(userId string, password string) bool {
			if userId == password {
				return true
			}
			return false
		},
	})
}
