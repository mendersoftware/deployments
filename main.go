// Copyright 2019 Northern.tech AS
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
	"fmt"
	"os"
	"strings"

	"github.com/mendersoftware/go-lib-micro/config"
	"github.com/mendersoftware/go-lib-micro/log"
	mstore "github.com/mendersoftware/go-lib-micro/store"
	"github.com/urfave/cli"

	dconfig "github.com/mendersoftware/deployments/config"
	"github.com/mendersoftware/deployments/store/mongo"
)

func main() {
	doMain(os.Args)
}

func doMain(args []string) {

	var configPath string

	app := cli.NewApp()
	app.Usage = "Deployments Service"
	app.Version = CreateVersionString()

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "config",
			Usage:       "Configuration `FILE`. Supports JSON, TOML, YAML and HCL formatted configs.",
			Destination: &configPath,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "server",
			Usage: "Run the service as a server",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "automigrate",
					Usage: "Run database migrations before starting.",
				},
			},

			Action: cmdServer,
		},
		{
			Name:  "migrate",
			Usage: "Run migrations and exit",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "tenant",
					Usage: "Tenant ID (optional).",
				},
			},

			Action: cmdMigrate,
		},
	}

	app.Action = cmdServer
	app.Before = func(args *cli.Context) error {

		err := config.FromConfigFile(configPath, dconfig.Defaults)
		if err != nil {
			return cli.NewExitError(
				fmt.Sprintf("error loading configuration: %s", err),
				1)
		}

		// Enable setting config values by environment variables
		config.Config.SetEnvPrefix("DEPLOYMENTS")
		config.Config.AutomaticEnv()
		config.Config.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

		return nil
	}

	app.Run(args)
}

func cmdServer(args *cli.Context) error {
	devSetup := args.GlobalBool("dev")

	l := log.New(log.Ctx{})

	if devSetup {
		l.Infof("setting up development configuration")
		config.Config.Set(dconfig.SettingMiddleware, dconfig.EnvDev)
	}

	l.Printf("Deployments Service, version %s starting up",
		CreateVersionString())

	dbSession, err := mongo.NewMongoSession(config.Config)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("failed to connect to db: %v", err),
			3)
	}

	if args.Bool("automigrate") {
		err = mongo.Migrate(context.Background(), mongo.DbVersion, dbSession, true)
		if err != nil {
			return cli.NewExitError(
				fmt.Sprintf("failed to run migrations: %v", err),
				3)
		}
	}
	dbSession.Close()

	err = RunServer(config.Config)
	if err != nil {
		return cli.NewExitError(err.Error(), 4)
	}

	return nil
}

func cmdMigrate(args *cli.Context) error {
	tenant := args.String("tenant")
	db := mstore.DbNameForTenant(tenant, mongo.DbName)

	dbSession, err := mongo.NewMongoSession(config.Config)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("failed to connect to db: %v", err),
			3)
	}

	err = mongo.MigrateSingle(context.Background(), db, mongo.DbVersion, dbSession, true)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("failed to run migrations: %v", err),
			3)
	}

	dbSession.Close()

	return nil
}
