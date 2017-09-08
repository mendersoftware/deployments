// Copyright 2017 Northern.tech AS
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
package migrations

import (
	"context"

	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	ctx_store "github.com/mendersoftware/go-lib-micro/store"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
)

const (
	DbVersion = "1.2.1"
	DbName    = "deployment_service"
)

func Migrate(ctx context.Context,
	version string,
	session *mgo.Session,
	automigrate bool) error {

	l := log.FromContext(ctx)

	dbs, err := migrate.GetTenantDbs(session, ctx_store.IsTenantDb(DbName))
	if err != nil {
		return errors.Wrap(err, "failed go retrieve tenant DBs")
	}

	if len(dbs) == 0 {
		dbs = []string{DbName}
	}

	if automigrate {
		l.Infof("automigrate is ON, will apply migrations")
	} else {
		l.Infof("automigrate is OFF, will check db version compatibility")
	}

	for _, d := range dbs {
		l.Infof("migrating %s", d)
		m := migrate.DummyMigrator{
			Session:     session,
			Db:          d,
			Automigrate: automigrate,
		}

		ver, err := migrate.NewVersion(version)
		if err != nil {
			return errors.Wrap(err, "failed to parse service version")
		}

		err = m.Apply(ctx, *ver, nil)
		if err != nil {
			return errors.Wrap(err, "failed to apply migrations")
		}
	}

	return nil
}
