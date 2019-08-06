package main

import (
	"testing"
)

func TestNormalize(t *testing.T) {
    cases := []struct{
        title string
        configs ConfigsModel
        fieldGetter func(ConfigsModel) string
        want string
    }{
        {
            title: "trim 's' from TestTimeout",
            configs: ConfigsModel{TestTimeout: "4.2s"},
            fieldGetter: func(normalized ConfigsModel) string {
                return normalized.TestTimeout
            },
            want: "4.2",
        },
    }

    for _, tt := range cases {
        if normalized := normalize(tt.configs); tt.fieldGetter(normalized) != tt.want {
            t.Fatalf("%s: got %s want %s", tt.title, tt.fieldGetter(normalized), tt.want)
        }
    }
}

func TestConfigsModelValidate(t *testing.T) {
    cases := []struct{
        title string
        testTimeout string
        wantError bool
    }{
        // happy path cases
        {
            title: "integer without s",
            testTimeout: "4",
            wantError: false,
        },
        {
            title: "integer with s",
            testTimeout: "4s",
            wantError: false,
        },
        {
            title: "fraction without s",
            testTimeout: "4.2",
            wantError: false,
        },
        {
            title: "fraction with s",
            testTimeout: "4.2s",
            wantError: false,
        },
        {
            title: "fraction max length",
            testTimeout: "4.123456789",
            wantError: false,
        },
        // error cases
        {
            title: "fraction too long",
            testTimeout: "4.1234567890",
            wantError: true,
        },
        {
            title: "incorrect separator",
            testTimeout: "4,2",
            wantError: true,
        },
        {
            title: "missing fraction part",
            testTimeout: "4.",
            wantError: true,
        },
        {
            title: "missing integer part",
            testTimeout: ".2",
            wantError: true,
        },
    }
    for _, tt := range cases {
        if err := validateTestTimeoutFormat(tt.testTimeout); (err != nil) != tt.wantError  {
            t.Fatalf("validateTestTimeoutFormat: %s: got %t want %t", tt.title, err != nil, tt.wantError)
        }
    }
    
}
