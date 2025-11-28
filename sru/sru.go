package sru

import (
	"encoding/xml"
	"net/http"
	"strconv"
)

type Server struct {
	SearchProvider func(string, int) ([][]byte, int, error)
	Host           string
	Port           int
}

// TODO error handling
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	op := r.URL.Query().Get("operation")

	var res *response

	switch op {
	case "searchRetrieve":
		q := r.URL.Query().Get("query")
		sizeStr := r.URL.Query().Get("maximumRecords")
		size, _ := strconv.Atoi(sizeStr)
		recs, total, _ := s.SearchProvider(q, size)
		res = &response{
			XMLName:         xml.Name{Space: "http://docs.oasis-open.org/ns/search-ws/sruResponse", Local: "searchRetrieveResponse"},
			Version:         "2.0",
			NumberOfRecords: total,
			Record:          make([]record, len(recs)),
		}
		for i, rec := range recs {
			res.Record[i] = record{
				RecordXMLEscaping: "xml",
				RecordPosition:    i + 1,
				RecordData:        recordData{rec},
			}
		}
	case "explain", "":
		rec, _ := xml.Marshal(explain{
			Host: s.Host,
			Port: s.Port,
		})
		res = &response{
			XMLName:         xml.Name{Space: "http://docs.oasis-open.org/ns/search-ws/sruResponse", Local: "explainResponse"},
			Version:         "2.0",
			NumberOfRecords: 1,
			Record: []record{{
				RecordSchema:      "http://explain.z3950.org/dtd/2.0/",
				RecordXMLEscaping: "xml",
				RecordPosition:    1,
				RecordData:        recordData{rec},
			}},
		}
	}

	b, _ := xml.Marshal(res)
	w.Header().Set("Content-Type", "application/xml")
	w.Write(b)
}

type response struct {
	XMLName            xml.Name
	Version            string   `xml:"version"`
	NumberOfRecords    int      `xml:"numberOfRecords"`
	NextRecordPosition int      `xml:"nextRecordPosition"`
	Record             []record `xml:"record"`
}

type record struct {
	RecordSchema      string     `xml:"recordSchema"`
	RecordXMLEscaping string     `xml:"recordXMLEscaping"`
	RecordData        recordData `xml:"recordData"`
	RecordPosition    int        `xml:"recordPosition"`
}

type recordData struct {
	RecordData []byte `xml:",innerxml"`
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
