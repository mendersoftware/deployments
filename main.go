// Copyright 2022 Northern.tech AS
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
	"time"

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

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name: "config",
			Usage: "Configuration `FILE`." +
				" Supports JSON, TOML, YAML and HCL formatted configs.",
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
		if err := dconfig.Setup(configPath); err != nil {
			return cli.NewExitError(err.Error(), 1)
		}

		return nil
	}

	err := app.Run(args)
	if err != nil {
		log.NewEmpty().Fatal(err.Error())
	}
}

func cmdServer(args *cli.Context) error {
	devSetup := args.GlobalBool("dev")

	l := log.New(log.Ctx{})

	if devSetup {
		l.Infof("setting up development configuration")
		config.Config.Set(dconfig.SettingMiddleware, dconfig.EnvDev)
	}

	l.Print("Deployments Service starting up")
	err := migrate("", args.Bool("automigrate"))
	if err != nil {
		return err
	}

	setupContext, cancel := context.WithTimeout(
		context.Background(),
		time.Second*30,
	)
	err = RunServer(setupContext)
	cancel()
	if err != nil {
		return cli.NewExitError(err.Error(), 4)
	}

	return nil
}

func cmdMigrate(args *cli.Context) error {
	tenant := args.String("tenant")
	return migrate(tenant, true)
}

func migrate(tenant string, automigrate bool) error {
	ctx := context.Background()

	dbClient, err := mongo.NewMongoClient(ctx, config.Config)
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("failed to connect to db: %v", err),
			3)
	}
	defer func() {
		_ = dbClient.Disconnect(ctx)
	}()

	if tenant != "" {
		db := mstore.DbNameForTenant(tenant, mongo.DbName)
		err = mongo.MigrateSingle(ctx, db, mongo.DbVersion, dbClient, true)
	} else {
		err = mongo.Migrate(ctx, mongo.DbVersion, dbClient, true)
	}
	if err != nil {
		return cli.NewExitError(
			fmt.Sprintf("failed to run migrations: %v", err),
			3)
	}

	return nil
}
