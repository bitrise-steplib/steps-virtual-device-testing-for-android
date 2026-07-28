//go:build maintenance

// This test calls the live Firebase Test Lab catalog, so it needs the gcloud CLI and
// credentials. The build tag keeps it out of `go test ./...` in the check workflow; the
// e2e `test_device_catalog_up_to_date` workflow runs it with `-tags maintenance`.

package maintenance

import (
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
	cmd := command.New("gcloud", "firebase", "test", "android", "models", "list", "--format", "text", "--filter=VIRTUAL")
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return fmt.Errorf("out: %s, err: %w", out, err)
	}

	if out == deviceList {
		return nil
	}

	cmd = command.New("gcloud", "firebase", "test", "android", "models", "list",
		"--filter=VIRTUAL",
		// Generally available models first, then newest OS first within each group, so the
		// models worth picking are at the top. Untagged means GA, and an empty tags[0]
		// sorts before "beta=*" and "preview=*". supportedVersionIds is ascending, so [-1]
		// is the newest OS. name breaks remaining ties, otherwise the row order is
		// arbitrary and the table churns between runs.
		"--sort-by", "tags[0],~supportedVersionIds[-1],name",
		"--format", deviceTableFormat)

	deviceTable, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return fmt.Errorf("out: %s, err: %w", out, err)
	}

	fmt.Println("Fresh devices list to use in this integration test:")
	fmt.Println(out)
	fmt.Println()
	fmt.Println("Fresh device table to use in the step's descriptor:")
	fmt.Println(deviceTable)

	return fmt.Errorf("device list has changed, update the corresponding step descriptor blocks")
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

const deviceList = `---
brand:                  Google
codename:               AmatiTvEmulator
form:                   VIRTUAL
formFactor:             TV
id:                     AmatiTvEmulator
manufacturer:           Google
name:                   Google TV Amati
screenDensity:          320
screenX:                1920
screenY:                1080
supportedAbis[0]:       x86
supportedVersionIds[0]: 29
tags[0]:                beta=29
tags[1]:                deprecated=29
---
brand:                  Generic
codename:               AndroidTablet270dpi.arm
form:                   VIRTUAL
formFactor:             TABLET
id:                     AndroidTablet270dpi.arm
manufacturer:           Generic
name:                   Generic 720x1600 Android tablet @ 270dpi (Arm)
screenDensity:          270
screenX:                720
screenY:                1600
supportedAbis[0]:       arm64-v8a
supportedVersionIds[0]: 30
---
brand:                  Google
codename:               GoogleTvEmulator
form:                   VIRTUAL
formFactor:             TV
id:                     GoogleTvEmulator
manufacturer:           Google
name:                   Google TV
screenDensity:          213
screenX:                1280
screenY:                720
supportedAbis[0]:       x86
supportedVersionIds[0]: 30
tags[0]:                beta=30
tags[1]:                deprecated=30
---
brand:                                                           Generic
codename:                                                        MediumPhone.arm
form:                                                            VIRTUAL
formFactor:                                                      PHONE
id:                                                              MediumPhone.arm
manufacturer:                                                    Generic
name:                                                            Medium Phone, 6.4in/16cm (Arm)
perVersionInfo[0].deviceCapacity:                                DEVICE_CAPACITY_HIGH
perVersionInfo[0].directAccessVersionInfo.directAccessSupported: True
perVersionInfo[0].versionId:                                     34
perVersionInfo[1].deviceCapacity:                                DEVICE_CAPACITY_HIGH
perVersionInfo[1].directAccessVersionInfo.directAccessSupported: True
perVersionInfo[1].versionId:                                     35
perVersionInfo[2].deviceCapacity:                                DEVICE_CAPACITY_HIGH
perVersionInfo[2].directAccessVersionInfo.directAccessSupported: True
perVersionInfo[2].versionId:                                     36
screenDensity:                                                   420
screenX:                                                         1080
screenY:                                                         2400
supportedAbis[0]:                                                arm64-v8a
supportedVersionIds[0]:                                          26
supportedVersionIds[1]:                                          27
supportedVersionIds[2]:                                          28
supportedVersionIds[3]:                                          29
supportedVersionIds[4]:                                          30
supportedVersionIds[5]:                                          31
supportedVersionIds[6]:                                          32
supportedVersionIds[7]:                                          33
supportedVersionIds[8]:                                          34
supportedVersionIds[9]:                                          35
supportedVersionIds[10]:                                         36
---
brand:                  Google
codename:               MediumPhone_ps16k.arm
form:                   VIRTUAL
formFactor:             PHONE
id:                     MediumPhone_ps16k.arm
manufacturer:           Generic
name:                   Medium Phone (16K page size), 6.4in/16cm (Arm)
screenDensity:          420
screenX:                1080
screenY:                2400
supportedAbis[0]:       arm64-v8a
supportedVersionIds[0]: 36
supportedVersionIds[1]: 37
tags[0]:                preview=36
tags[1]:                preview=37
---
brand:                  Google
codename:               MediumPhone_ps16k_backcompat.arm
form:                   VIRTUAL
formFactor:             PHONE
id:                     MediumPhone_ps16k_backcompat.arm
manufacturer:           Generic
name:                   Medium Phone (16K page size), 6.4in/16cm (Arm)
screenDensity:          420
screenX:                1080
screenY:                2400
supportedAbis[0]:       arm64-v8a
supportedVersionIds[0]: 36
tags[0]:                preview=36
---
brand:                  Generic
codename:               MediumTablet.arm
form:                   VIRTUAL
formFactor:             TABLET
id:                     MediumTablet.arm
manufacturer:           Generic
name:                   Medium Tablet, 10.05in/25cm (Arm)
screenDensity:          320
screenX:                1600
screenY:                2560
supportedAbis[0]:       arm64-v8a
supportedVersionIds[0]: 26
supportedVersionIds[1]: 27
supportedVersionIds[2]: 28
supportedVersionIds[3]: 29
supportedVersionIds[4]: 30
supportedVersionIds[5]: 31
supportedVersionIds[6]: 32
supportedVersionIds[7]: 33
supportedVersionIds[8]: 34
supportedVersionIds[9]: 35
---
brand:                  Google
codename:               Pixel2.arm
form:                   VIRTUAL
formFactor:             PHONE
id:                     Pixel2.arm
manufacturer:           Google
name:                   Pixel 2 (Arm)
screenDensity:          420
screenX:                1080
screenY:                1920
supportedAbis[0]:       arm64-v8a
supportedVersionIds[0]: 26
supportedVersionIds[1]: 27
supportedVersionIds[2]: 28
supportedVersionIds[3]: 29
supportedVersionIds[4]: 30
supportedVersionIds[5]: 31
supportedVersionIds[6]: 32
supportedVersionIds[7]: 33
---
brand:                  Generic
codename:               SmallPhone.arm
form:                   VIRTUAL
formFactor:             PHONE
id:                     SmallPhone.arm
manufacturer:           Generic
name:                   Small Phone, 4.65in/12cm (Arm)
screenDensity:          320
screenX:                720
screenY:                1280
supportedAbis[0]:       arm64-v8a
supportedVersionIds[0]: 26
supportedVersionIds[1]: 27
supportedVersionIds[2]: 28
supportedVersionIds[3]: 29
supportedVersionIds[4]: 30
supportedVersionIds[5]: 31
supportedVersionIds[6]: 32
supportedVersionIds[7]: 33
supportedVersionIds[8]: 34
supportedVersionIds[9]: 35`
