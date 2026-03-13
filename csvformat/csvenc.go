// Package csvformat encodes works as CSV.
package csvformat

import (
	"bytes"
	"encoding/csv"
	"io"

	"github.com/ugent-library/bbl"
)

var header = []string{
	"id",
	"kind",
	"status",
	"title",
	"publication_year",
}

// WorkEncoder encodes a single work as a CSV row (with header).
type WorkEncoder struct{}

func (e *WorkEncoder) Encode(work *bbl.Work) ([]byte, error) {
	var buf bytes.Buffer
	w := &WorkWriter{}
	if err := w.Begin(&buf); err != nil {
		return nil, err
	}
	if err := w.Encode(&buf, work); err != nil {
		return nil, err
	}
	if err := w.End(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WorkWriter writes a stream of works as CSV rows.
type WorkWriter struct{}

func (e *WorkWriter) Begin(w io.Writer) error {
	cw := csv.NewWriter(w)
	cw.Write(header)
	cw.Flush()
	return cw.Error()
}

func (e *WorkWriter) Encode(w io.Writer, work *bbl.Work) error {
	var title string
	if len(work.Titles) > 0 {
		title = work.Titles[0].Val
	}

	cw := csv.NewWriter(w)
	cw.Write([]string{
		work.ID.String(),
		work.Kind,
		work.Status,
		title,
		work.PublicationYear,
	})
	cw.Flush()
	return cw.Error()
}

func (e *WorkWriter) End(w io.Writer) error { return nil }

var (
	_ bbl.WorkEncoder = (*WorkEncoder)(nil)
	_ bbl.WorkWriter  = (*WorkWriter)(nil)
)
