package sru

import (
	"encoding/xml"
	"net/http"

	"github.com/ugent-library/bbl"
)

type Server struct {
	Index bbl.Index
	Host  string
	Port  int
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	op := r.URL.Query().Get("operation")

	var res any

	switch op {
	case "explain", "":
		res = &explainResponse{
			Version: "2.0",
			Record: []record{{
				RecordSchema:      "http://explain.z3950.org/dtd/2.0/",
				RecordXMLEscaping: "xml",
				RecordData: explain{
					Host: s.Host,
					Port: s.Port,
				},
			}},
		}
	}

	b, _ := xml.Marshal(res)
	w.Header().Set("Content-Type", "application/xml")
	w.Write(b)
}

type explainResponse struct {
	XMLName xml.Name `xml:"http://www.loc.gov/zing/srw/ explainResponse"`
	Version string   `xml:"version"`
	Record  []record `xml:"record"`
}

type record struct {
	RecordSchema      string `xml:"recordSchema"`
	RecordXMLEscaping string `xml:"recordXMLEscaping"`
	RecordData        any    `xml:"recordData"`
	RecordPosition    int    `xml:"recordPosition,omitempty"`
}

type explain struct {
	XMLName     xml.Name   `xml:"http://explain.z3950.org/dtd/2.0/ explain"`
	Host        string     `xml:"serverInfo>host"`
	Port        int        `xml:"serverInfo>port"`
	Title       []info     `xml:"databaseInfo>title"`
	Description []info     `xml:"databaseInfo>description"`
	IndexInfo   indexInfo  `xml:"indexInfo"`
	SchemaInfo  schemaInfo `xml:"schemaInfo"`
}

type info struct {
	Lang    string `xml:"lang,attr,omitempty"`
	Primary bool   `xml:"primary,attr,omitempty"`
	Value   string `xml:",chardata"`
}

type indexInfo struct {
	Index []index `xml:"index"`
}

type index struct {
	ID    string `xml:"id,attr,omitempty"`
	Title string `xml:"title"`
	Name  string `xml:"map>name"`
}

type schemaInfo struct {
	Schema []schema `xml:"schema"`
}

type schema struct {
	Identifier string `xml:"identifier,attr"`
	Name       string `xml:"name,attr"`
	Sort       bool   `xml:"sort,attr"`
	Title      string `xml:"title"`
}
