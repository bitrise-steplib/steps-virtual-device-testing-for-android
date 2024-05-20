package main

import (
	"testing"

	toolresults "google.golang.org/api/toolresults/v1beta3"
)

func TestGetSuccessOfExecution_RetryEnabled_AllSucceed(t *testing.T) {
	steps := []*toolresults.Step{
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
			CompletionTime: &toolresults.Timestamp{Seconds: 1},
			Outcome:        &toolresults.Outcome{Summary: "success"},
		},
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "ios"}},
			CompletionTime: &toolresults.Timestamp{Seconds: 2},
			Outcome:        &toolresults.Outcome{Summary: "success"},
		},
	}

	isSuccess, err := GetSuccessOfExecution(steps, true)

	expected := true
	if err != nil {
		t.Errorf("Expected no errors. Go %s", err)
	}

	if isSuccess != expected {
		t.Errorf("Expected success to be %v, got %v", expected, isSuccess)
	}
}

func TestGetSuccessOfExecution_RetryEnabled_FirstFailThenSucceed(t *testing.T) {
	steps := []*toolresults.Step{
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
			CompletionTime: &toolresults.Timestamp{Seconds: 1},
			Outcome:        &toolresults.Outcome{Summary: "failure"},
		},
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
			CompletionTime: &toolresults.Timestamp{Seconds: 3},
			Outcome:        &toolresults.Outcome{Summary: "success"},
		},
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "ios"}},
			CompletionTime: &toolresults.Timestamp{Seconds: 2},
			Outcome:        &toolresults.Outcome{Summary: "success"},
		},
	}

	isSuccess, err := GetSuccessOfExecution(steps, true)
	if err != nil {
		t.Errorf("Expected no errors. Got %s", err)
	}
	if !isSuccess {
		t.Errorf("Expected success to be true, got false")
	}
}

func TestGetSuccessOfExecution_RetryEnabled_FailDifferentDimension(t *testing.T) {
	steps := []*toolresults.Step{
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
			CompletionTime: &toolresults.Timestamp{Seconds: 2},
			Outcome:        &toolresults.Outcome{Summary: "failure"},
		},
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
			CompletionTime: &toolresults.Timestamp{Seconds: 4},
			Outcome:        &toolresults.Outcome{Summary: "success"},
		},
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "ios"}},
			CompletionTime: &toolresults.Timestamp{Seconds: 3},
			Outcome:        &toolresults.Outcome{Summary: "failure"},
		},
	}

	isSuccess, err := GetSuccessOfExecution(steps, true)
	if err != nil {
		t.Errorf("Expected no errors. Got %s", err)
	}
	if isSuccess {
		t.Errorf("Expected success to be false, got true")
	}
}

func TestGetSuccessOfExecution_RetryDisabled_FailForDimension(t *testing.T) {
	steps := []*toolresults.Step{
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
			CompletionTime: &toolresults.Timestamp{Seconds: 1},
			Outcome:        &toolresults.Outcome{Summary: "failure"},
		},
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "ios"}},
			CompletionTime: &toolresults.Timestamp{Seconds: 2},
			Outcome:        &toolresults.Outcome{Summary: "success"},
		},
	}

	isSuccess, err := GetSuccessOfExecution(steps, false)
	if err != nil {
		t.Errorf("Expected no errors. Got %s", err)
	}
	if isSuccess {
		t.Errorf("Expected success to be false, got true")
	}
}

func TestGetSuccessOfExecution_RetryDisabled_FailBothDimensions(t *testing.T) {
	steps := []*toolresults.Step{
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
			CompletionTime: &toolresults.Timestamp{Seconds: 1},
			Outcome:        &toolresults.Outcome{Summary: "failure"},
		},
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "ios"}},
			CompletionTime: &toolresults.Timestamp{Seconds: 2},
			Outcome:        &toolresults.Outcome{Summary: "failure"},
		},
	}

	isSuccess, err := GetSuccessOfExecution(steps, false)
	if err != nil {
		t.Errorf("Expected no errors. Got %s", err)
	}
	if isSuccess {
		t.Errorf("Expected success to be false, got true")
	}
}

func TestGetSuccessOfExecution_RetryDisabled_AllSuccess(t *testing.T) {
	steps := []*toolresults.Step{
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
			CompletionTime: &toolresults.Timestamp{Seconds: 1},
			Outcome:        &toolresults.Outcome{Summary: "success"},
		},
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "ios"}},
			CompletionTime: &toolresults.Timestamp{Seconds: 2},
			Outcome:        &toolresults.Outcome{Summary: "success"},
		},
	}

	isSuccess, err := GetSuccessOfExecution(steps, false)
	if err != nil {
		t.Errorf("Expected no errors. Got %s", err)
	}
	if !isSuccess {
		t.Errorf("Expected success to be true, got false")
	}
}
