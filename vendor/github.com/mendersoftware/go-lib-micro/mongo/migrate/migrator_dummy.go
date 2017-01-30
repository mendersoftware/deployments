// Copyright 2016 Mender Software AS
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
	"gopkg.in/mgo.v2"
)

// MigratorDummy does not actually apply migrations, just inserts the
// target version into the db to mark the initial/current state.
type DummyMigrator struct {
	Session *mgo.Session
	Db      string
}

// Apply makes MigratorDummy implement the Migrator interface.
func (m *DummyMigrator) Apply(version *Version, migrations []Migration) error {
	applied, err := GetMigrationInfo(m.Session, m.Db)
	if err != nil {
		return err
	}

	if len(applied) == 0 {
		return UpdateMigrationInfo(version, m.Session, m.Db)
	}

	return nil
}
