package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

func testAssetsDownloadURL(base, appSlug, buildSlug, token string) string {
	return base + "/assets/" + appSlug + "/" + buildSlug + "/" + token
}

func downloadTestAssets(url string) (string, error) {
	fmt.Println()
	log.Infof("Downloading test assets")
	//url := configs.APIBaseURL + "/assets/" + configs.AppSlug + "/" + configs.BuildSlug + "/" + configs.APIToken

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %s", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get http response: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get http response, status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body, error: %s", err)
	}

	responseModel := map[string]string{}

	err = json.Unmarshal(body, &responseModel)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response body: %s", err)
	}

	tempDir, err := pathutil.NormalizedOSTempDirPath("vdtesting_test_assets")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %s", err)
	}

	for fileName, fileURL := range responseModel {
		err := downloadFile(fileURL, filepath.Join(tempDir, fileName))
		if err != nil {
			return "", fmt.Errorf("failed to download file: %s", err)
		}
	}

	return tempDir, nil
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
