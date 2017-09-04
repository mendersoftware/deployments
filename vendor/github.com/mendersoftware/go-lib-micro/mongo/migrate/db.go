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
package migrate

import (
	"time"

	"github.com/mendersoftware/go-lib-micro/store"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
)

// this is a small internal data layer for the migration utils, may be shared by diff migrators
const (
	DbMigrationsColl = "migration_info"
)

type MigrationEntry struct {
	Version   Version   `bson:"version"`
	Timestamp time.Time `bson:"timestamp"`
}

// GetMigrationInfo retrieves a list of migrations applied to the db.
func GetMigrationInfo(sess *mgo.Session, db string) ([]MigrationEntry, error) {
	s := sess.Copy()
	defer s.Close()
	c := s.DB(db).C(DbMigrationsColl)

	var info []MigrationEntry

	var err = c.Find(nil).All(&info)
	if err != nil {
		return nil, errors.Wrap(err, "db: failed to get migration info")
	}

	return info, nil
}

// UpdateMigrationInfo inserts a migration entry in the migration info collection.
func UpdateMigrationInfo(version Version, sess *mgo.Session, db string) error {
	s := sess.Copy()
	defer s.Close()
	c := s.DB(db).C(DbMigrationsColl)

	entry := MigrationEntry{
		Version:   version,
		Timestamp: time.Now(),
	}

	err := c.Insert(entry)
	if err != nil {
		return errors.Wrap(err, "db: failed to insert migration info")
	}

	return nil
}

func GetTenantDbs(sess *mgo.Session, matcher store.TenantDbMatchFunc) ([]string, error) {
	s := sess.Copy()
	defer s.Close()

	dbs, err := s.DatabaseNames()
	if err != nil {
		return nil, err
	}

	tenantDbs := []string{}
	for _, db := range dbs {
		if matcher(db) {
			tenantDbs = append(tenantDbs, db)
		}
	}

	return tenantDbs, err
}
