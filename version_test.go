package main

import "testing"

func TestCreateVersionString(t *testing.T) {
	testList := []struct {
		out    string
		tag    string
		commit string
		branch string
		build  string
	}{
		{"Version: unknown", "", "", "", ""},
		{"Version: 0.0.1", "0.0.1", "", "", ""},
		{"Version: 0.0.1 BuildNumber: 123", "0.0.1", "", "", "123"},
		{"Version: unknown", "", "9821b882934v", "", ""},
		{"Version: unknown BuildNumber: 234", "", "9821b882934v", "", "234"},
		{"Version: master_9821b882934v", "", "9821b882934v", "master", ""},
		{"Version: master_9821b882934v BuildNumber: 234", "", "9821b882934v", "master", "234"},
	}

	for _, test := range testList {
		Tag = test.tag
		Commit = test.commit
		Branch = test.branch
		BuildNumber = test.build

		if str := CreateVersionString(); str != test.out {
			t.FailNow()
		}
	}
}
