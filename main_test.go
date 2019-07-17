package main

import (
	"testing"
)

func TestValidateTimeout(t *testing.T) {
	if err := validateTimeout("2.123456789"); err != nil {
		t.Errorf("error: %s", err)
	}
	if validateTimeout("2.123") != nil {
		t.Errorf("error")
	}
	if validateTimeout("2") != nil {
		t.Errorf("error")
	}

	if validateTimeout("") == nil {
		t.Errorf("error")
	}
	if validateTimeout("2.") == nil {
		t.Errorf("error")
	}
	if validateTimeout(".1") == nil {
		t.Errorf("error")
	}
	if validateTimeout(".") == nil {
		t.Errorf("error")
	}

}