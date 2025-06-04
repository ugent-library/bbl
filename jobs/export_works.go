package jobs

import "github.com/ugent-library/bbl"

type ExportWorks struct {
	UserID string          `json:"user_id,omitempty"`
	Opts   *bbl.SearchOpts `json:"opts"`
	Format string          `json:"format"`
}

func (ExportWorks) Kind() string { return "export_works" }

type ExportWorksOutput struct {
	FileID string `json:"file_id"`
}
