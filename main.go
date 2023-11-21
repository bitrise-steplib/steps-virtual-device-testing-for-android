package main

import (
	"fmt"
	"os"
	"time"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-utils/log"
	toolresults "google.golang.org/api/toolresults/v1beta3"
)

const (
	testTypeInstrumentation = "instrumentation"
	testTypeRobo            = "robo"
)

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
	log.TInfof("Uploading app and test files")

	testAssets, err := uploadTestAssets(configs)
	if err != nil {
		failf("Failed to upload test assets, error: %s", err)
	}
	log.TDonef("=> Files uploaded")

	fmt.Println()
	log.TInfof("Starting test")

	if err = startTestRun(configs, testAssets); err != nil {
		failf("Starting test run failed, error: %s", err)
	}
	log.TDonef("=> Test started")

	fmt.Println()
	log.TInfof("Waiting for test results")

	url := testResultsURL(configs.APIBaseURL, configs.AppSlug, configs.BuildSlug, configs.APIToken)

	finished := false
	var steps []*toolresults.Step
	var printedLogs []string

	for !finished {
		steps, printedLogs = fetchTestResults(url, printedLogs)
		if steps == nil {
			time.Sleep(5 * time.Second)
		} else {
			finished = true
		}
	}

	successful := printTestResult(steps)

	if configs.DownloadTestResults {
		fmt.Println()
		log.Infof("Downloading test assets")

		url := testAssetsDownloadURL(configs.APIBaseURL, configs.AppSlug, configs.BuildSlug, configs.APIToken)
		assetsDir, err := downloadTestAssets(url)
		if err != nil {
			failf("%s", err)
		}

		log.Donef("=> Assets downloaded")
		if err := tools.ExportEnvironmentWithEnvman("VDTESTING_DOWNLOADED_FILES_DIR", assetsDir); err != nil {
			log.Warnf("Failed to export environment (VDTESTING_DOWNLOADED_FILES_DIR), error: %s", err)
		} else {
			log.Printf("The downloaded test assets path (%s) is exported to the VDTESTING_DOWNLOADED_FILES_DIR environment variable.", assetsDir)
		}
	}

	if !successful {
		os.Exit(1)
	}
}

func failf(f string, v ...interface{}) {
	log.Errorf(f, v...)
	os.Exit(1)
}
