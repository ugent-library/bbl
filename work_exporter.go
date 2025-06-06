package bbl

import (
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"maps"
)

type WorkExporter interface {
	Add(*Work) error
	Done() error
}

type WorkExporterFactory = func(io.Writer) (WorkExporter, error)

var workExporters = map[string]WorkExporterFactory{
	"jsonl": func(w io.Writer) (WorkExporter, error) {
		return &jsonlExporter[*Work]{enc: json.NewEncoder(w)}, nil
	},
}

func RegisterWorkExporter(format string, factory WorkExporterFactory) {
	workExporters[format] = factory
}

func WorkExporters() iter.Seq[string] {
	return maps.Keys(workExporters)
}

func NewWorkExporter(w io.Writer, format string) (WorkExporter, error) {
	factory, ok := workExporters[format]
	if !ok {
		return nil, fmt.Errorf("NewWorkExporter: unknown exporter %q", format)
	}
	return factory(w)
}
