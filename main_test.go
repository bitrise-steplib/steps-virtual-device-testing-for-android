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
