package output

import (
	"github.com/bitrise-io/go-steputils/v2/export"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
)

type OutputExporter interface {
	ExportOutput(key, value string) error
}

type outputExporter struct {
	exporter export.Exporter
}

func NewOutputExporter() OutputExporter {
	return &outputExporter{exporter: export.NewExporter(command.NewFactory(env.NewRepository()))}
}

func (e *outputExporter) ExportOutput(key, value string) error {
	return e.exporter.ExportOutput(key, value)
}
