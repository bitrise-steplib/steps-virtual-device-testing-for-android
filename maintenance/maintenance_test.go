// This test calls the live Firebase Test Lab catalog, so it needs the gcloud CLI and
// credentials. It is its own module (see go.mod), which keeps it out of the Step's
// `go test ./...` and its dependencies out of the Step's vendor/.

package maintenance

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"gopkg.in/yaml.v3"
)

// With -update the test writes the fresh catalog into the golden file and step.yml instead of
// failing, which is what the `maintenance` workflow does. Without it the test only reports.
var update = flag.Bool("update", false, "rewrite testdata/device_list.txt and step.yml from the live catalog")

const (
	deviceListPath = "testdata/device_list.txt"
	stepYMLPath    = "../step.yml"

	testDevicesInput  = "test_devices"
	tableHeaderPrefix = "Available devices"
	generatedOn       = "(generated on "
	codeFence         = "```"
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

	if !*update {
		fmt.Println("Fresh device list to use in this maintenance test:")
		fmt.Println(deviceList)
		fmt.Println()
		fmt.Println("Fresh device table to use in the step's descriptor:")
		fmt.Println(deviceTable)

		return fmt.Errorf("device list has changed, re-run with -update to apply it to %s and %s",
			deviceListPath, stepYMLPath)
	}

	if err := os.WriteFile(deviceListPath, []byte(deviceList+"\n"), 0600); err != nil {
		return fmt.Errorf("write %s: %w", deviceListPath, err)
	}
	if err := updateStepYML(deviceTable); err != nil {
		return err
	}

	fmt.Printf("Device list has changed, updated %s and %s.\n", deviceListPath, stepYMLPath)
	fmt.Println("Regenerate README.md and commit both.")

	return nil
}

func fetchDeviceList() (string, error) {
	cmd := command.New("gcloud", "firebase", "test", "android", "models", "list",
		"--filter=VIRTUAL",
		"--format", "text")

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return "", fmt.Errorf("out: %s, err: %w", out, err)
	}

	return out, nil
}

func fetchDeviceTable() (string, error) {
	cmd := command.New("gcloud", "firebase", "test", "android", "models", "list",
		"--filter=VIRTUAL",
		// Generally available models first, then newest OS first within each group, so the
		// models worth picking are at the top. Untagged means GA, and an empty tags[0]
		// sorts before "beta=*" and "preview=*". supportedVersionIds is ascending, so [-1]
		// is the newest OS. name breaks remaining ties, otherwise the row order is
		// arbitrary and the table churns between runs.
		"--sort-by", "tags[0],~supportedVersionIds[-1],name",
		"--format", deviceTableFormat)

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return "", fmt.Errorf("out: %s, err: %w", out, err)
	}

	return out, nil
}

// yamlMapValue returns the value node for key in a YAML mapping node.
func yamlMapValue(mapping *yaml.Node, key string) (*yaml.Node, error) {
	if mapping.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected a mapping, got kind %d", mapping.Kind)
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1], nil
		}
	}

	return nil, fmt.Errorf("no %s key", key)
}

// testDevicesDescription returns the description node of the test_devices input. Each item of
// the inputs sequence is a mapping whose first key is the input's name.
func testDevicesDescription(doc *yaml.Node) (*yaml.Node, error) {
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("empty document")
	}

	inputs, err := yamlMapValue(doc.Content[0], "inputs")
	if err != nil {
		return nil, err
	}

	for _, input := range inputs.Content {
		if len(input.Content) == 0 || input.Content[0].Value != testDevicesInput {
			continue
		}

		opts, err := yamlMapValue(input, "opts")
		if err != nil {
			return nil, fmt.Errorf("%s input: %w", testDevicesInput, err)
		}

		description, err := yamlMapValue(opts, "description")
		if err != nil {
			return nil, fmt.Errorf("%s input: %w", testDevicesInput, err)
		}

		return description, nil
	}

	return nil, fmt.Errorf("no %s input", testDevicesInput)
}

// replaceTableBlock swaps the fenced device table in the input description for a fresh one and
// stamps today's date on the line introducing it.
func replaceTableBlock(description, deviceTable string) (string, error) {
	lines := strings.Split(description, "\n")

	header := -1
	for i, line := range lines {
		if strings.HasPrefix(line, tableHeaderPrefix) && strings.Contains(line, generatedOn) {
			header = i
			break
		}
	}
	if header == -1 {
		return "", fmt.Errorf("no %q line introducing the device table", tableHeaderPrefix)
	}
	if header+1 >= len(lines) || lines[header+1] != codeFence {
		return "", fmt.Errorf("the %q line is not followed by %s", tableHeaderPrefix, codeFence)
	}

	end := -1
	for i := header + 2; i < len(lines); i++ {
		if lines[i] == codeFence {
			end = i
			break
		}
	}
	if end == -1 {
		return "", fmt.Errorf("the device table's %s is not closed", codeFence)
	}

	prefix, _, _ := strings.Cut(lines[header], generatedOn)
	updated := append([]string{}, lines[:header]...)
	updated = append(updated, prefix+generatedOn+time.Now().Format(time.DateOnly)+"):", codeFence)
	updated = append(updated, strings.Split(deviceTable, "\n")...)
	updated = append(updated, lines[end:]...)

	return strings.Join(updated, "\n"), nil
}

// updateStepYML rewrites the device table in the test_devices input description. The description
// is located by parsing step.yml, but only its lines are rewritten: re-encoding the whole
// document would reformat every other block in the file.
func updateStepYML(deviceTable string) error {
	content, err := os.ReadFile(stepYMLPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", stepYMLPath, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return fmt.Errorf("parse %s: %w", stepYMLPath, err)
	}

	description, err := testDevicesDescription(&doc)
	if err != nil {
		return fmt.Errorf("parse %s: %w", stepYMLPath, err)
	}
	if description.Style != yaml.LiteralStyle {
		return fmt.Errorf("%s: expected the %s description to be a literal block scalar", stepYMLPath, testDevicesInput)
	}

	updated, err := replaceTableBlock(description.Value, deviceTable)
	if err != nil {
		return fmt.Errorf("%s: %w", stepYMLPath, err)
	}

	// Node.Line points at the `description: |` line, so the block scalar's content starts on
	// the next one and runs while lines are blank or indented at least as far as the first.
	lines := strings.Split(string(content), "\n")
	first := description.Line
	if first >= len(lines) {
		return fmt.Errorf("%s: description has no content", stepYMLPath)
	}
	indent := lines[first][:len(lines[first])-len(strings.TrimLeft(lines[first], " "))]
	last := first
	for last < len(lines) && (strings.TrimSpace(lines[last]) == "" || strings.HasPrefix(lines[last], indent)) {
		last++
	}

	rewritten := append([]string{}, lines[:first]...)
	for _, line := range strings.Split(strings.TrimRight(updated, "\n"), "\n") {
		if line == "" {
			rewritten = append(rewritten, "")
			continue
		}
		rewritten = append(rewritten, indent+line)
	}
	rewritten = append(rewritten, lines[last:]...)

	if err := os.WriteFile(stepYMLPath, []byte(strings.Join(rewritten, "\n")), 0600); err != nil {
		return fmt.Errorf("write %s: %w", stepYMLPath, err)
	}

	return nil
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
