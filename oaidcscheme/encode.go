package oaidcscheme

import (
	"bytes"
	"context"
	"encoding/xml"

	"github.com/ugent-library/bbl"
)

const startTag = `<oai_dc:dc xmlns="http://www.openarchives.org/OAI/2.0/oai_dc/"
xmlns:oai_dc="http://www.openarchives.org/OAI/2.0/oai_dc/"
xmlns:dc="http://purl.org/dc/elements/1.1/"
xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
xsi:schemaLocation="http://www.openarchives.org/OAI/2.0/oai_dc/ http://www.openarchives.org/OAI/2.0/oai_dc.xsd">
`

const endTag = `
</oai_dc:dc>
`

func EncodeWork(ctx context.Context, rec *bbl.Work) ([]byte, error) {
	b := &bytes.Buffer{}
	b.WriteString(startTag)

	for _, text := range rec.Attrs.Titles {
		writeField(b, "title", text.Val)
	}

	b.WriteString(endTag)

	return b.Bytes(), nil
}

func writeField(b *bytes.Buffer, tag, val string) {
	if val != "" {
		b.WriteString("<dc:")
		b.WriteString(tag)
		b.WriteString(">")
		xml.EscapeText(b, []byte(val))
		b.WriteString("</dc:")
		b.WriteString(tag)
		b.WriteString(">")
	}
}
