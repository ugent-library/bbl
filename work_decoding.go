package bbl

import (
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"strings"
)

// WorkDecoder decodes a single work import record from bytes.
type WorkDecoder interface {
	Decode(data []byte) (*ImportWorkInput, error)
}

// WorkReader reads a stream of work import records from a reader.
type WorkReader interface {
	Read(r io.Reader) iter.Seq2[*ImportWorkInput, error]
}

var workDecoders = map[string]func() WorkDecoder{
	"json": func() WorkDecoder { return &jsonWorkDecoder{} },
}

var workReaders = map[string]func() WorkReader{
	"jsonl": func() WorkReader { return &jsonlWorkReader{} },
}

// NewWorkDecoder creates a new decoder for the given format.
func NewWorkDecoder(format string) (WorkDecoder, error) {
	factory, ok := workDecoders[format]
	if !ok {
		return nil, fmt.Errorf("unknown work decoder format %q (available: %s)", format, strings.Join(WorkDecoderFormats(), ", "))
	}
	return factory(), nil
}

// NewWorkReader creates a new reader for the given format.
func NewWorkReader(format string) (WorkReader, error) {
	factory, ok := workReaders[format]
	if !ok {
		return nil, fmt.Errorf("unknown work reader format %q (available: %s)", format, strings.Join(WorkReaderFormats(), ", "))
	}
	return factory(), nil
}

// DecodeWork is a convenience for decoding a single work.
func DecodeWork(format string, data []byte) (*ImportWorkInput, error) {
	dec, err := NewWorkDecoder(format)
	if err != nil {
		return nil, err
	}
	return dec.Decode(data)
}

// ReadWorks is a convenience for reading works from a reader.
func ReadWorks(r io.Reader, format string) (iter.Seq2[*ImportWorkInput, error], error) {
	rdr, err := NewWorkReader(format)
	if err != nil {
		return nil, err
	}
	return rdr.Read(r), nil
}

// --- JSON decoder ---

type jsonWorkDecoder struct{}

func (d *jsonWorkDecoder) Decode(data []byte) (*ImportWorkInput, error) {
	var v ImportWorkInput
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	v.SourceRecord = data
	return &v, nil
}

// --- JSONL reader ---

type jsonlWorkReader struct{}

func (d *jsonlWorkReader) Read(r io.Reader) iter.Seq2[*ImportWorkInput, error] {
	return func(yield func(*ImportWorkInput, error) bool) {
		dec := json.NewDecoder(r)
		for {
			var raw json.RawMessage
			if err := dec.Decode(&raw); err == io.EOF {
				return
			} else if err != nil {
				yield(nil, err)
				return
			}
			var v ImportWorkInput
			if err := json.Unmarshal(raw, &v); err != nil {
				yield(nil, err)
				return
			}
			v.SourceRecord = raw
			if !yield(&v, nil) {
				return
			}
		}
	}
}

// WorkDecoderFormats returns the available decoder format names.
func WorkDecoderFormats() []string {
	formats := make([]string, 0, len(workDecoders))
	for name := range workDecoders {
		formats = append(formats, name)
	}
	return formats
}

// WorkReaderFormats returns the available reader format names.
func WorkReaderFormats() []string {
	formats := make([]string, 0, len(workReaders))
	for name := range workReaders {
		formats = append(formats, name)
	}
	return formats
}

// WorkDecoderFormatsHelp returns a comma-separated list of available decoder formats.
func WorkDecoderFormatsHelp() string {
	return strings.Join(WorkDecoderFormats(), ", ")
}

// WorkReaderFormatsHelp returns a comma-separated list of available reader formats.
func WorkReaderFormatsHelp() string {
	return strings.Join(WorkReaderFormats(), ", ")
}

// RegisterWorkDecoder registers a custom work decoder format.
func RegisterWorkDecoder(format string, factory func() WorkDecoder) {
	workDecoders[format] = factory
}

// RegisterWorkReader registers a custom work reader format.
func RegisterWorkReader(format string, factory func() WorkReader) {
	workReaders[format] = factory
}
