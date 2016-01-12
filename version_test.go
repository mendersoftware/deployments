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
