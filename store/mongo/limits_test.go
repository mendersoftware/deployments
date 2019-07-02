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

package mongo

import (
	"context"
	"testing"

	"github.com/mendersoftware/go-lib-micro/identity"
	ctxstore "github.com/mendersoftware/go-lib-micro/store"
	"github.com/stretchr/testify/assert"

	"github.com/mendersoftware/deployments/model"
)

// db and test management funcs
func getDb(ctx context.Context) *DataStoreMongo {
	db.Wipe()

	ds := NewDataStoreMongoWithSession(db.Session())

	return ds
}

func TestGetLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TestGetLimit in short mode.")
	}

	lim1 := model.Limit{
		Name:  "foo",
		Value: 123,
	}
	lim2 := model.Limit{
		Name:  "bar",
		Value: 456,
	}
	lim3OtherTenant := model.Limit{
		Name:  "bar",
		Value: 920,
	}

	tenant := "foo"

	dbCtx := identity.WithContext(context.Background(), &identity.Identity{
		Tenant: tenant,
	})
	db := getDb(dbCtx)
	defer db.session.Close()
	s := db.session.Copy()
	defer s.Close()

	coll := s.DB(ctxstore.DbFromContext(dbCtx, DatabaseName)).C(CollectionLimits)
	assert.NoError(t, coll.Insert(lim1, lim2))

	dbCtxOtherTenant := identity.WithContext(context.Background(), &identity.Identity{
		Tenant: "other-" + tenant,
	})
	collOtherTenant := s.DB(ctxstore.DbFromContext(dbCtxOtherTenant,
		DatabaseName)).C(CollectionLimits)
	assert.NoError(t, collOtherTenant.Insert(lim3OtherTenant))

	// check if value is fetched correctly
	lim, err := db.GetLimit(dbCtx, "foo")
	assert.NoError(t, err)
	assert.EqualValues(t, lim1, *lim)

	// try with something that does not exist
	lim, err = db.GetLimit(dbCtx, "nonexistent-foo")
	assert.EqualError(t, err, ErrLimitNotFound.Error())
	assert.Nil(t, lim)

	// switch tenants
	lim, err = db.GetLimit(dbCtxOtherTenant, "foo")
	assert.EqualError(t, err, ErrLimitNotFound.Error())

	lim, err = db.GetLimit(dbCtxOtherTenant, "bar")
	assert.NoError(t, err)
	assert.EqualValues(t, lim3OtherTenant, *lim)
}
