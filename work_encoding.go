package bbl

import (
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"strings"
)

// WorkEncoder encodes a single work into a self-contained document.
type WorkEncoder interface {
	Encode(work *Work) ([]byte, error)
}

// WorkWriter writes a stream of works to a writer.
// Begin writes any preamble (e.g. CSV header, XML root open tag).
// Encode writes a single work.
// End writes any postamble (e.g. XML root close tag).
type WorkWriter interface {
	Begin(w io.Writer) error
	Encode(w io.Writer, work *Work) error
	End(w io.Writer) error
}

var workEncoders = map[string]func() WorkEncoder{
	"json":  func() WorkEncoder { return &jsonWorkEncoder{} },
	"jsonl": func() WorkEncoder { return &jsonlWorkEncoder{} },
}

var workWriters = map[string]func() WorkWriter{
	"json":  func() WorkWriter { return &jsonArrayWorkWriter{} },
	"jsonl": func() WorkWriter { return &jsonlWorkWriter{} },
}

// NewWorkEncoder creates a new encoder for the given format.
func NewWorkEncoder(format string) (WorkEncoder, error) {
	factory, ok := workEncoders[format]
	if !ok {
		return nil, fmt.Errorf("unknown work encoder format %q (available: %s)", format, strings.Join(WorkEncoderFormats(), ", "))
	}
	return factory(), nil
}

// NewWorkWriter creates a new writer for the given format.
func NewWorkWriter(format string) (WorkWriter, error) {
	factory, ok := workWriters[format]
	if !ok {
		return nil, fmt.Errorf("unknown work writer format %q (available: %s)", format, strings.Join(WorkWriterFormats(), ", "))
	}
	return factory(), nil
}

// EncodeWork is a convenience for encoding a single work.
func EncodeWork(format string, work *Work) ([]byte, error) {
	enc, err := NewWorkEncoder(format)
	if err != nil {
		return nil, err
	}
	return enc.Encode(work)
}

// RegisterWorkEncoder registers a custom work encoder format.
func RegisterWorkEncoder(format string, factory func() WorkEncoder) {
	workEncoders[format] = factory
}

// HasWorkEncoder reports whether a work encoder is registered for the given format.
func HasWorkEncoder(format string) bool {
	_, ok := workEncoders[format]
	return ok
}

// RegisterWorkWriter registers a custom work writer format.
func RegisterWorkWriter(format string, factory func() WorkWriter) {
	workWriters[format] = factory
}

// WorkEncoderFormats returns the available encoder format names.
func WorkEncoderFormats() []string {
	formats := make([]string, 0, len(workEncoders))
	for name := range workEncoders {
		formats = append(formats, name)
	}
	return formats
}

// WorkWriterFormats returns the available writer format names.
func WorkWriterFormats() []string {
	formats := make([]string, 0, len(workWriters))
	for name := range workWriters {
		formats = append(formats, name)
	}
	return formats
}

// WorkEncoderFormatsHelp returns a comma-separated list of available encoder formats.
func WorkEncoderFormatsHelp() string {
	return strings.Join(WorkEncoderFormats(), ", ")
}

// WorkWriterFormatsHelp returns a comma-separated list of available writer formats.
func WorkWriterFormatsHelp() string {
	return strings.Join(WorkWriterFormats(), ", ")
}

// WriteWorks writes works from an iterator using the given writer.
func WriteWorks(w io.Writer, exp WorkWriter, works iter.Seq2[*Work, error]) (int, error) {
	if err := exp.Begin(w); err != nil {
		return 0, err
	}
	var n int
	for work, err := range works {
		if err != nil {
			return n, err
		}
		if err := exp.Encode(w, work); err != nil {
			return n, err
		}
		n++
	}
	if err := exp.End(w); err != nil {
		return n, err
	}
	return n, nil
}

// WriteWork is a convenience for writing a single work (Begin+Encode+End).
func WriteWork(w io.Writer, exp WorkWriter, work *Work) error {
	if err := exp.Begin(w); err != nil {
		return err
	}
	if err := exp.Encode(w, work); err != nil {
		return err
	}
	return exp.End(w)
}

// --- JSON encoder ---

type jsonWorkEncoder struct{}

func (e *jsonWorkEncoder) Encode(work *Work) ([]byte, error) {
	return json.Marshal(work)
}

// --- JSONL encoder ---

type jsonlWorkEncoder struct{}

func (e *jsonlWorkEncoder) Encode(work *Work) ([]byte, error) {
	b, err := json.Marshal(work)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// --- JSONL writer ---

type jsonlWorkWriter struct{}

func (e *jsonlWorkWriter) Begin(w io.Writer) error { return nil }

func (e *jsonlWorkWriter) Encode(w io.Writer, work *Work) error {
	b, err := json.Marshal(work)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = w.Write(b)
	return err
}

func (e *jsonlWorkWriter) End(w io.Writer) error { return nil }

// --- JSON array writer ---

type jsonArrayWorkWriter struct {
	first bool
}

func (e *jsonArrayWorkWriter) Begin(w io.Writer) error {
	e.first = true
	_, err := w.Write([]byte("["))
	return err
}

func (e *jsonArrayWorkWriter) Encode(w io.Writer, work *Work) error {
	if !e.first {
		if _, err := w.Write([]byte(",")); err != nil {
			return err
		}
	}
	e.first = false
	b, err := json.Marshal(work)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func (e *jsonArrayWorkWriter) End(w io.Writer) error {
	_, err := w.Write([]byte("]\n"))
	return err
}
