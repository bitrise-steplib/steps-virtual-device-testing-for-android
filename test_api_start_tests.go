package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/bitrise-io/go-utils/log"
	testing "google.golang.org/api/testing/v1"
)

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
