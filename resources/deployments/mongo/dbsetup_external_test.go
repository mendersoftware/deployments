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

package mongo_test

import (
	"os"
	"testing"

	"gopkg.in/mgo.v2/dbtest"
)

var db *dbtest.DBServer

const (
	DefaultDBDir = "/tmp"
)

// Overwrites test execution and allows for test database setup
func TestMain(m *testing.M) {

	// os.Exit would ignore defers, workaround
	status := func() int {
		// Start test database server
		db = &dbtest.DBServer{}
		db.SetPath(DefaultDBDir)
		// Tier down databaser server
		// Note: 	if test panics, it will require manual database tier down
		//			testing package executes tests in goroutines therefore
		//			we can't catch panics issued in tests.
		// db.Stop()
		defer db.Stop()
		return m.Run()
	}()

	os.Exit(status)
}
