package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	testing "google.golang.org/api/testing/v1"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

// This embedded basic application was created based on the https://github.com/tothszabi/BasicApp repository.
//
//go:embed resources/simple-fallback-bitrise-app.apk
var emptyAndroidApp []byte

// ConfigsModel ...
type ConfigsModel struct {
	// api
	APIBaseURL string `env:"api_base_url"`
	BuildSlug  string `env:"BITRISE_BUILD_SLUG,required"`
	AppSlug    string `env:"BITRISE_APP_SLUG,required"`
	APIToken   string `env:"api_token,required"`

	// shared
	AppPath         string `env:"app_path"`
	TestApkPath     string `env:"test_apk_path"`
	TestType        string `env:"test_type,opt[instrumentation,robo,gameloop]"`
	TestDevicesList string `env:"test_devices"`
	TestDevices     []*testing.AndroidDevice
	AppPackageID    string `env:"app_package_id"`

	// test setup
	AutoGoogleLogin          bool   `env:"auto_google_login,opt[true,false]"`
	EnvironmentVariablesList string `env:"environment_variables"`
	EnvironmentVariables     []*testing.EnvironmentVariable
	ObbFilesList             string `env:"obb_files_list"`
	ObbFiles                 []string

	// shared debug
	TestTimeout           float64 `env:"test_timeout,range]0..3600]"`
	FlakyTestAttempts     int     `env:"num_flaky_test_attempts,range[0..10]"`
	DownloadTestResults   bool    `env:"download_test_results,opt[true,false]"`
	DirectoriesToPullList string  `env:"directories_to_pull"`
	DirectoriesToPull     []string
	VerboseLog            bool `env:"use_verbose_log,opt[true,false]"`

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

func (configs *ConfigsModel) print() {
	log.Infof("Configs:")
	log.Printf("- AppPath: %s", configs.AppPath)
	if configs.ApkPath != "" {
		log.Printf("- ApkPath: %s", configs.ApkPath)
	}
	if configs.AppPackageID != "" {
		log.Printf("- AppPackageID: %s", configs.AppPackageID)
	}
	log.Printf("- TestTimeout: %f", configs.TestTimeout)
	log.Printf("- FlakyTestAttempts: %d", configs.FlakyTestAttempts)
	log.Printf("- DownloadTestResults: %t", configs.DownloadTestResults)
	log.Printf("- DirectoriesToPull: %s", configs.DirectoriesToPullList)
	log.Printf("- AutoGoogleLogin: %t", configs.AutoGoogleLogin)
	log.Printf("- EnvironmentVariables: %s", configs.EnvironmentVariablesList)
	log.Printf("- ObbFilesList: %s", configs.ObbFilesList)

	log.Printf("- TestDevices:\n---")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if _, err := fmt.Fprintln(w, "Model\tAPI Level\tLocale\tOrientation\t"); err != nil {
		failf("Failed to write in tabwriter, error: %s", err)
	}
	for _, testDevice := range configs.TestDevices {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", testDevice.AndroidModelId, testDevice.AndroidVersionId, testDevice.Locale, testDevice.Orientation); err != nil {
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
		log.Printf("- UseOrchestrator: %t", configs.UseOrchestrator)
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

func (configs *ConfigsModel) validate() error {
	configs.migrate()

	if strings.TrimSpace(configs.APIBaseURL) == "" {
		if _, set := os.LookupEnv("BITRISE_IO"); !set {
			log.Warnf("Warning: please make sure that Virtual Device Testing add-on is turned on under your app's settings tab.")
		}
		return fmt.Errorf("- APIBaseURL: required variable is not present")
	}

	if strings.TrimSpace(configs.AppPath) == "" {
		log.Warnf("Warning: Using embedded Android application as AppPath value is empty")

		path, err := pathutil.NormalizedOSTempDirPath("")
		if err != nil {
			return fmt.Errorf("- AppPath: failed to create temporary directory for embedded application")
		}
		appPath := filepath.Join(path, "app.apk")
		if err = fileutil.WriteBytesToFile(appPath, emptyAndroidApp); err != nil {
			return fmt.Errorf("- AppPath: failed to write embedded application to the temporary directory")
		}

		configs.AppPath = appPath
	}

	if configs.TestType == testTypeInstrumentation {
		if strings.TrimSpace(configs.TestApkPath) == "" {
			return fmt.Errorf("- TestApkPath: required variable is not present. Is it possible that you used gradle-runner step and forgot to set `assembleDebugAndroidTest` task?")
		}
		if _, err := os.Stat(configs.TestApkPath); err != nil {
			return fmt.Errorf("- TestApkPath: failed to get file info, error: %s. Is it possible that you used gradle-runner step and forgot to set `assembleDebugAndroidTest` task?", err)
		}
	}

	configs.RoboScenarioFile = strings.TrimSpace(configs.RoboScenarioFile)
	if configs.TestType == testTypeRobo && configs.RoboScenarioFile != "" {
		if _, err := os.Stat(configs.RoboScenarioFile); err != nil {
			return fmt.Errorf("- RoboScenarioFile: failed to get file info, error: %s", err)
		}
	}

	var err error
	if configs.TestDevices, err = parseDeviceList(configs.TestDevicesList); err != nil {
		return fmt.Errorf("- TestDevices: %s", err)
	}

	if configs.ObbFiles, err = parseObbFilesList(configs.ObbFilesList); err != nil {
		return fmt.Errorf("- ObbFiles: %s", err)
	}

	configs.DirectoriesToPull = parseDirectoriesToPull(configs.DirectoriesToPullList)
	configs.EnvironmentVariables = parseTestSetupEnvVars(configs.EnvironmentVariablesList)

	return nil
}

func (configs *ConfigsModel) migrate() {
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
}

func parseDeviceList(deviceList string) ([]*testing.AndroidDevice, error) {
	var testDevices []*testing.AndroidDevice

	scanner := bufio.NewScanner(strings.NewReader(deviceList))
	for scanner.Scan() {
		device := scanner.Text()
		device = strings.TrimSpace(device)
		if device == "" {
			continue
		}

		deviceParams := strings.Split(device, ",")
		if len(deviceParams) != 4 {
			return nil, fmt.Errorf("invalid test device configuration: %s", device)
		}

		testDevices = append(testDevices, &testing.AndroidDevice{
			AndroidModelId:   deviceParams[0],
			AndroidVersionId: deviceParams[1],
			Locale:           deviceParams[2],
			Orientation:      deviceParams[3],
		})
	}

	return testDevices, nil
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

func parseDirectoriesToPull(directoriesToPullList string) []string {
	scanner := bufio.NewScanner(strings.NewReader(directoriesToPullList))
	directoriesToPull := []string{}
	for scanner.Scan() {
		path := scanner.Text()
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		directoriesToPull = append(directoriesToPull, path)
	}

	return directoriesToPull
}

func parseTestSetupEnvVars(envVarList string) []*testing.EnvironmentVariable {
	scanner := bufio.NewScanner(strings.NewReader(envVarList))
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

	return envs
}
