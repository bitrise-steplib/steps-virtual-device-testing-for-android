package main

import (
	"testing"
)

func TestValidateTimeout(t *testing.T) {
	testcases := []struct{
		title    string
		duration string
		wantE    bool
	}{
		{
			title:    "nine digit fraction",
			duration: "2.123456789",
			wantE:    false,
		},
		{
			title:    "three digit fraction",
			duration: "2.123",
			wantE:    false,
		},
		{
			title:    "plain integer",
			duration: "2",
			wantE:    false,
		},
		{
			title:    "empty value",
			duration: "",
			wantE:    true,
		},
		{
			title:    "invalid fraction delimiter",
			duration: "2,123",
			wantE:    true,
		},
		{
			title:    "missing fraction digits",
			duration: "2.",
			wantE:    true,
		},
		{
			title:    "missing integer digits",
			duration: ".123",
			wantE:    true,
		},
		{
			title:    "only a dot",
			duration: ".",
			wantE:    true,
		},
	}
	for _, tt := range testcases {
		if err := validateTimeout(tt.duration); (err != nil) != tt.wantE {
			t.Errorf("%s: got error (%s), but want error is %t", tt.title, err, tt.wantE)
		}

	} 
}
