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

	cmd = command.New("gcloud", "firebase", "test", "android", "models", "list", "--filter=VIRTUAL")
	outFormatted, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return fmt.Errorf("out: %s, err: %w", out, err)
	}

	fmt.Println("Fresh devices list to use in this integration test:")
	fmt.Println(out)
	fmt.Println()
	fmt.Println("Fresh device list to use in the step's descriptor:")
	fmt.Println(outFormatted)

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
supportedVersionIds[8]:                                          35
supportedVersionIds[9]:                                          36
supportedVersionIds[10]:                                         34
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
