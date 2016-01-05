package main

import (
	"log"
	"os"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/artifacts/config"
	"github.com/mendersoftware/artifacts/handlers"
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

func SetupMiddleware(c config.ConfigReader, api *rest.Api) {

	api.Use(DefaultDevStack...)

	api.Use(&rest.CorsMiddleware{
		RejectNonCorsRequests: false,

		// Should be tested with some list
		OriginValidator: func(origin string, request *rest.Request) bool {
			// Accept all requests
			return true
		},

		// Preflight request cache lenght
		AccessControlMaxAge: 60,

		// Allow authentication requests
		AccessControlAllowCredentials: true,

		// Allowed headers
		AllowedMethods: []string{
			handlers.HttpMethodGet,
			handlers.HttpMethodPost,
			handlers.HttpMethodPut,
			handlers.HttpMethodDelete,
			handlers.HttpMethodOptions,
		},

		// Allowed heardes
		AllowedHeaders: []string{
			handlers.HttpHeaderAllow,
			handlers.HttpHeaderContentType,
			handlers.HttpHeaderOrigin,
			handlers.HttpHeaderAuthorization,
			handlers.HttpHeaderAcceptEncoding,
			handlers.HttpHeaderAccessControlRequestHeaders,
			handlers.HttpHeaderAccessControlRequestMethod,
		},
	})
}
