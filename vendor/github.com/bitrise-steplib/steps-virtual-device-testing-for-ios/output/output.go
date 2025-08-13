package output

import (
	"encoding/xml"
	"fmt"
	"os"

	"github.com/bitrise-io/go-utils/v2/log"
)

const (
	flakyTestCasesEnvVarKey              = "BITRISE_FLAKY_TEST_CASES"
	flakyTestCasesEnvVarSizeLimitInBytes = 1024
)

type Exporter interface {
	ExportTestResultsDir(dir string) error
	ExportFlakyTestsEnvVar(mergedTestResultXmlPths []string) error
}

type exporter struct {
	outputExporter OutputExporter
	logger         log.Logger
}

func NewExporter(outputExporter OutputExporter, logger log.Logger) Exporter {
	return &exporter{
		outputExporter: outputExporter,
		logger:         logger,
	}
}

func (e exporter) ExportTestResultsDir(dir string) error {
	if err := e.outputExporter.ExportOutput("VDTESTING_DOWNLOADED_FILES_DIR", dir); err != nil {
		return err
	}
	e.logger.Donef("The downloaded test assets path (%s) is exported to the VDTESTING_DOWNLOADED_FILES_DIR environment variable.", dir)
	return nil
}

func (e exporter) ExportFlakyTestsEnvVar(mergedTestResultXmlPths []string) error {
	var flakyTestSuites []TestSuite
	for _, testResultXMLPth := range mergedTestResultXmlPths {
		testSuite, err := e.convertTestReport(testResultXMLPth)
		if err != nil {
			return fmt.Errorf("failed to convert test report (%s): %w", testResultXMLPth, err)
		}

		if testSuite.Flakes > 0 {
			flakyTestSuites = append(flakyTestSuites, testSuite)
		}
	}

	if err := e.exportFlakyTestCasesEnvVar(flakyTestSuites); err != nil {
		return fmt.Errorf("failed to export flaky test cases env var: %w", err)
	}

	return nil
}

func (e exporter) convertTestReport(pth string) (TestSuite, error) {
	data, err := os.ReadFile(pth)
	if err != nil {
		return TestSuite{}, err
	}

	var testSuite TestSuite
	if err := xml.Unmarshal(data, &testSuite); err != nil {
		return TestSuite{}, nil
	}

	return testSuite, nil
}

func (e exporter) exportFlakyTestCasesEnvVar(flakyTestSuites []TestSuite) error {
	if len(flakyTestSuites) == 0 {
		return nil
	}

	storedFlakyTestCases := map[string]bool{}
	var flakyTestCases []string
	for _, testSuite := range flakyTestSuites {
		for _, testCase := range testSuite.TestCases {
			if testCase.Flaky != "true" {
				continue
			}

			testCaseName := testCase.Name
			if len(testCase.ClassName) > 0 {
				testCaseName = fmt.Sprintf("%s.%s", testCase.ClassName, testCase.Name)
			}

			if len(testSuite.Name) > 0 {
				testCaseName = testSuite.Name + "." + testCaseName
			}

			if _, stored := storedFlakyTestCases[testCaseName]; !stored {
				storedFlakyTestCases[testCaseName] = true
				flakyTestCases = append(flakyTestCases, testCaseName)
			}
		}
	}

	if len(flakyTestCases) == 0 {
		return nil
	} else {
		e.logger.TDonef("%d flaky test case(s) detected, exporting %s env var", len(flakyTestCases), flakyTestCasesEnvVarKey)
	}

	var flakyTestCasesMessage string
	for i, flakyTestCase := range flakyTestCases {
		flakyTestCasesMessageLine := fmt.Sprintf("- %s\n", flakyTestCase)

		if len(flakyTestCasesMessage)+len(flakyTestCasesMessageLine) > flakyTestCasesEnvVarSizeLimitInBytes {
			e.logger.TWarnf("%s env var size limit (%d characters) exceeded. Skipping %d test cases.", flakyTestCasesEnvVarKey, flakyTestCasesEnvVarSizeLimitInBytes, len(flakyTestCases)-i)
			break
		}

		flakyTestCasesMessage += flakyTestCasesMessageLine
	}

	if err := e.outputExporter.ExportOutput(flakyTestCasesEnvVarKey, flakyTestCasesMessage); err != nil {
		return fmt.Errorf("failed to export %s: %w", flakyTestCasesEnvVarKey, err)
	}

	return nil
}
