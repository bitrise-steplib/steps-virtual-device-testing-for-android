package main

import (
	"encoding/json"
	"reflect"
	"testing"

	toolresults "google.golang.org/api/toolresults/v1beta3"
)

func TestGetNewSuccessValue_FailedStep(t *testing.T) {
	result := getNewSuccessValue(true, false, false, false)
	if result != false {
		t.Errorf("Expected false, got %v", result)
	}
}

func TestGetNewSuccessValue_SuccessfulFinalStepWithFlakyRetries(t *testing.T) {
	result := getNewSuccessValue(false, true, true, true)
	if result != true {
		t.Errorf("Expected true, got %v", result)
	}
}

func TestGetNewSuccessValue_SuccessfulFinalStepWithoutFlakyRetries(t *testing.T) {
	result := getNewSuccessValue(false, true, true, false)
	if result != false {
		t.Errorf("Expected false, got %v", result)
	}
}

func TestGetNewSuccessValue_NonFinalSuccessfulStep(t *testing.T) {
	result := getNewSuccessValue(false, true, false, true)
	if result != false {
		t.Errorf("Expected false, got %v", result)
	}
}

func TestGroupedSortedSteps(t *testing.T) {
	// Mock steps with dimension values and completion times.

	android1 := toolresults.Step{
		DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
		CompletionTime: &toolresults.Timestamp{Seconds: 1},
	}

	android2 := toolresults.Step{
		DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
		CompletionTime: &toolresults.Timestamp{Seconds: 2},
	}

	android3 := toolresults.Step{
		DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
		CompletionTime: &toolresults.Timestamp{Seconds: 3},
	}

	iOS1 := toolresults.Step{
		DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "ios"}},
		CompletionTime: &toolresults.Timestamp{Seconds: 1},
	}

	steps := []*toolresults.Step{&android3, &android1, &iOS1, &android2}

	expected := make(map[string][]*toolresults.Step)
	androidKey, _ := json.Marshal(android1.DimensionValue)
	iosKey, _ := json.Marshal(iOS1.DimensionValue)
	expected[string(androidKey)] = []*toolresults.Step{&android1, &android2, &android3}
	expected[string(iosKey)] = []*toolresults.Step{&iOS1}

	actual := groupedSortedSteps(steps)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("groupedSortedSteps() = %v, want %v", actual, expected)
	}
}
