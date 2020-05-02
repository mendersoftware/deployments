// Copyright 2020 Northern.tech AS
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

package main

import (
	"context"
	"net/http"

	"github.com/ant0ine/go-json-rest/rest"

	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/mendersoftware/go-lib-micro/log"

	api_http "github.com/mendersoftware/deployments/api/http"
	dconfig "github.com/mendersoftware/deployments/config"
	mstore "github.com/mendersoftware/deployments/store/mongo"
)

func RunServer(c config.Reader) error {

	ctx := context.Background()
	dbClient, err := mstore.NewMongoClient(ctx, c)
	if err != nil {
		return err
	}
	defer dbClient.Disconnect(ctx)

	router, err := api_http.NewRouter(ctx, c, dbClient)
	if err != nil {
		return err
	}

	api := rest.NewApi()
	SetupMiddleware(c, api)
	api.SetApp(router)

	listen := c.GetString(dconfig.SettingListen)

	if c.IsSet(dconfig.SettingHttps) {

		cert := c.GetString(dconfig.SettingHttpsCertificate)
		key := c.GetString(dconfig.SettingHttpsKey)

		return http.ListenAndServeTLS(listen, cert, key, api.MakeHandler())
	}

	l := log.FromContext(ctx)
	l.Printf("listening on %s", listen)
	return http.ListenAndServe(listen, api.MakeHandler())
}
