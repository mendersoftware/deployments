package main

import (
	"log"
	"net/http"

	"github.com/mendersoftware/services/Godeps/_workspace/src/github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/services/Godeps/_workspace/src/github.com/codegangsta/cli"
)

func main() {

	app := cli.NewApp()

	// Add handling for global flags
	SetupGlobalFlags(app)

	// Entry point for application
	app.Before = ValidateGlobalFlags
	app.Action = StartServer

	app.RunAndExitOnError()
}

func StartServer(c *cli.Context) {

	api := rest.NewApi()

	// setup middleware - layers of common pre/post processing of the requests
	InstallMiddleware(c, api)

	router, err := NewRouter(c)
	if err != nil {
		log.Fatal(err)
	}

	api.SetApp(router)

	log.Fatal(Listen(c, api.MakeHandler()))
}

// Start HTTP/HTTPS server depending on global settings.
func Listen(c *cli.Context, handler http.Handler) error {

	listen := c.String(ListenFlag)
	isHttps := c.Bool(HTTPSFlag)

	if isHttps {
		cert := c.String(TLSCertificateFlag)
		key := c.String(TLSKeyFlag)

		return http.ListenAndServeTLS(listen, cert, key, handler)
	}

	return http.ListenAndServe(listen, handler)
}
