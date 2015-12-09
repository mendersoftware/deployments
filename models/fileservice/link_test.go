package fileservice

import (
	"testing"
	"time"
)

func TestNewLink(t *testing.T) {
	now := time.Now()
	uri := "http://example.com"
	link := NewLink(uri, now)

	if link.Uri != uri {
		t.FailNow()
	}

	if link.Expire != now {
		t.FailNow()
	}
}
