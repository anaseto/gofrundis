package xhtml

import (
	"regexp"
	"testing"
)

func TestGenuuid(t *testing.T) {
	uuid, err := genuuid()
	if err != nil {
		t.Fatal(err)
	}
	matched, err := regexp.MatchString(`[\da-f]{8}-[\da-f]{4}-[\da-f]{4}-[\da-f]{4}-[\da-f]{12}`, uuid)
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Fatal("does not look like an uuid:", uuid)
	}
}
