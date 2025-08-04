package output

import (
	"fmt"

	"github.com/bitrise-io/go-steputils/v2/export"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/converters/junitxml"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/testreport"
)

const (
	flakyTestCasesEnvVarKey              = "BITRISE_FLAKY_TEST_CASES"
	flakyTestCasesEnvVarSizeLimitInBytes = 1024
)

type Exporter interface {
	ExportTestResultsDir(dir string) error
	ExportFlakyTestsEnvVar(testResultXML string) error
}

type exporter struct {
	outputExporter export.Exporter
	converter      junitxml.Converter
	logger         log.Logger
}

func NewExporter(outputExporter export.Exporter, converter junitxml.Converter, logger log.Logger) Exporter {
	return &exporter{
		outputExporter: outputExporter,
		converter:      converter,
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

func (e exporter) ExportFlakyTestsEnvVar(testResultXML string) error {
	testReport, err := e.convertTestReport(testResultXML)
	if err != nil {
		return fmt.Errorf("failed to convert test report (%s): %w", testResultXML, err)
	}

	flakyTestSuites := e.getFlakyTestSuites(testReport)

	if err := e.exportFlakyTestCasesEnvVar(flakyTestSuites); err != nil {
		return fmt.Errorf("failed to export flaky test cases env var: %w", err)
	}

	return nil
}

func (e exporter) convertTestReport(pth string) (testreport.TestReport, error) {
	if !e.converter.Detect([]string{pth}) {
		return testreport.TestReport{}, nil
	}

	testReport, err := e.converter.Convert()
	if err != nil {
		return testreport.TestReport{}, fmt.Errorf("failed to convert test report from %s: %w", pth, err)
	}

	return testReport, nil
}

func (e exporter) getFlakyTestSuites(testReport testreport.TestReport) []testreport.TestSuite {
	var flakyTestSuites []testreport.TestSuite

	for _, suite := range testReport.TestSuites {
		var flakyTests []testreport.TestCase
		testCasesToStatus := map[string]bool{}
		alreadySeenFlakyTests := map[string]bool{}

		for _, testCase := range suite.TestCases {
			testCaseID := testCase.ClassName + "." + testCase.Name

			newIsFailed := false
			if testCase.Failure != nil {
				newIsFailed = true
			}

			previousIsFailed, alreadySeen := testCasesToStatus[testCaseID]
			if !alreadySeen {
				testCasesToStatus[testCaseID] = newIsFailed
			} else {
				_, seen := alreadySeenFlakyTests[testCaseID]
				if !seen && (previousIsFailed != newIsFailed) {
					flakyTests = append(flakyTests, testreport.TestCase{
						XMLName:   testCase.XMLName,
						Name:      testCase.Name,
						ClassName: testCase.ClassName,
					})
					alreadySeenFlakyTests[testCaseID] = true
				}
			}
		}

		if len(flakyTests) > 0 {
			flakyTestSuites = append(flakyTestSuites, testreport.TestSuite{
				XMLName:   suite.XMLName,
				Name:      suite.Name,
				TestCases: flakyTests,
			})
		}
	}

	return flakyTestSuites
}

func (e exporter) exportFlakyTestCasesEnvVar(flakyTestSuites []testreport.TestSuite) error {
	if len(flakyTestSuites) == 0 {
		return nil
	}

	storedFlakyTestCases := map[string]bool{}
	var flakyTestCases []string

	for _, testSuite := range flakyTestSuites {
		for _, testCase := range testSuite.TestCases {
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

	if len(flakyTestCases) > 0 {
		e.logger.Donef("%d flaky test case(s) detected, exporting %s env var", len(flakyTestCases), flakyTestCasesEnvVarKey)
	}

	var flakyTestCasesMessage string
	for i, flakyTestCase := range flakyTestCases {
		flakyTestCasesMessageLine := fmt.Sprintf("- %s\n", flakyTestCase)

		if len(flakyTestCasesMessage)+len(flakyTestCasesMessageLine) > flakyTestCasesEnvVarSizeLimitInBytes {
			e.logger.Warnf("%s env var size limit (%d characters) exceeded. Skipping %d test cases.", flakyTestCasesEnvVarKey, flakyTestCasesEnvVarSizeLimitInBytes, len(flakyTestCases)-i)
			break
		}

		flakyTestCasesMessage += flakyTestCasesMessageLine
	}

	if err := e.outputExporter.ExportOutput(flakyTestCasesEnvVarKey, flakyTestCasesMessage); err != nil {
		return fmt.Errorf("failed to export %s: %w", flakyTestCasesEnvVarKey, err)
	}

	return nil
}
