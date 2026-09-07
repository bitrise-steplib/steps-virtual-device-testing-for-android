//go:build maintenance

// This test calls the live Firebase Test Lab catalog, so it needs the gcloud CLI and
// credentials. The build tag keeps it out of `go test ./...` in the check workflow; the
// e2e `test_device_catalog_up_to_date` workflow runs it with `-tags maintenance`.

package maintenance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
)

// deviceListPath holds the catalog this Step's device table was generated from. It is a file
// rather than a const in this test so that applying an update is a file write, not a test
// rewriting its own source.
const deviceListPath = "testdata/device_list.txt"

func TestDeviceList(t *testing.T) {
	signedIn, err := checkAccounts()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	if !signedIn {
		if err := signIn(); err != nil {
			t.Error(err)
			t.FailNow()
		}
	}

	if err := checkDeviceList(); err != nil {
		t.Error(err)
	}
}

func checkDeviceList() error {
	deviceList, err := fetchDeviceList()
	if err != nil {
		return err
	}

	expected, err := os.ReadFile(deviceListPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", deviceListPath, err)
	}

	if deviceList == strings.TrimRight(string(expected), "\n") {
		return nil
	}

	deviceTable, err := fetchDeviceTable()
	if err != nil {
		return err
	}

	fmt.Printf("Fresh device list to write to %s:\n", deviceListPath)
	fmt.Println(deviceList)
	fmt.Println()
	fmt.Println("Fresh device table to use in the step's descriptor:")
	fmt.Println(deviceTable)

	return fmt.Errorf("device list has changed, update %s and the step's device table",
		deviceListPath)
}

// gcloudStdout runs gcloud and returns its trimmed stdout.
//
// Only stdout, because this output is golden data: it is compared against deviceListPath, and
// the table is meant to be pasted into step.yml. gcloud writes component update notices and
// credential warnings to stderr, and combined output would mix them into both. stderr is still
// read, it carries gcloud's diagnostics when the call fails.
func gcloudStdout(args ...string) (string, error) {
	var stdout, stderr bytes.Buffer

	cmd := command.New("gcloud", args...).SetStdout(&stdout).SetStderr(&stderr)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w, stderr: %s",
			cmd.PrintableCommandArgs(), err, strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(stdout.String()), nil
}

func fetchDeviceList() (string, error) {
	return gcloudStdout("firebase", "test", "android", "models", "list",
		"--filter=VIRTUAL",
		"--format", "text")
}

func fetchDeviceTable() (string, error) {
	return gcloudStdout("firebase", "test", "android", "models", "list",
		"--filter=VIRTUAL",
		// Generally available models first, then newest OS first within each group, so the
		// models worth picking are at the top. Untagged means GA, and an empty tags[0]
		// sorts before "beta=*" and "preview=*". supportedVersionIds is ascending, so [-1]
		// is the newest OS. name breaks remaining ties, otherwise the row order is
		// arbitrary and the table churns between runs.
		"--sort-by", "tags[0],~supportedVersionIds[-1],name",
		"--format", deviceTableFormat)
}

func signIn() error {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("_serv_acc_")
	if err != nil {
		return err
	}

	servAccFileContent := os.Getenv("SERVICE_ACCOUNT_JSON")
	if servAccFileContent == "" {
		return fmt.Errorf("$SERVICE_ACCOUNT_JSON is not set")
	}

	servAccFilePAth := filepath.Join(tmpDir, "serv-acc.json")
	if err := fileutil.WriteStringToFile(servAccFilePAth, servAccFileContent); err != nil {
		return err
	}

	var servAcc struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.NewDecoder(strings.NewReader(servAccFileContent)).Decode(&servAcc); err != nil {
		return err
	}
	if servAcc.ProjectID == "" {
		return fmt.Errorf("invalid service account json, no project_id found")
	}

	cmd := command.New("gcloud",
		"auth",
		"activate-service-account",
		fmt.Sprintf("--key-file=%s", servAccFilePAth),
		"--project", servAcc.ProjectID)

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return fmt.Errorf("out: %s, err: %w", out, err)
	}

	return nil
}

func checkAccounts() (bool, error) {
	cmd := command.New("gcloud", "auth", "list", "--format", "json")
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return false, err
	}

	var accounts []interface{}
	if err := json.NewDecoder(strings.NewReader(out)).Decode(&accounts); err != nil {
		return false, err
	}

	return len(accounts) > 0, nil
}

// deviceTableFormat renders the catalog as a single box table, matching the table in the
// test_devices input of step.yml. It differs from the default `models list` format only in
// column order: what a user configuring the input needs first comes first. `form.color()`
// clears the default blue on the FORM value, otherwise the output carries ANSI escapes that
// would end up pasted into step.yml.
const deviceTableFormat = `table[box](
	name:label=MODEL_NAME,
	id:label=MODEL_ID,
	supportedVersionIds.list(undefined="none"):label=OS_VERSION_IDS,
	tags.list(separator=", "):label=TAGS,
	manufacturer:label=MAKE,
	format("{0:>4} x {1:<4}", screenY, screenX):label=RESOLUTION,
	form.color():label=FORM)`
