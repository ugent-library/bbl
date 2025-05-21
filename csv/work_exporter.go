package csv

import (
	"encoding/csv"
	"io"
	"iter"

	"github.com/ugent-library/bbl"
)

var header = []string{
	"id",
	"kind",
	"subkind",
	"title",
}

func ExportWorks(recs iter.Seq[*bbl.Work], w io.Writer) error {
	writer := csv.NewWriter(w)
	if err := writer.Write(header); err != nil {
		return err
	}
	row := make([]string, len(header))
	for rec := range recs {
		row[0] = rec.ID
		row[1] = rec.Kind
		row[2] = rec.Subkind
		row[3] = rec.Title()
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
