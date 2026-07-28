// The catalog check is tooling, not part of the Step, so it lives in its own module: its
// dependencies stay out of the Step's go.mod and out of vendor/.
module github.com/bitrise-steplib/steps-virtual-device-testing-for-android/maintenance

go 1.21

require (
	github.com/bitrise-io/go-utils v1.0.13
	gopkg.in/yaml.v3 v3.0.1
)
