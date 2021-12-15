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
	"github.com/pkg/errors"
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
		return errors.Wrap(err, out)
	}

	if out == deviceList {
		return nil
	}

	cmd = command.New("gcloud", "firebase", "test", "android", "models", "list", "--filter=VIRTUAL")
	outFormatted, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return errors.Wrap(err, out)
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
	return errors.Wrap(err, out)
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
codename:               Nexus10
form:                   VIRTUAL
formFactor:             TABLET
id:                     Nexus10
manufacturer:           Samsung
name:                   Nexus 10
screenDensity:          320
screenX:                1600
screenY:                2560
supportedAbis[0]:       x86
supportedVersionIds[0]: 19
supportedVersionIds[1]: 21
supportedVersionIds[2]: 22
---
brand:                  Google
codename:               Nexus4
form:                   VIRTUAL
formFactor:             PHONE
id:                     Nexus4
manufacturer:           LG
name:                   Nexus 4
screenDensity:          320
screenX:                768
screenY:                1280
supportedAbis[0]:       x86
supportedVersionIds[0]: 19
supportedVersionIds[1]: 21
supportedVersionIds[2]: 22
---
brand:                  Google
codename:               Nexus5
form:                   VIRTUAL
formFactor:             PHONE
id:                     Nexus5
manufacturer:           LG
name:                   Nexus 5
screenDensity:          480
screenX:                1080
screenY:                1920
supportedAbis[0]:       x86
supportedAbis[1]:       23:armeabi
supportedAbis[2]:       23:armeabi-v7a
supportedVersionIds[0]: 19
supportedVersionIds[1]: 21
supportedVersionIds[2]: 22
supportedVersionIds[3]: 23
---
brand:                  Google
codename:               Nexus5X
form:                   VIRTUAL
formFactor:             PHONE
id:                     Nexus5X
manufacturer:           LG
name:                   Nexus 5X
screenDensity:          420
screenX:                1080
screenY:                1920
supportedAbis[0]:       x86
supportedAbis[1]:       23:armeabi
supportedAbis[2]:       23:armeabi-v7a
supportedAbis[3]:       24:armeabi
supportedAbis[4]:       24:armeabi-v7a
supportedAbis[5]:       25:armeabi
supportedAbis[6]:       25:armeabi-v7a
supportedAbis[7]:       26:armeabi
supportedAbis[8]:       26:armeabi-v7a
supportedVersionIds[0]: 23
supportedVersionIds[1]: 24
supportedVersionIds[2]: 25
supportedVersionIds[3]: 26
---
brand:                  Google
codename:               Nexus6
form:                   VIRTUAL
formFactor:             PHONE
id:                     Nexus6
manufacturer:           Motorola
name:                   Nexus 6
screenDensity:          560
screenX:                1440
screenY:                2560
supportedAbis[0]:       x86
supportedAbis[1]:       23:armeabi
supportedAbis[2]:       23:armeabi-v7a
supportedAbis[3]:       24:armeabi
supportedAbis[4]:       24:armeabi-v7a
supportedAbis[5]:       25:armeabi
supportedAbis[6]:       25:armeabi-v7a
supportedVersionIds[0]: 21
supportedVersionIds[1]: 22
supportedVersionIds[2]: 23
supportedVersionIds[3]: 24
supportedVersionIds[4]: 25
---
brand:                  Google
codename:               Nexus6P
form:                   VIRTUAL
formFactor:             PHONE
id:                     Nexus6P
manufacturer:           Google
name:                   Nexus 6P
screenDensity:          560
screenX:                1440
screenY:                2560
supportedAbis[0]:       x86
supportedAbis[1]:       23:armeabi
supportedAbis[2]:       23:armeabi-v7a
supportedAbis[3]:       24:armeabi
supportedAbis[4]:       24:armeabi-v7a
supportedAbis[5]:       25:armeabi
supportedAbis[6]:       25:armeabi-v7a
supportedAbis[7]:       26:armeabi
supportedAbis[8]:       26:armeabi-v7a
supportedAbis[9]:       27:armeabi
supportedAbis[10]:      27:armeabi-v7a
supportedVersionIds[0]: 23
supportedVersionIds[1]: 24
supportedVersionIds[2]: 25
supportedVersionIds[3]: 26
supportedVersionIds[4]: 27
---
brand:                  Google
codename:               Nexus7
form:                   VIRTUAL
formFactor:             TABLET
id:                     Nexus7
manufacturer:           Asus
name:                   Nexus 7 (2012)
screenDensity:          213
screenX:                800
screenY:                1280
supportedAbis[0]:       x86
supportedVersionIds[0]: 19
supportedVersionIds[1]: 21
supportedVersionIds[2]: 22
---
brand:                  Generic
codename:               Nexus7_clone_16_9
form:                   VIRTUAL
formFactor:             TABLET
id:                     Nexus7_clone_16_9
manufacturer:           Generic
name:                   Nexus7 clone, DVD 16:9 aspect ratio
screenDensity:          160
screenX:                720
screenY:                1280
supportedAbis[0]:       x86
supportedAbis[1]:       23:armeabi
supportedAbis[2]:       23:armeabi-v7a
supportedAbis[3]:       24:armeabi
supportedAbis[4]:       24:armeabi-v7a
supportedAbis[5]:       25:armeabi
supportedAbis[6]:       25:armeabi-v7a
supportedAbis[7]:       26:armeabi
supportedAbis[8]:       26:armeabi-v7a
supportedVersionIds[0]: 23
supportedVersionIds[1]: 24
supportedVersionIds[2]: 25
supportedVersionIds[3]: 26
tags[0]:                beta
---
brand:                  Google
codename:               Nexus9
form:                   VIRTUAL
formFactor:             TABLET
id:                     Nexus9
manufacturer:           HTC
name:                   Nexus 9
screenDensity:          320
screenX:                1536
screenY:                2048
supportedAbis[0]:       x86
supportedAbis[1]:       23:armeabi
supportedAbis[2]:       23:armeabi-v7a
supportedAbis[3]:       24:armeabi
supportedAbis[4]:       24:armeabi-v7a
supportedAbis[5]:       25:armeabi
supportedAbis[6]:       25:armeabi-v7a
supportedVersionIds[0]: 21
supportedVersionIds[1]: 22
supportedVersionIds[2]: 23
supportedVersionIds[3]: 24
supportedVersionIds[4]: 25
---
brand:                  Generic
codename:               NexusLowRes
form:                   VIRTUAL
formFactor:             PHONE
id:                     NexusLowRes
manufacturer:           Generic
name:                   Low-resolution MDPI phone
screenDensity:          160
screenX:                360
screenY:                640
supportedAbis[0]:       23:armeabi
supportedAbis[1]:       23:armeabi-v7a
supportedAbis[2]:       23:x86
supportedAbis[3]:       24:armeabi
supportedAbis[4]:       24:armeabi-v7a
supportedAbis[5]:       24:x86
supportedAbis[6]:       25:armeabi
supportedAbis[7]:       25:armeabi-v7a
supportedAbis[8]:       25:x86
supportedAbis[9]:       26:armeabi
supportedAbis[10]:      26:armeabi-v7a
supportedAbis[11]:      26:x86
supportedAbis[12]:      27:armeabi
supportedAbis[13]:      27:armeabi-v7a
supportedAbis[14]:      27:x86
supportedAbis[15]:      28:armeabi
supportedAbis[16]:      28:armeabi-v7a
supportedAbis[17]:      28:x86
supportedAbis[18]:      29:armeabi
supportedAbis[19]:      29:armeabi-v7a
supportedAbis[20]:      29:x86
supportedAbis[21]:      30:x86
supportedAbis[22]:      31:x86_64
supportedVersionIds[0]: 23
supportedVersionIds[1]: 24
supportedVersionIds[2]: 25
supportedVersionIds[3]: 26
supportedVersionIds[4]: 27
supportedVersionIds[5]: 28
supportedVersionIds[6]: 29
supportedVersionIds[7]: 30
tags[0]:                beta=30
---
brand:                  Google
codename:               Pixel2
form:                   VIRTUAL
formFactor:             PHONE
id:                     Pixel2
manufacturer:           Google
name:                   Pixel 2
screenDensity:          441
screenX:                1080
screenY:                1920
supportedAbis[0]:       26:armeabi
supportedAbis[1]:       26:armeabi-v7a
supportedAbis[2]:       26:x86
supportedAbis[3]:       27:armeabi
supportedAbis[4]:       27:armeabi-v7a
supportedAbis[5]:       27:x86
supportedAbis[6]:       28:armeabi
supportedAbis[7]:       28:armeabi-v7a
supportedAbis[8]:       28:x86
supportedAbis[9]:       29:armeabi
supportedAbis[10]:      29:armeabi-v7a
supportedAbis[11]:      29:x86
supportedAbis[12]:      30:x86
supportedAbis[13]:      31:x86_64
supportedVersionIds[0]: 26
supportedVersionIds[1]: 27
supportedVersionIds[2]: 28
supportedVersionIds[3]: 29
supportedVersionIds[4]: 30
tags[0]:                beta=30
---
brand:                  google
codename:               Pixel3
form:                   VIRTUAL
formFactor:             PHONE
id:                     Pixel3
manufacturer:           Google
name:                   Pixel 3
screenDensity:          440
screenX:                1080
screenY:                2160
supportedAbis[0]:       30:x86
supportedVersionIds[0]: 30
tags[0]:                beta=30`
