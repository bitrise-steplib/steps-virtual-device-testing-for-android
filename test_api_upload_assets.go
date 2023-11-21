package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bitrise-io/go-utils/log"
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
