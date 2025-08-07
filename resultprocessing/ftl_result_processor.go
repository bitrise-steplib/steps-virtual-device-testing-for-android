package resultprocessing

//// GetSuccessOfExecution ...
//// Given multiple steps, return whether we can consider the overall execution successful
//// To do this, step executions are grouped by dimension (aka what device they ran on)
//// and each dimension must have a successful execution
//func GetSuccessOfExecution(steps []*toolresults.Step) (bool, error) {
//	outcomeByDimension, err := getOutcomeByDimension(steps)
//	if err != nil {
//		return false, err
//	}
//
//	for _, outcome := range outcomeByDimension {
//		if outcome.Summary != "success" {
//			return false, nil
//		}
//	}
//
//	return true, nil
//}
//
//func getOutcomeByDimension(steps []*toolresults.Step) (map[string]*toolresults.Outcome, error) {
//	groupedByDimension := make(map[string]*toolresults.Outcome)
//	for _, step := range steps {
//		key, err := json.Marshal(step.DimensionValue)
//		if err != nil {
//			return nil, err
//		}
//		if key != nil {
//			dimensionStr := string(key)
//			if groupedByDimension[dimensionStr] == nil || groupedByDimension[dimensionStr].Summary != "success" {
//				groupedByDimension[dimensionStr] = step.Outcome
//			}
//		}
//	}
//
//	return groupedByDimension, nil
//}
