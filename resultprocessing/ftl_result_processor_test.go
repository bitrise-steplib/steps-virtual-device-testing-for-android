package resultprocessing

//func TestGetSuccessOfExecution_AllSucceed(t *testing.T) {
//	steps := []*toolresults.Step{
//		{
//			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
//			Outcome:        &toolresults.Outcome{Summary: "success"},
//		},
//		{
//			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "ios"}},
//			Outcome:        &toolresults.Outcome{Summary: "success"},
//		},
//	}
//
//	isSuccess, err := GetSuccessOfExecution(steps)
//	require.NoError(t, err)
//	require.True(t, isSuccess)
//}
//
//func TestGetSuccessOfExecution_FirstFailThenSucceed(t *testing.T) {
//	steps := []*toolresults.Step{
//		{
//			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
//			Outcome:        &toolresults.Outcome{Summary: "failure"},
//		},
//		{
//			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
//			Outcome:        &toolresults.Outcome{Summary: "success"},
//		},
//		{
//			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "ios"}},
//			Outcome:        &toolresults.Outcome{Summary: "success"},
//		},
//	}
//
//	isSuccess, err := GetSuccessOfExecution(steps)
//	require.NoError(t, err)
//	require.True(t, isSuccess)
//}
//
//func TestGetSuccessOfExecution_FailDifferentDimension(t *testing.T) {
//	steps := []*toolresults.Step{
//		{
//			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
//			Outcome:        &toolresults.Outcome{Summary: "failure"},
//		},
//		{
//			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
//			Outcome:        &toolresults.Outcome{Summary: "success"},
//		},
//		{
//			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "ios"}},
//			Outcome:        &toolresults.Outcome{Summary: "failure"},
//		},
//	}
//
//	isSuccess, err := GetSuccessOfExecution(steps)
//	require.NoError(t, err)
//	require.False(t, isSuccess)
//}
//
//func TestGetSuccessOfExecution_FailForDimension(t *testing.T) {
//	steps := []*toolresults.Step{
//		{
//			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
//			CompletionTime: &toolresults.Timestamp{Seconds: 1},
//			Outcome:        &toolresults.Outcome{Summary: "failure"},
//		},
//		{
//			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "ios"}},
//			CompletionTime: &toolresults.Timestamp{Seconds: 2},
//			Outcome:        &toolresults.Outcome{Summary: "success"},
//		},
//	}
//
//	isSuccess, err := GetSuccessOfExecution(steps)
//	require.NoError(t, err)
//	require.False(t, isSuccess)
//}
//
//func TestGetSuccessOfExecution_FailBothDimensions(t *testing.T) {
//	steps := []*toolresults.Step{
//		{
//			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "android"}},
//			Outcome:        &toolresults.Outcome{Summary: "failure"},
//		},
//		{
//			DimensionValue: []*toolresults.StepDimensionValueEntry{{Key: "os", Value: "ios"}},
//			Outcome:        &toolresults.Outcome{Summary: "failure"},
//		},
//	}
//
//	isSuccess, err := GetSuccessOfExecution(steps)
//	require.NoError(t, err)
//	require.False(t, isSuccess)
//}
