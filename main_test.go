package main

import (
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

func TestMakeSortedCopyOfSteps_SortedOrder(t *testing.T) {
	steps := []*toolresults.Step{
		{CompletionTime: &toolresults.Timestamp{Seconds: 2}},
		{CompletionTime: &toolresults.Timestamp{Seconds: 3}},
		{CompletionTime: &toolresults.Timestamp{Seconds: 1}},
	}

	sortedSteps := makeSortedCopyOfSteps(steps)

	if sortedSteps[2].CompletionTime.Seconds != 3 ||
		sortedSteps[1].CompletionTime.Seconds != 2 ||
		sortedSteps[0].CompletionTime.Seconds != 1 {
		t.Errorf("Steps are not sorted in the correct order")
	}
}

func TestMakeSortedCopyOfSteps_WithNilCompletionTimes(t *testing.T) {
	// Define a slice of steps where some have nil completion times
	steps := []*toolresults.Step{
		{CompletionTime: nil},
		{CompletionTime: &toolresults.Timestamp{Seconds: 1}},
		{CompletionTime: nil},
	}

	// Call the function to make a sorted copy
	sortedSteps := makeSortedCopyOfSteps(steps)

	// Check if the steps with nil completion times are at the end of the slice
	if sortedSteps[0].CompletionTime != nil || sortedSteps[1].CompletionTime != nil {
		t.Errorf("Steps with nil completion times are not sorted correctly")
	}
}

func TestMakeSortedCopyOfSteps_OriginalSliceUnmodified(t *testing.T) {
	// Define a slice of steps
	steps := []*toolresults.Step{
		{CompletionTime: &toolresults.Timestamp{Seconds: 2}},
		{CompletionTime: &toolresults.Timestamp{Seconds: 1}},
	}

	// Make a copy of the original slice for comparison
	originalSteps := make([]*toolresults.Step, len(steps))
	copy(originalSteps, steps)

	// Call the function to make a sorted copy
	_ = makeSortedCopyOfSteps(steps)

	// Check if the original slice is unmodified
	for i, step := range steps {
		if step != originalSteps[i] {
			t.Errorf("The original slice has been modified")
		}
	}
}
