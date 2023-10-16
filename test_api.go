package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bitrise-io/go-utils/log"
	testing "google.golang.org/api/testing/v1"
)

// TestAsset describes a requested test asset
type TestAsset struct {
	UploadURL string `json:"uploadUrl"`
	GcsPath   string `json:"gcsPath"`
	Filename  string `json:"filename"`
}

// TestAssetsAndroid describes requested Android test asset and as the returned test asset upload URLs
type TestAssetsAndroid struct {
	isBundle   bool
	testApp    *TestAsset
	Apk        TestAsset   `json:"apk,omitempty"`
	Aab        TestAsset   `json:"aab,omitmepty"`
	TestApk    TestAsset   `json:"testApk,omitempty"`
	RoboScript TestAsset   `json:"roboScript,omitempty"`
	ObbFiles   []TestAsset `json:"obbFiles,omitempty"`
}

func uploadTestAssets(configs ConfigsModel) (TestAssetsAndroid, error) {
	var testAssets TestAssetsAndroid

	url := configs.APIBaseURL + "/assets/android/" + configs.AppSlug + "/" + configs.BuildSlug + "/" + configs.APIToken

	if strings.ToLower(filepath.Ext(configs.AppPath)) == ".aab" {
		testAssets.isBundle = true
	}
	log.Debugf("App path (%s), is bundle: %t", configs.AppPath, testAssets.isBundle)

	var requestedAssets TestAssetsAndroid
	if testAssets.isBundle {
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
		return TestAssetsAndroid{}, fmt.Errorf("failed to encode to json: %+v", requestedAssets)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return TestAssetsAndroid{}, fmt.Errorf("failed to create http request, error: %s", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return TestAssetsAndroid{}, fmt.Errorf("failed to get http response, error: %s", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return TestAssetsAndroid{}, fmt.Errorf("failed to read response body (status code: %d), error: %s", resp.StatusCode, err)
	}

	if resp.StatusCode != http.StatusOK {
		return TestAssetsAndroid{}, fmt.Errorf("failed to start test: %d, error: %s", resp.StatusCode, string(body))
	}

	err = json.Unmarshal(body, &testAssets)
	if err != nil {
		return TestAssetsAndroid{}, fmt.Errorf("failed to unmarshal response body, error: %s", err)
	}

	if testAssets.isBundle {
		testAssets.testApp = &testAssets.Aab
	} else {
		testAssets.testApp = &testAssets.Apk
	}
	log.Debugf("Uploading file(%s) to (%s)", configs.AppPath, testAssets.testApp.GcsPath)

	err = uploadFile(testAssets.testApp.UploadURL, configs.AppPath)
	if err != nil {
		return TestAssetsAndroid{}, fmt.Errorf("failed to upload file(%s) to (%s), error: %s", configs.AppPath, testAssets.testApp.UploadURL, err)
	}

	if configs.TestType == testTypeInstrumentation {
		if err := uploadFile(testAssets.TestApk.UploadURL, configs.TestApkPath); err != nil {
			return TestAssetsAndroid{}, fmt.Errorf("failed to upload file(%s) to (%s), error: %s", configs.TestApkPath, testAssets.TestApk.UploadURL, err)
		}
	}

	if configs.TestType == testTypeRobo && configs.RoboScenarioFile != "" {
		if err := uploadFile(testAssets.RoboScript.UploadURL, configs.RoboScenarioFile); err != nil {
			return TestAssetsAndroid{}, fmt.Errorf("failed to upload file(%s) to (%s), error: %s", configs.RoboScenarioFile, testAssets.RoboScript.UploadURL, err)
		}
	}

	if len(testAssets.ObbFiles) != len(configs.ObbFiles) {
		return TestAssetsAndroid{}, fmt.Errorf("invalid length of obb file upload URLs in response: %+v", testAssets)
	}
	for i, obbFile := range configs.ObbFiles {
		if err := uploadFile(testAssets.ObbFiles[i].UploadURL, obbFile); err != nil {
			return TestAssetsAndroid{}, fmt.Errorf("failed to upload obb file (%s) to (%s), error: %s", obbFile, testAssets.ObbFiles[i].UploadURL, err)
		}
	}

	return testAssets, nil
}

func startTestRun(configs ConfigsModel, testAssets TestAssetsAndroid) error {
	url := configs.APIBaseURL + "/" + configs.AppSlug + "/" + configs.BuildSlug + "/" + configs.APIToken

	testModel := &testing.TestMatrix{}
	testModel.EnvironmentMatrix = &testing.EnvironmentMatrix{AndroidDeviceList: &testing.AndroidDeviceList{}}

	testModel.EnvironmentMatrix.AndroidDeviceList.AndroidDevices = configs.TestDevices
	testModel.FlakyTestAttempts = int64(configs.FlakyTestAttempts)

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
			EnvironmentVariables: configs.EnvironmentVariables,
			FilesToPush:          filesToPush,
			DirectoriesToPull:    configs.DirectoriesToPull,
			Account:              account,
		},
	}

	switch configs.TestType {
	case testTypeInstrumentation:
		testModel.TestSpecification.AndroidInstrumentationTest = &testing.AndroidInstrumentationTest{}

		if testAssets.isBundle {
			testModel.TestSpecification.AndroidInstrumentationTest.AppBundle = &testing.AppBundle{
				BundleLocation: &testing.FileReference{GcsPath: testAssets.testApp.GcsPath},
			}
		} else {
			testModel.TestSpecification.AndroidInstrumentationTest.AppApk = &testing.FileReference{GcsPath: testAssets.testApp.GcsPath}
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

		if int64(configs.NumberOfUniformShards) > 0 {
			testModel.TestSpecification.AndroidInstrumentationTest.ShardingOption = &testing.ShardingOption{}
			testModel.TestSpecification.AndroidInstrumentationTest.ShardingOption.UniformSharding = &testing.UniformSharding{}
			testModel.TestSpecification.AndroidInstrumentationTest.ShardingOption.UniformSharding.NumShards = int64(configs.NumberOfUniformShards)
		}
		log.Debugf("AndroidInstrumentationTest: %+v", testModel.TestSpecification.AndroidInstrumentationTest)
	case testTypeRobo:
		testModel.TestSpecification.AndroidRoboTest = &testing.AndroidRoboTest{}

		if testAssets.isBundle {
			testModel.TestSpecification.AndroidRoboTest.AppBundle = &testing.AppBundle{
				BundleLocation: &testing.FileReference{GcsPath: testAssets.testApp.GcsPath},
			}
		} else {
			testModel.TestSpecification.AndroidRoboTest.AppApk = &testing.FileReference{GcsPath: testAssets.testApp.GcsPath}
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
				return fmt.Errorf("failed to parse string(%s) to integer, error: %s", configs.RoboMaxDepth, err)
			}
			testModel.TestSpecification.AndroidRoboTest.MaxDepth = int64(maxDepth)
		}
		if configs.RoboMaxSteps != "" {
			maxSteps, err := strconv.Atoi(configs.RoboMaxSteps)
			if err != nil {
				return fmt.Errorf("failed to parse string(%s) to integer, error: %s", configs.RoboMaxSteps, err)
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
					return fmt.Errorf("invalid directive configuration: %s", directive)
				}
				roboDirectives = append(roboDirectives, &testing.RoboDirective{ResourceName: directiveParams[0], InputText: directiveParams[1], ActionType: directiveParams[2]})
			}
			testModel.TestSpecification.AndroidRoboTest.RoboDirectives = roboDirectives
		}
		if configs.RoboScenarioFile != "" {
			log.Debugf("Robo scenario file: %s", testAssets.RoboScript.GcsPath)
			testModel.TestSpecification.AndroidRoboTest.RoboScript = &testing.FileReference{
				GcsPath: testAssets.RoboScript.GcsPath,
			}
		}
	case "gameloop":
		testModel.TestSpecification.AndroidTestLoop = &testing.AndroidTestLoop{}

		if testAssets.isBundle {
			testModel.TestSpecification.AndroidTestLoop.AppBundle = &testing.AppBundle{
				BundleLocation: &testing.FileReference{GcsPath: testAssets.testApp.GcsPath},
			}
		} else {
			testModel.TestSpecification.AndroidTestLoop.AppApk = &testing.FileReference{GcsPath: testAssets.testApp.GcsPath}
		}

		if configs.AppPackageID != "" {
			testModel.TestSpecification.AndroidTestLoop.AppPackageId = configs.AppPackageID
		}
		if configs.LoopScenarios != "" {
			loopScenarios := []int64{}
			for _, scenarioStr := range strings.Split(strings.TrimSpace(configs.LoopScenarios), ",") {
				scenario, err := strconv.Atoi(scenarioStr)
				if err != nil {
					return fmt.Errorf("failed to parse string(%s) to integer, error: %s", scenarioStr, err)
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
		return fmt.Errorf("failed to marshal test model, error: %s", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonByte))
	if err != nil {
		return fmt.Errorf("failed to create http request, error: %s", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get http response, error: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body, error: %s", err)
		}
		return fmt.Errorf("failed to start test: %d, error: %s", resp.StatusCode, string(body))
	}

	return nil
}
