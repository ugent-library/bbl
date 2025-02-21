package cli

import (
	"io"
	"log/slog"

	"github.com/lmittmann/tint"
)

func newLogger(w io.Writer) *slog.Logger {
	if config.Env == "development" {
		return slog.New(tint.NewHandler(w, &tint.Options{Level: slog.LevelDebug}))
	} else {
		return slog.New(slog.NewJSONHandler(w, nil))
	}
}
