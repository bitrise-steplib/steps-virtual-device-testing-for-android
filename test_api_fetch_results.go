package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"text/tabwriter"

	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/sliceutil"
	toolresults "google.golang.org/api/toolresults/v1beta3"
)

func testResultsURL(apiBaseURL, appSlug, buildSlug, apiToken string) string {
	return apiBaseURL + "/" + appSlug + "/" + buildSlug + "/" + apiToken
}

func fetchTestResults(url string, printedLogs []string) ([]*toolresults.Step, []string) {
	finished := false

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		failf("Failed to create http request, error: %s", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		resp, err = client.Do(req)
		if err != nil {
			failf("Failed to get http response, error: %s", err)
		}
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		failf("Failed to read response body, error: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		failf("Failed to get test status, error: %s", string(body))
	}

	responseModel := &toolresults.ListStepsResponse{}

	err = json.Unmarshal(body, responseModel)
	if err != nil {
		failf("Failed to unmarshal response body, error: %s, body: %s", err, string(body))
	}

	finished = true
	testsRunning := 0
	for _, step := range responseModel.Steps {
		if step.State != "complete" {
			finished = false
			testsRunning++
		}
	}

	msg := ""
	if len(responseModel.Steps) == 0 {
		finished = false
		msg = fmt.Sprintf("- Validating")
	} else {
		msg = fmt.Sprintf("- (%d/%d) running", testsRunning, len(responseModel.Steps))
	}

	if !sliceutil.IsStringInSlice(msg, printedLogs) {
		log.Printf(msg)
		printedLogs = append(printedLogs, msg)
	}

	if finished {
		return responseModel.Steps, printedLogs
	}

	return nil, printedLogs
}

func printTestResult(steps []*toolresults.Step) bool {
	successful := false

	log.Donef("=> Test finished")
	fmt.Println()

	log.Infof("Test results:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if _, err := fmt.Fprintln(w, "Model\tAPI Level\tLocale\tOrientation\tOutcome\t"); err != nil {
		failf("Failed to write in tabwriter, error: %s", err)
	}

	for _, step := range steps {
		dimensions := map[string]string{}
		for _, dimension := range step.DimensionValue {
			dimensions[dimension.Key] = dimension.Value
		}

		outcome := step.Outcome.Summary

		switch outcome {
		case "success":
			outcome = colorstring.Green(outcome)
		case "failure":
			successful = false
			if step.Outcome.FailureDetail != nil {
				if step.Outcome.FailureDetail.Crashed {
					outcome += "(Crashed)"
				}
				if step.Outcome.FailureDetail.NotInstalled {
					outcome += "(NotInstalled)"
				}
				if step.Outcome.FailureDetail.OtherNativeCrash {
					outcome += "(OtherNativeCrash)"
				}
				if step.Outcome.FailureDetail.TimedOut {
					outcome += "(TimedOut)"
				}
				if step.Outcome.FailureDetail.UnableToCrawl {
					outcome += "(UnableToCrawl)"
				}
			}
			outcome = colorstring.Red(outcome)
		case "inconclusive":
			successful = false
			if step.Outcome.InconclusiveDetail != nil {
				if step.Outcome.InconclusiveDetail.AbortedByUser {
					outcome += "(AbortedByUser)"
				}
				if step.Outcome.InconclusiveDetail.InfrastructureFailure {
					outcome += "(InfrastructureFailure)"
				}
			}
			outcome = colorstring.Yellow(outcome)
		case "skipped":
			successful = false
			if step.Outcome.SkippedDetail != nil {
				if step.Outcome.SkippedDetail.IncompatibleAppVersion {
					outcome += "(IncompatibleAppVersion)"
				}
				if step.Outcome.SkippedDetail.IncompatibleArchitecture {
					outcome += "(IncompatibleArchitecture)"
				}
				if step.Outcome.SkippedDetail.IncompatibleDevice {
					outcome += "(IncompatibleDevice)"
				}
			}
			outcome = colorstring.Blue(outcome)
		}

		if _, err := fmt.Fprintln(w, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t", dimensions["Model"], dimensions["Version"], dimensions["Locale"], dimensions["Orientation"], outcome)); err != nil {
			failf("Failed to write in tabwriter, error: %s", err)
		}
	}
	if err := w.Flush(); err != nil {
		log.Errorf("Failed to flush writer, error: %s", err)
	}

	return successful
}
