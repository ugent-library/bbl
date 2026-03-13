// Package oaidcformat encodes works as OAI Dublin Core XML.
package oaidcformat

import (
	"bytes"
	"encoding/xml"

	"github.com/ugent-library/bbl"
)

const xmlStart = `<oai_dc:dc xmlns:oai_dc="http://www.openarchives.org/OAI/2.0/oai_dc/" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.openarchives.org/OAI/2.0/oai_dc/ http://www.openarchives.org/OAI/2.0/oai_dc.xsd">`
const xmlEnd = `</oai_dc:dc>`

// WorkEncoder encodes a single work as OAI Dublin Core XML.
type WorkEncoder struct{}

func (e *WorkEncoder) Encode(work *bbl.Work) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(xmlStart)
	for _, t := range work.Titles {
		field(&buf, "title", t.Val)
	}
	for _, c := range work.Contributors {
		field(&buf, "creator", contributorName(c))
	}
	for _, kw := range work.Keywords {
		field(&buf, "subject", kw.Val)
	}
	for _, a := range work.Abstracts {
		field(&buf, "description", a.Val)
	}
	if work.Publisher != "" {
		field(&buf, "publisher", work.Publisher)
	}
	if work.PublicationYear != "" {
		field(&buf, "date", work.PublicationYear)
	}
	field(&buf, "type", work.Kind)
	for _, id := range work.Identifiers {
		field(&buf, "identifier", id.Val)
	}
	buf.WriteString(xmlEnd)
	return buf.Bytes(), nil
}

func contributorName(c bbl.WorkContributor) string {
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

func field(buf *bytes.Buffer, tag, val string) {
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

var _ bbl.WorkEncoder = (*WorkEncoder)(nil)
