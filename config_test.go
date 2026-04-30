package main

import (
	"reflect"
	"testing"
)

func TestParseQuarantinedTests(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			name:  "plain test name",
			input: `[{"className":"pkg.Cls","testCaseName":"foo"}]`,
			want:  []string{"notClass pkg.Cls#foo"},
		},
		{
			name:  "JUnit indexed parameter suffix",
			input: `[{"className":"pkg.Cls","testCaseName":"foo[0]"}]`,
			want:  []string{"notClass pkg.Cls#foo"},
		},
		{
			name:  "JUnit5 named parameter suffix (SSW-2961 reproducer)",
			input: `[{"className":"com.syscocorp.mss.products.ProductDetailsFragmentTest","testCaseName":"test_show_product_specialOffer_banner_when_offer_available[1: TestData(unified=false)]"}]`,
			want:  []string{"notClass com.syscocorp.mss.products.ProductDetailsFragmentTest#test_show_product_specialOffer_banner_when_offer_available"},
		},
		{
			name:  "whitespace before parameter suffix",
			input: `[{"className":"pkg.Cls","testCaseName":"foo [0]"}]`,
			want:  []string{"notClass pkg.Cls#foo"},
		},
		{
			name:  "empty test case name is skipped",
			input: `[{"className":"pkg.Cls","testCaseName":""}]`,
			want:  nil,
		},
		{
			name:  "empty class name is skipped",
			input: `[{"className":"","testCaseName":"foo"}]`,
			want:  nil,
		},
		{
			name:  "test case name reduces to empty after stripping is skipped",
			input: `[{"className":"pkg.Cls","testCaseName":"[0]"}]`,
			want:  nil,
		},
		{
			name:  "multiple entries mix plain and parametrized",
			input: `[{"className":"pkg.A","testCaseName":"foo"},{"className":"pkg.B","testCaseName":"bar[1: TestData(unified=false)]"}]`,
			want: []string{
				"notClass pkg.A#foo",
				"notClass pkg.B#bar",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseQuarantinedTests(tc.input)
			if err != nil {
				t.Fatalf("parseQuarantinedTests() returned error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parseQuarantinedTests() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestParseQuarantinedTests_InvalidJSON(t *testing.T) {
	_, err := parseQuarantinedTests("not json")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
