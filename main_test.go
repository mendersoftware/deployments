package main

import "testing"

func TestHandleConfigFile(t *testing.T) {

	if _, err := HandleConfigFile(""); err == nil {
		t.FailNow()
	}

	// Depends on default config being avaiable and correct (which is nice!)
	if _, err := HandleConfigFile("config.yaml"); err != nil {
		t.FailNow()
	}

}
