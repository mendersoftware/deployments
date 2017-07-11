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

package main

import "testing"

func TestCreateVersionString(t *testing.T) {
	testList := []struct {
		out    string
		tag    string
		commit string
		branch string
	}{
		{"unknown", "", "", ""},
		{"0.0.1", "0.0.1", "", ""},
		{"unknown", "", "9821b882934v", ""},
		{"master_9821b882934v", "", "9821b882934v", "master"},
	}

	for _, test := range testList {
		Tag = test.tag
		Commit = test.commit
		Branch = test.branch

		if str := CreateVersionString(); str != test.out {
			t.FailNow()
		}
	}
}
