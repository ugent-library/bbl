package bbl

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
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
	"json":   func() WorkEncoder { return &jsonWorkEncoder{} },
	"jsonl":  func() WorkEncoder { return &jsonlWorkEncoder{} },
	"csv":    func() WorkEncoder { return &csvWorkEncoder{} },
	"oai_dc": func() WorkEncoder { return &oaidcWorkEncoder{} },
}

var workWriters = map[string]func() WorkWriter{
	"json":  func() WorkWriter { return &jsonArrayWorkWriter{} },
	"jsonl": func() WorkWriter { return &jsonlWorkWriter{} },
	"csv":   func() WorkWriter { return &csvWorkWriter{} },
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

// --- CSV writer ---

var csvWorkHeader = []string{
	"id",
	"kind",
	"status",
	"title",
	"publication_year",
}

type csvWorkEncoder struct{}

func (e *csvWorkEncoder) Encode(work *Work) ([]byte, error) {
	var buf bytes.Buffer
	exp := &csvWorkWriter{}
	if err := exp.Begin(&buf); err != nil {
		return nil, err
	}
	if err := exp.Encode(&buf, work); err != nil {
		return nil, err
	}
	if err := exp.End(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type csvWorkWriter struct{}

func (e *csvWorkWriter) Begin(w io.Writer) error {
	cw := csv.NewWriter(w)
	cw.Write(csvWorkHeader)
	cw.Flush()
	return cw.Error()
}

func (e *csvWorkWriter) Encode(w io.Writer, work *Work) error {
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
		"", // TODO: publication year from assertions
	})
	cw.Flush()
	return cw.Error()
}

func (e *csvWorkWriter) End(w io.Writer) error { return nil }

// --- OAI DC encoder ---

const oaidcStart = `<oai_dc:dc xmlns:oai_dc="http://www.openarchives.org/OAI/2.0/oai_dc/" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.openarchives.org/OAI/2.0/oai_dc/ http://www.openarchives.org/OAI/2.0/oai_dc.xsd">`
const oaidcEnd = `</oai_dc:dc>`

type oaidcWorkEncoder struct{}

func (e *oaidcWorkEncoder) Encode(work *Work) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(oaidcStart)
	for _, t := range work.Titles {
		oaidcField(&buf, "title", t.Val)
	}
	for _, c := range work.Contributors {
		oaidcField(&buf, "creator", contributorName(c))
	}
	for _, kw := range work.Keywords {
		oaidcField(&buf, "subject", kw.Val)
	}
	for _, a := range work.Abstracts {
		oaidcField(&buf, "description", a.Val)
	}
	// TODO: publisher and publication year from assertions
	oaidcField(&buf, "type", work.Kind)
	for _, id := range work.Identifiers {
		oaidcField(&buf, "identifier", id.Val)
	}
	buf.WriteString(oaidcEnd)
	return buf.Bytes(), nil
}

func contributorName(c WorkContributor) string {
	if c.Name != "" {
		return c.Name
	}
	switch {
	case c.FamilyName != "" && c.GivenName != "":
		return c.FamilyName + ", " + c.GivenName
	case c.FamilyName != "":
		return c.FamilyName
	case c.GivenName != "":
		return c.GivenName
	default:
		return ""
	}
}

func oaidcField(buf *bytes.Buffer, tag, val string) {
	if val == "" {
		return
	}
	buf.WriteString("<dc:")
	buf.WriteString(tag)
	buf.WriteByte('>')
	xml.EscapeText(buf, []byte(val))
	buf.WriteString("</dc:")
	buf.WriteString(tag)
	buf.WriteByte('>')
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

// RegisterWorkEncoder registers a custom work encoder format.
func RegisterWorkEncoder(format string, factory func() WorkEncoder) {
	workEncoders[format] = factory
}

// RegisterWorkWriter registers a custom work writer format.
func RegisterWorkWriter(format string, factory func() WorkWriter) {
	workWriters[format] = factory
}
