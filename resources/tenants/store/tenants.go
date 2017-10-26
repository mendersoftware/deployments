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
package store

import (
	"context"

	"gopkg.in/mgo.v2"

	"github.com/mendersoftware/deployments/migrations"
	mstore "github.com/mendersoftware/go-lib-micro/store"
)

type Store interface {
	ProvisionTenant(ctx context.Context, tenantId string) error
}

type store struct {
	session *mgo.Session
}

func NewStore(session *mgo.Session) *store {
	return &store{
		session: session,
	}
}

func (ts *store) ProvisionTenant(ctx context.Context, tenantId string) error {
	session := ts.session.Copy()
	defer session.Close()

	dbname := mstore.DbNameForTenant(tenantId, migrations.DbName)

	return migrations.MigrateSingle(ctx, dbname, migrations.DbVersion, session, true)
}
