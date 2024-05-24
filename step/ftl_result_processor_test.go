package step

import (
	"testing"

	toolresults "google.golang.org/api/toolresults/v1beta3"
)

func TestGetSuccessOfExecution_AllSucceed(t *testing.T) {
	steps := []*toolresults.Step{
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
			Outcome:        &toolresults.Outcome{Summary: "success"},
		},
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "ios"}},
			Outcome:        &toolresults.Outcome{Summary: "success"},
		},
	}

	isSuccess, err := GetSuccessOfExecution(steps)

	expected := true
	if err != nil {
		t.Errorf("Expected no errors. Go %s", err)
	}

	if isSuccess != expected {
		t.Errorf("Expected success to be %v, got %v", expected, isSuccess)
	}
}

func TestGetSuccessOfExecution_FirstFailThenSucceed(t *testing.T) {
	steps := []*toolresults.Step{
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
			Outcome:        &toolresults.Outcome{Summary: "failure"},
		},
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
			Outcome:        &toolresults.Outcome{Summary: "success"},
		},
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "ios"}},
			Outcome:        &toolresults.Outcome{Summary: "success"},
		},
	}

	isSuccess, err := GetSuccessOfExecution(steps)
	if err != nil {
		t.Errorf("Expected no errors. Got %s", err)
	}
	if !isSuccess {
		t.Errorf("Expected success to be true, got false")
	}
}

func TestGetSuccessOfExecution_FailDifferentDimension(t *testing.T) {
	steps := []*toolresults.Step{
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
			Outcome:        &toolresults.Outcome{Summary: "failure"},
		},
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
			Outcome:        &toolresults.Outcome{Summary: "success"},
		},
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "ios"}},
			Outcome:        &toolresults.Outcome{Summary: "failure"},
		},
	}

	isSuccess, err := GetSuccessOfExecution(steps)
	if err != nil {
		t.Errorf("Expected no errors. Got %s", err)
	}
	if isSuccess {
		t.Errorf("Expected success to be false, got true")
	}
}

func TestGetSuccessOfExecution_FailForDimension(t *testing.T) {
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

	isSuccess, err := GetSuccessOfExecution(steps)
	if err != nil {
		t.Errorf("Expected no errors. Got %s", err)
	}
	if isSuccess {
		t.Errorf("Expected success to be false, got true")
	}
}

func TestGetSuccessOfExecution_FailBothDimensions(t *testing.T) {
	steps := []*toolresults.Step{
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
			Outcome:        &toolresults.Outcome{Summary: "failure"},
		},
		{
			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "ios"}},
			Outcome:        &toolresults.Outcome{Summary: "failure"},
		},
	}

	isSuccess, err := GetSuccessOfExecution(steps)
	if err != nil {
		t.Errorf("Expected no errors. Got %s", err)
	}
	if isSuccess {
		t.Errorf("Expected success to be false, got true")
	}
}
