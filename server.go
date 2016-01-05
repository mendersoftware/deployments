package main

import (
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/mendersoftware/artifacts/config"
)

func RunServer(c config.ConfigReader) error {
	router, err := NewRouter(c)
	if err != nil {
		return err
	}

	api := rest.NewApi()
	SetupMiddleware(c, api)
	api.SetApp(router)

	listen := c.GetString(SettingListen)

	if c.IsSet(SettingHttps) {

		cert := c.GetString(SettingHttpsCertificate)
		key := c.GetString(SettingHttpsKey)

		return http.ListenAndServeTLS(listen, cert, key, api.MakeHandler())
	}

	return http.ListenAndServe(listen, api.MakeHandler())
}
