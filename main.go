package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/sliceutil"
	toolresults "google.golang.org/api/toolresults/v1beta3"
)

const (
	testTypeInstrumentation = "instrumentation"
	testTypeRobo            = "robo"
)

type StepError struct {
	Step    *toolresults.Step
	Message string
}

func (e *StepError) Error() string {
	return fmt.Sprintf("error in step %s: %s", e.Step.Description, e.Message)
}

func failf(f string, v ...interface{}) {
	log.Errorf(f, v...)
	os.Exit(1)
}

func main() {
	var configs ConfigsModel
	if err := stepconf.Parse(&configs); err != nil {
		failf("Invalid input: %s", err)
	}

	if err := configs.validate(); err != nil {
		log.Errorf("Failed to parse config:")
		failf("%s", err)
	}

	fmt.Println()
	configs.print()

	log.SetEnableDebugLog(configs.VerboseLog)

	fmt.Println()
	successful := true

	log.Infof("Uploading app and test files")

	testAssets, err := uploadTestAssets(configs)
	if err != nil {
		failf("Failed to upload test assets, error: %s", err)
	}
	log.Donef("=> Files uploaded")

	fmt.Println()
	log.Infof("Starting test")

	if err = startTestRun(configs, testAssets); err != nil {
		failf("Starting test run failed, error: %s", err)
	}
	log.Donef("=> Test started")

	fmt.Println()
	log.Infof("Waiting for test results")
	{
		finished := false
		printedLogs := []string{}
		for !finished {
			url := configs.APIBaseURL + "/" + configs.AppSlug + "/" + configs.BuildSlug + "/" + configs.APIToken

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
				log.Donef("=> Test finished")
				fmt.Println()

				log.Infof("Test results:")
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
				if _, err := fmt.Fprintln(w, "Model\tAPI Level\tLocale\tOrientation\tOutcome\t"); err != nil {
					failf("Failed to write in tabwriter, error: %s", err)
				}

				retriesEnabled := configs.FlakyTestAttempts > 0
				successful, err = GetSuccessOfExecution(responseModel.Steps, retriesEnabled)
				if err != nil {
					failf("Failed to process results, error: %s", err)
				}

				for _, step := range responseModel.Steps {
					dimensions := map[string]string{}
					for _, dimension := range step.DimensionValue {
						dimensions[dimension.Key] = dimension.Value
					}

					outcome := processStepResult(step)

					if _, err := fmt.Fprintln(w, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t", dimensions["Model"], dimensions["Version"], dimensions["Locale"], dimensions["Orientation"], outcome)); err != nil {
						failf("Failed to write in tabwriter, error: %s", err)
					}
				}

				if err := w.Flush(); err != nil {
					log.Errorf("Failed to flush writer, error: %s", err)
				}
			}
			if !finished {
				time.Sleep(10 * time.Second)
			}
		}
	}

	if configs.DownloadTestResults {
		fmt.Println()
		log.Infof("Downloading test assets")
		{
			url := configs.APIBaseURL + "/assets/" + configs.AppSlug + "/" + configs.BuildSlug + "/" + configs.APIToken

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				failf("Failed to create http request, error: %s", err)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				failf("Failed to get http response, error: %s", err)
			}

			if resp.StatusCode != http.StatusOK {
				failf("Failed to get http response, status code: %d", resp.StatusCode)
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				failf("Failed to read response body, error: %s", err)
			}

			responseModel := map[string]string{}

			err = json.Unmarshal(body, &responseModel)
			if err != nil {
				failf("Failed to unmarshal response body, error: %s", err)
			}

			tempDir, err := pathutil.NormalizedOSTempDirPath("vdtesting_test_assets")
			if err != nil {
				failf("Failed to create temp dir, error: %s", err)
			}

			for fileName, fileURL := range responseModel {
				err := downloadFile(fileURL, filepath.Join(tempDir, fileName))
				if err != nil {
					failf("Failed to download file, error: %s", err)
				}
			}

			log.Donef("=> Assets downloaded")
			if err := tools.ExportEnvironmentWithEnvman("VDTESTING_DOWNLOADED_FILES_DIR", tempDir); err != nil {
				log.Warnf("Failed to export environment (VDTESTING_DOWNLOADED_FILES_DIR), error: %s", err)
			} else {
				log.Printf("The downloaded test assets path (%s) is exported to the VDTESTING_DOWNLOADED_FILES_DIR environment variable.", tempDir)
			}
		}
	}

	if !successful {
		os.Exit(1)
	}
}

func downloadFile(url string, localPath string) error {
	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("Failed to open the local cache file for write: %s", err)
	}
	defer func() {
		if err := out.Close(); err != nil {
			log.Printf("Failed to close Archive download file (%s): %s", localPath, err)
		}
	}()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Failed to create cache download request: %s", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close Archive download response body: %s", err)
		}
	}()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Failed to download archive - non success response code: %d", resp.StatusCode)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("Failed to save cache content into file: %s", err)
	}

	return nil
}

func uploadFile(uploadURL string, archiveFilePath string) error {
	archFile, err := os.Open(archiveFilePath)
	if err != nil {
		return fmt.Errorf("Failed to open archive file for upload (%s): %s", archiveFilePath, err)
	}
	isFileCloseRequired := true
	defer func() {
		if !isFileCloseRequired {
			return
		}
		if err := archFile.Close(); err != nil {
			log.Printf(" (!) Failed to close archive file (%s): %s", archiveFilePath, err)
		}
	}()

	fileInfo, err := archFile.Stat()
	if err != nil {
		return fmt.Errorf("Failed to get File Stats of the Archive file (%s): %s", archiveFilePath, err)
	}
	fileSize := fileInfo.Size()

	req, err := http.NewRequest("PUT", uploadURL, archFile)
	if err != nil {
		return fmt.Errorf("Failed to create upload request: %s", err)
	}

	req.Header.Add("Content-Length", strconv.FormatInt(fileSize, 10))
	req.ContentLength = fileSize

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to upload: %s", err)
	}
	isFileCloseRequired = false
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf(" [!] Failed to close response body: %s", err)
		}
	}()

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Failed to read response: %s", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Failed to upload file, response code was: %d", resp.StatusCode)
	}

	return nil
}

func processStepResult(step *toolresults.Step) string {
	outcome := step.Outcome.Summary

	switch outcome {
	case "success":
		outcome = colorstring.Green(outcome)
	case "failure":
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
	return outcome
}

func GetSuccessOfExecution(steps []*toolresults.Step, retriesEnabled bool) (bool, error) {
	if retriesEnabled == true {
		return getSuccessOfExecutionWithRetries(steps)
	} else {
		return getSuccessOfExecutionNoRetries(steps)
	}
}

func getSuccessOfExecutionWithRetries(steps []*toolresults.Step) (bool, error) {
	lastStepByDimension, err := getLastCompletedStepByDimension(steps)
	if err != nil {
		return false, err
	}

	for _, lastStep := range lastStepByDimension {
		if lastStep.Outcome.Summary != "success" {
			return false, nil
		}
	}
	return true, nil
}

func getSuccessOfExecutionNoRetries(steps []*toolresults.Step) (bool, error) {
	for _, step := range steps {
		if step.Outcome.Summary != "success" {
			return false, nil
		}
	}
	return true, nil
}

func getLastCompletedStepByDimension(steps []*toolresults.Step) (map[string]*toolresults.Step, error) {
	groupedByDimension := make(map[string]*toolresults.Step)
	for _, step := range steps {
		key, err := json.Marshal(step.DimensionValue)
		if err != nil {
			return nil, err
		}
		if key != nil {
			dimensionStr := string(key)

			if step.CompletionTime == nil {
				return nil, &StepError{
					Step:    step,
					Message: "Missing CompletionTime",
				}
			}

			if groupedByDimension[dimensionStr] == nil || groupedByDimension[dimensionStr].CompletionTime.Seconds < step.CompletionTime.Seconds {
				groupedByDimension[dimensionStr] = step
			}
		}
	}

	return groupedByDimension, nil
}
