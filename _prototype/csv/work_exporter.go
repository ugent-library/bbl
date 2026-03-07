package csv

import (
	"encoding/csv"
	"io"

	"github.com/ugent-library/bbl"
)

var header = []string{
	"id",
	"kind",
	"subkind",
	"title",
}

type Exporter struct {
	writer *csv.Writer
}

func NewWorkExporter(w io.Writer) (bbl.WorkExporter, error) {
	writer := csv.NewWriter(w)
	if err := writer.Write(header); err != nil {
		return nil, err
	}
	return &Exporter{writer: writer}, nil
}

func (e *Exporter) Add(rec *bbl.Work) error {
	return e.writer.Write([]string{
		rec.ID,
		rec.Kind,
		rec.Subkind,
		rec.GetTitle(),
	})
}

func (e *Exporter) Done() error {
	e.writer.Flush()
	return e.writer.Error()
}
