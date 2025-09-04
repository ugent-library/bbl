package bbl

import (
	"fmt"
	"iter"
	"maps"
)

type WorkImporter interface {
	Get(string) (*Work, error)
}

var workImporters = map[string]WorkImporter{}

func RegisterWorkImporter(source string, importer WorkImporter) {
	workImporters[source] = importer
}

func WorkImporters() iter.Seq[string] {
	return maps.Keys(workImporters)
}

func ImportWork(source, id string) (*Work, error) {
	importer, ok := workImporters[source]
	if !ok {
		return nil, fmt.Errorf("ImportWork: unknown importer %q", source)
	}
	return importer.Get(id)
}
