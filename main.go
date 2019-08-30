package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	testing "google.golang.org/api/testing/v1"
	toolresults "google.golang.org/api/toolresults/v1beta3"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-tools/go-steputils/input"
	"github.com/bitrise-tools/go-steputils/tools"
)

const (
	maxTimeoutSeconds       = 1800
	testTypeInstrumentation = "instrumentation"
	testTypeRobo            = "robo"
)

// ConfigsModel ...
type ConfigsModel struct {
	// api
	APIBaseURL string `env:"api_base_url"`
	BuildSlug  string `env:"BITRISE_BUILD_SLUG,required"`
	AppSlug    string `env:"BITRISE_APP_SLUG,required"`
	APIToken   string `env:"api_token,required"`

	// shared
	AppPath      string `env:"app_path"`
	TestApkPath  string `env:"test_apk_path"`
	TestType     string `env:"test_type,opt[instrumentation,robo,gameloop]"`
	TestDevices  string `env:"test_devices"`
	AppPackageID string `env:"app_package_id"`

	// test setup
	AutoGoogleLogin      bool   `env:"auto_google_login,opt[true,false]"`
	EnvironmentVariables string `env:"environment_variables"`
	obbFilesList         string `env:"obb_files_list"`
	ObbFiles             []string

	// shared debug
	TestTimeout         float64 `env:"test_timeout,required"`
	FlakyTestAttempts   int     `env:"num_flaky_test_attempts,range[0..10]"`
	DownloadTestResults string  `env:"download_test_results"`
	DirectoriesToPull   string  `env:"directories_to_pull"`
	VerboseLog          bool    `env:"use_verbose_log,opt[true,false]"`

	// instrumentation
	InstTestPackageID   string `env:"inst_test_package_id"`
	InstTestRunnerClass string `env:"inst_test_runner_class"`
	InstTestTargets     string `env:"inst_test_targets"`
	UseOrchestrator     bool   `env:"inst_use_orchestrator,opt[true,false]"`

	// robo
	RoboInitialActivity string `env:"robo_initial_activity"`
	RoboDirectives      string `env:"robo_directives"`
	RoboScenarioFile    string `env:"robo_scenario_file"`
	RoboMaxDepth        string `env:"robo_max_depth"`
	RoboMaxSteps        string `env:"robo_max_steps"`

	// loop
	LoopScenarios       string `env:"loop_scenarios"`
	LoopScenarioLabels  string `env:"loop_scenario_labels"`
	LoopScenarioNumbers string `env:"loop_scenario_numbers"`

	// deprecated
	ApkPath string `env:"apk_path"`
}

func (configs ConfigsModel) print() {
	log.Infof("Configs:")
	log.Printf("- AppPath: %s", configs.AppPath)
	if configs.ApkPath != "" {
		log.Printf("- ApkPath: %s", configs.ApkPath)
	}
	if configs.AppPackageID != "" {
		log.Printf("- AppPackageID: %s", configs.AppPackageID)
	}
	log.Printf("- TestTimeout: %s", configs.TestTimeout)
	log.Printf("- FlakyTestAttempts: %s", configs.FlakyTestAttempts)
	log.Printf("- DownloadTestResults: %s", configs.DownloadTestResults)
	log.Printf("- DirectoriesToPull: %s", configs.DirectoriesToPull)
	log.Printf("- AutoGoogleLogin: %s", configs.AutoGoogleLogin)
	log.Printf("- EnvironmentVariables: %s", configs.EnvironmentVariables)
	log.Printf("- ObbFilesList: %s", configs.obbFilesList)
	log.Printf("- TestDevices:\n---")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if _, err := fmt.Fprintln(w, "Model\tAPI Level\tLocale\tOrientation\t"); err != nil {
		failf("Failed to write in tabwriter, error: %s", err)
	}
	scanner := bufio.NewScanner(strings.NewReader(configs.TestDevices))
	for scanner.Scan() {
		device := scanner.Text()
		device = strings.TrimSpace(device)
		if device == "" {
			continue
		}

		deviceParams := strings.Split(device, ",")

		if len(deviceParams) != 4 {
			continue
		}

		if _, err := fmt.Fprintln(w, fmt.Sprintf("%s\t%s\t%s\t%s\t", deviceParams[0], deviceParams[1], deviceParams[3], deviceParams[2])); err != nil {
			failf("Failed to write in tabwriter, error: %s", err)
		}
	}
	if err := w.Flush(); err != nil {
		log.Errorf("Failed to flush writer, error: %s", err)
	}

	log.Printf("---")
	log.Printf("- TestType: %s", configs.TestType)
	// instruments
	if configs.TestType == testTypeInstrumentation {
		log.Printf("- TestApkPath: %s", configs.TestApkPath)
		log.Printf("- InstTestPackageID: %s", configs.InstTestPackageID)
		log.Printf("- InstTestRunnerClass: %s", configs.InstTestRunnerClass)
		log.Printf("- InstTestTargets: %s", configs.InstTestTargets)
		log.Printf("- UseOrchestrator: %s", configs.UseOrchestrator)
	}

	//robo
	if configs.TestType == testTypeRobo {
		log.Printf("- RoboInitialActivity: %s", configs.RoboInitialActivity)
		log.Printf("- RoboScenarioFile: %s", configs.RoboScenarioFile)
		log.Printf("- RoboDirectives: %s", configs.RoboDirectives)
		log.Printf("- RoboMaxDepth: %s", configs.RoboMaxDepth)
		log.Printf("- RoboMaxSteps: %s", configs.RoboMaxSteps)
	}

	// loop
	if configs.TestType == "gameloop" {
		log.Printf("- LoopScenarios: %s", configs.LoopScenarios)
		log.Printf("- LoopScenarioLabels: %s", configs.LoopScenarioLabels)
		log.Printf("- LoopScenarioNumbers: %s", configs.LoopScenarioNumbers)
	}
}

func (configs ConfigsModel) validate() error {
	if err := input.ValidateIfNotEmpty(configs.APIBaseURL); err != nil {
		if _, set := os.LookupEnv("BITRISE_IO"); !set {
			log.Warnf("Warning: please make sure that Virtual Device Testing add-on is turned on under your app's settings tab.")
		}
		return fmt.Errorf("Issue with APIBaseURL: %s", err)
	}

	if configs.ApkPath != "" {
		configs.AppPath = configs.ApkPath
	}
	if err := input.ValidateIfNotEmpty(configs.AppPath); err != nil {
		return fmt.Errorf("Issue with ApkPath: %s", err)
	}
	if err := input.ValidateIfPathExists(configs.AppPath); err != nil {
		return fmt.Errorf("Issue with ApkPath: %s", err)
	}

	if configs.TestType == testTypeInstrumentation {
		if err := input.ValidateIfNotEmpty(configs.TestApkPath); err != nil {
			return fmt.Errorf("Issue with TestApkPath: %s. Is it possible that you used gradle-runner step and forgot to set `assembleDebugAndroidTest` task?", err)
		}
		if err := input.ValidateIfPathExists(configs.TestApkPath); err != nil {
			return fmt.Errorf("Issue with TestApkPath: %s. Is it possible that you used gradle-runner step and forgot to set `assembleDebugAndroidTest` task?", err)
		}
	}

	configs.RoboScenarioFile = strings.TrimSpace(configs.RoboScenarioFile)
	if configs.TestType == testTypeRobo && configs.RoboScenarioFile != "" {
		if err := input.ValidateIfPathExists(configs.RoboScenarioFile); err != nil {
			return fmt.Errorf("Issue with RoboScenarioFile: %s", err)
		}
	}

	var err error
	if configs.ObbFiles, err = parseObbFilesList(configs.obbFilesList); err != nil {
		return err
	}

	return nil
}

func parseObbFilesList(obbFilesList string) ([]string, error) {
	var obbFiles []string
	files := strings.Split(obbFilesList, "\n")

	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		if _, err := os.Stat(file); err != nil {
			return nil, fmt.Errorf("could not get file info for obb file (%s), error: %s", file, err)
		}

		obbFiles = append(obbFiles, file)
	}

	return obbFiles, nil
}

func failf(f string, v ...interface{}) {
	log.Errorf(f, v...)
	os.Exit(1)
}

// TestAsset describes a requested test asset
type TestAsset struct {
	UploadURL string `json:"uploadUrl"`
	GcsPath   string `json:"gcsPath"`
	Filename  string `json:"filename"`
}

// TestAssetsAndroid describes requested Android test asset and as the returned test asset upload URLs
type TestAssetsAndroid struct {
	Apk        TestAsset   `json:"apk,omitempty"`
	Aab        TestAsset   `json:"aab,omitmepty"`
	TestApk    TestAsset   `json:"testApk,omitempty"`
	RoboScript TestAsset   `json:"roboScript,omitempty"`
	ObbFiles   []TestAsset `json:"obbFiles,omitempty"`
}

func main() {
	var configs ConfigsModel
	if err := stepconf.Parse(&configs); err != nil {
		failf("Invalid input: %s", err)
	}

	fmt.Println()
	configs.print()

	if err := configs.validate(); err != nil {
		failf("%s", err)
	}
	log.SetEnableDebugLog(configs.VerboseLog)

	if configs.ApkPath != "" {
		log.Warnf("'Apk path' (apk_path) input is deprected, use 'App path' (app_path) instead.")
		log.Warnf("'Apk path' (%s) is specified, overrides App path (%s)", configs.ApkPath, configs.AppPath)
		configs.AppPath = configs.ApkPath
	}
	if configs.AppPackageID != "" {
		log.Warnf("'App package ID' (app_package_id) input is deprecated. Leave empty to automatically extract it from the App manifest")
	}
	if configs.InstTestPackageID != "" {
		log.Warnf("'Test package ID' (inst_test_package_id) input is deprecatad. Leave empty to automatically extract it from the App manifest")
	}

	fmt.Println()
	successful := true

	var isBundle bool
	log.Infof("Uploading app and test files")
	var testAssets TestAssetsAndroid
	var testApp TestAsset
	{
		url := configs.APIBaseURL + "/assets/android/" + configs.AppSlug + "/" + configs.BuildSlug + "/" + configs.APIToken

		if strings.ToLower(filepath.Ext(configs.AppPath)) == ".aab" {
			isBundle = true
		}
		log.Debugf("App path (%s), is bundle: %b", configs.AppPath, isBundle)

		var requestedAssets TestAssetsAndroid
		if isBundle {
			requestedAssets.Aab = TestAsset{
				Filename: filepath.Base(configs.AppPath),
			}
		} else {
			requestedAssets.Apk = TestAsset{
				Filename: filepath.Base(configs.AppPath),
			}
		}
		if configs.TestType == testTypeInstrumentation {
			requestedAssets.TestApk = TestAsset{
				Filename: filepath.Base(configs.TestApkPath),
			}
		}
		if configs.TestType == testTypeRobo && configs.RoboScenarioFile != "" {
			requestedAssets.RoboScript = TestAsset{
				Filename: filepath.Base(configs.RoboScenarioFile),
			}
		}
		for _, obbFile := range configs.ObbFiles {
			requestedAssets.ObbFiles = append(requestedAssets.ObbFiles, TestAsset{
				Filename: filepath.Base(obbFile),
			})
		}

		log.Debugf("Assets requested: %+v", requestedAssets)

		data, err := json.Marshal(requestedAssets)
		if err != nil {
			failf("Failed to encode to json: %s", requestedAssets)
		}

		req, err := http.NewRequest("POST", url, bytes.NewReader(data))
		if err != nil {
			failf("Failed to create http request, error: %s", err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			failf("Failed to get http response, error: %s", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				failf("Failed to read response body, error: %s", err)
			}
			failf("Failed to start test: %d, error: %s", resp.StatusCode, string(body))
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			failf("Failed to read response body, error: %s", err)
		}

		err = json.Unmarshal(body, &testAssets)
		if err != nil {
			failf("Failed to unmarshal response body, error: %s", err)
		}

		if isBundle {
			testApp = testAssets.Aab
		} else {
			testApp = testAssets.Apk
		}
		log.Debugf("Uploading file(%s) to (%s)", configs.AppPath, testApp.GcsPath)

		err = uploadFile(testApp.UploadURL, configs.AppPath)
		if err != nil {
			failf("Failed to upload file(%s) to (%s), error: %s", configs.AppPath, testApp.UploadURL, err)
		}

		if configs.TestType == testTypeInstrumentation {
			err = uploadFile(testAssets.TestApk.UploadURL, configs.TestApkPath)
			if err != nil {
				failf("Failed to upload file(%s) to (%s), error: %s", configs.TestApkPath, testAssets.TestApk.UploadURL, err)
			}
		}

		if len(testAssets.ObbFiles) != len(configs.ObbFiles) {
			failf("Invalid length of obb file upload URLs in response: %+v", testAssets)
		}
		for i, obbFile := range configs.ObbFiles {
			if err := uploadFile(testAssets.ObbFiles[i].UploadURL, obbFile); err != nil {
				failf("Failed to upload obb file (%s) to (%s), error: %s", obbFile, testAssets.ObbFiles[i].UploadURL, err)
			}
		}

		log.Donef("=> Files uploaded")
	}

	fmt.Println()
	log.Infof("Start test")
	{
		url := configs.APIBaseURL + "/" + configs.AppSlug + "/" + configs.BuildSlug + "/" + configs.APIToken

		testModel := &testing.TestMatrix{}
		testModel.EnvironmentMatrix = &testing.EnvironmentMatrix{AndroidDeviceList: &testing.AndroidDeviceList{}}
		testModel.EnvironmentMatrix.AndroidDeviceList.AndroidDevices = []*testing.AndroidDevice{}

		scanner := bufio.NewScanner(strings.NewReader(configs.TestDevices))
		for scanner.Scan() {
			device := scanner.Text()
			device = strings.TrimSpace(device)
			if device == "" {
				continue
			}

			deviceParams := strings.Split(device, ",")
			if len(deviceParams) != 4 {
				failf("Invalid test device configuration: %s", device)
			}

			newDevice := testing.AndroidDevice{
				AndroidModelId:   deviceParams[0],
				AndroidVersionId: deviceParams[1],
				Locale:           deviceParams[2],
				Orientation:      deviceParams[3],
			}

			testModel.EnvironmentMatrix.AndroidDeviceList.AndroidDevices = append(testModel.EnvironmentMatrix.AndroidDeviceList.AndroidDevices, &newDevice)
		}

		if configs.FlakyTestAttempts < 0 || configs.FlakyTestAttempts > 10 {
			failf("Invalid input 'Number of times a test execution is reattempted': has to be between 0 and 10, provided %s", configs.FlakyTestAttempts)
		}
		testModel.FlakyTestAttempts = int64(configs.FlakyTestAttempts)

		// parse directories to pull
		scanner = bufio.NewScanner(strings.NewReader(configs.DirectoriesToPull))
		directoriesToPull := []string{}
		for scanner.Scan() {
			path := scanner.Text()
			path = strings.TrimSpace(path)
			if path == "" {
				continue
			}
			directoriesToPull = append(directoriesToPull, path)
		}

		// parse environment variables
		scanner = bufio.NewScanner(strings.NewReader(configs.EnvironmentVariables))
		envs := []*testing.EnvironmentVariable{}
		for scanner.Scan() {
			envStr := scanner.Text()

			if envStr == "" {
				continue
			}

			if !strings.Contains(envStr, "=") {
				continue
			}

			envStrSplit := strings.Split(envStr, "=")
			envKey := envStrSplit[0]
			envValue := strings.Join(envStrSplit[1:], "=")

			envs = append(envs, &testing.EnvironmentVariable{Key: envKey, Value: envValue})
		}

		// obb files to upload
		var filesToPush []*testing.DeviceFile
		for _, obbFile := range testAssets.ObbFiles {
			filesToPush = append(filesToPush, &testing.DeviceFile{
				ObbFile: &testing.ObbFile{
					Obb: &testing.FileReference{
						GcsPath: obbFile.GcsPath,
					},
					ObbFileName: obbFile.Filename,
				},
			})
		}

		if configs.TestTimeout > float64(maxTimeoutSeconds) {
			log.Warnf("timeout value (%f) is greater than available maximum (%f). Maximum will be used instead.", configs.TestTimeout, maxTimeoutSeconds)
			configs.TestTimeout = maxTimeoutSeconds
		}

		// a nil account does not log in to test Google account before test is started
		var account *testing.Account
		if configs.AutoGoogleLogin {
			account = &testing.Account{
				GoogleAuto: &testing.GoogleAuto{},
			}
		}

		testModel.TestSpecification = &testing.TestSpecification{
			TestTimeout: fmt.Sprintf("%fs", configs.TestTimeout),
			TestSetup: &testing.TestSetup{
				EnvironmentVariables: envs,
				FilesToPush:          filesToPush,
				DirectoriesToPull:    directoriesToPull,
				Account:              account,
			},
		}

		switch configs.TestType {
		case testTypeInstrumentation:
			testModel.TestSpecification.AndroidInstrumentationTest = &testing.AndroidInstrumentationTest{}

			if isBundle {
				testModel.TestSpecification.AndroidInstrumentationTest.AppBundle = &testing.AppBundle{
					BundleLocation: &testing.FileReference{GcsPath: testApp.GcsPath},
				}
			} else {
				testModel.TestSpecification.AndroidInstrumentationTest.AppApk = &testing.FileReference{GcsPath: testApp.GcsPath}
			}

			testModel.TestSpecification.AndroidInstrumentationTest.TestApk = &testing.FileReference{GcsPath: testAssets.TestApk.GcsPath}
			if configs.AppPackageID != "" {
				testModel.TestSpecification.AndroidInstrumentationTest.AppPackageId = configs.AppPackageID
			}
			if configs.InstTestPackageID != "" {
				testModel.TestSpecification.AndroidInstrumentationTest.TestPackageId = configs.InstTestPackageID
			}
			if configs.InstTestRunnerClass != "" {
				testModel.TestSpecification.AndroidInstrumentationTest.TestRunnerClass = configs.InstTestRunnerClass
			}
			if configs.InstTestTargets != "" {
				targets := strings.Split(strings.TrimSpace(configs.InstTestTargets), ",")
				testModel.TestSpecification.AndroidInstrumentationTest.TestTargets = targets
			}
			if configs.UseOrchestrator {
				testModel.TestSpecification.AndroidInstrumentationTest.OrchestratorOption = "USE_ORCHESTRATOR"
			} else {
				testModel.TestSpecification.AndroidInstrumentationTest.OrchestratorOption = "DO_NOT_USE_ORCHESTRATOR"
			}
			log.Debugf("AndroidInstrumentationTest: %+v", testModel.TestSpecification.AndroidInstrumentationTest)
		case testTypeRobo:
			testModel.TestSpecification.AndroidRoboTest = &testing.AndroidRoboTest{}

			if isBundle {
				testModel.TestSpecification.AndroidRoboTest.AppBundle = &testing.AppBundle{
					BundleLocation: &testing.FileReference{GcsPath: testApp.GcsPath},
				}
			} else {
				testModel.TestSpecification.AndroidRoboTest.AppApk = &testing.FileReference{GcsPath: testApp.GcsPath}
			}

			if configs.AppPackageID != "" {
				testModel.TestSpecification.AndroidRoboTest.AppPackageId = configs.AppPackageID
			}
			if configs.RoboInitialActivity != "" {
				testModel.TestSpecification.AndroidRoboTest.AppInitialActivity = configs.RoboInitialActivity
			}
			if configs.RoboMaxDepth != "" {
				maxDepth, err := strconv.Atoi(configs.RoboMaxDepth)
				if err != nil {
					failf("Failed to parse string(%s) to integer, error: %s", configs.RoboMaxDepth, err)
				}
				testModel.TestSpecification.AndroidRoboTest.MaxDepth = int64(maxDepth)
			}
			if configs.RoboMaxSteps != "" {
				maxSteps, err := strconv.Atoi(configs.RoboMaxSteps)
				if err != nil {
					failf("Failed to parse string(%s) to integer, error: %s", configs.RoboMaxSteps, err)
				}
				testModel.TestSpecification.AndroidRoboTest.MaxSteps = int64(maxSteps)
			}
			if configs.RoboDirectives != "" {
				roboDirectives := []*testing.RoboDirective{}
				scanner := bufio.NewScanner(strings.NewReader(configs.RoboDirectives))
				for scanner.Scan() {
					directive := scanner.Text()
					directive = strings.TrimSpace(directive)
					if directive == "" {
						continue
					}

					directiveParams := strings.Split(directive, ",")
					if len(directiveParams) != 3 {
						failf("Invalid directive configuration: %s", directive)
					}
					roboDirectives = append(roboDirectives, &testing.RoboDirective{ResourceName: directiveParams[0], InputText: directiveParams[1], ActionType: directiveParams[2]})
				}
				testModel.TestSpecification.AndroidRoboTest.RoboDirectives = roboDirectives
			}
			if configs.RoboScenarioFile != "" {
				testModel.TestSpecification.AndroidRoboTest.RoboScript = &testing.FileReference{
					GcsPath: testAssets.RoboScript.GcsPath,
				}
			}
		case "gameloop":
			testModel.TestSpecification.AndroidTestLoop = &testing.AndroidTestLoop{}

			if isBundle {
				testModel.TestSpecification.AndroidTestLoop.AppBundle = &testing.AppBundle{
					BundleLocation: &testing.FileReference{GcsPath: testApp.GcsPath},
				}
			} else {
				testModel.TestSpecification.AndroidTestLoop.AppApk = &testing.FileReference{GcsPath: testApp.GcsPath}
			}

			if configs.AppPackageID != "" {
				testModel.TestSpecification.AndroidTestLoop.AppPackageId = configs.AppPackageID
			}
			if configs.LoopScenarios != "" {
				loopScenarios := []int64{}
				for _, scenarioStr := range strings.Split(strings.TrimSpace(configs.LoopScenarios), ",") {
					scenario, err := strconv.Atoi(scenarioStr)
					if err != nil {
						failf("Failed to parse string(%s) to integer, error: %s", scenarioStr, err)
					}
					loopScenarios = append(loopScenarios, int64(scenario))
				}
				testModel.TestSpecification.AndroidTestLoop.Scenarios = loopScenarios
			}
			if configs.LoopScenarioLabels != "" {
				scenarioLabels := strings.Split(strings.TrimSpace(configs.LoopScenarioLabels), ",")
				testModel.TestSpecification.AndroidTestLoop.ScenarioLabels = scenarioLabels
			}
		}

		jsonByte, err := json.Marshal(testModel)
		if err != nil {
			failf("Failed to marshal test model, error: %s", err)
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonByte))
		if err != nil {
			failf("Failed to create http request, error: %s", err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			failf("Failed to get http response, error: %s", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				failf("Failed to read response body, error: %s", err)
			}
			failf("Failed to start test: %d, error: %s", resp.StatusCode, string(body))
		}

		log.Donef("=> Test started")
	}

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
			if resp.StatusCode != http.StatusOK || err != nil {
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

				for _, step := range responseModel.Steps {
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
			}
			if !finished {
				time.Sleep(5 * time.Second)
			}
		}
	}

	if configs.DownloadTestResults == "true" {
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
