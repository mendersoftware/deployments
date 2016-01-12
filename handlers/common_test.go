package handlers

import (
	"testing"
)

func TestMissingRequiredQueryMsg(t *testing.T) {
	if MissingRequiredQueryMsg("SomeName") != `Required query parameter missing: 'SomeName'` {
		t.FailNow()
	}
}
