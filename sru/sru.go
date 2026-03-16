// Package sru implements a minimal SRU (Search/Retrieve via URL) 1.2 server.
// Supports operations: explain, searchRetrieve.
// CQL support is minimal: free-text terms and index=value queries.
package sru

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
)

const (
	Version        = "1.2"
	NS             = "http://www.loc.gov/zing/srw/"
	DiagNS         = "http://www.loc.gov/zing/srw/diagnostic/"
	DefaultMaxRecs = 100

	// Standard record schema identifiers.
	SchemaDC      = "info:srw/schema/1/dc-v1.1"
	SchemaMARCXML = "info:srw/schema/1/marcxml-v1.1"
	SchemaMODS    = "info:srw/schema/1/mods-v3.6"
)

// Index maps a CQL index name to an internal field name.
type Index struct {
	CQLName string
	Title   string
	Field   string // internal field name for filtering; empty = free text query
}

// Schema describes a supported record schema.
type Schema struct {
	Name       string // short name, e.g. "oai_dc"
	Identifier string // URI, e.g. "info:srw/schema/1/dc-v1.1"
	Title      string
}

// SearchResult is what the search callback returns to the SRU handler.
type SearchResult struct {
	Total   int
	Records [][]byte // each entry is an encoded record (e.g. OAI-DC XML)
}

// SearchFunc is the callback the app provides. It receives the parsed CQL query
// and pagination params, and returns encoded records.
// The index and value come from CQL parsing. Index is empty for free-text (serverChoice).
// Offset is 0-based. Size is the maximum number of records to return.
type SearchFunc func(ctx context.Context, index, value string, offset, size int) (*SearchResult, error)

// ServerConfig configures an SRU endpoint.
type ServerConfig struct {
	Database string   // database name for explain
	Title    string   // human-readable title for explain
	Indexes  []Index  // supported CQL indexes
	Schemas  []Schema // supported record schemas
	Search   SearchFunc
}

// Handler returns an http.Handler that implements the SRU protocol.
func Handler(cfg ServerConfig) http.Handler {
	defaultSchema := ""
	if len(cfg.Schemas) > 0 {
		defaultSchema = cfg.Schemas[0].Name
	}

	indexMap := make(map[string]Index)
	for _, idx := range cfg.Indexes {
		indexMap[idx.CQLName] = idx
	}

	schemaSet := make(map[string]bool)
	for _, s := range cfg.Schemas {
		schemaSet[s.Name] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := parseRequest(r)

		switch req.operation {
		case "", "explain":
			writeXML(w, http.StatusOK, buildExplain(cfg))
			return
		case "searchRetrieve":
			// handled below
		default:
			writeXML(w, http.StatusOK, newDiagnostic(
				DiagUnsupportedOperation, "Unsupported operation", req.operation,
			))
			return
		}

		if req.version != "" && req.version != Version {
			writeXML(w, http.StatusOK, newDiagnostic(
				DiagUnsupportedVersion, "Unsupported version", req.version,
			))
			return
		}

		cql, err := parseCQL(req.query)
		if err != nil {
			writeXML(w, http.StatusOK, newDiagnostic(
				DiagQuerySyntaxError, "Query syntax error", err.Error(),
			))
			return
		}

		// Resolve CQL index.
		var index, value string
		switch cql.index {
		case "", "cql.serverChoice":
			value = cql.value
		default:
			idx, ok := indexMap[cql.index]
			if !ok {
				writeXML(w, http.StatusOK, newDiagnostic(
					DiagUnsupportedIndex, "Unsupported index", cql.index,
				))
				return
			}
			if idx.Field == "" {
				value = cql.value
			} else {
				index = idx.Field
				value = cql.value
			}
		}

		schema := req.recordSchema
		if schema == "" {
			schema = defaultSchema
		}
		if !schemaSet[schema] {
			writeXML(w, http.StatusOK, newDiagnostic(
				DiagUnknownSchemaForRetrieval, "Unknown schema", schema,
			))
			return
		}

		result, err := cfg.Search(r.Context(), index, value, req.startRecord-1, req.maximumRecords)
		if err != nil {
			writeXML(w, http.StatusOK, newDiagnostic(
				DiagGeneralSystemError, "Search error", "",
			))
			return
		}

		var records []record
		for i, rec := range result.Records {
			records = append(records, newRecord(schema, req.recordPacking, req.startRecord+i, rec))
		}

		writeXML(w, http.StatusOK, newSearchResponse(req, result.Total, records))
	})
}

// --- internal request parsing ---

type request struct {
	operation      string
	version        string
	query          string
	startRecord    int
	maximumRecords int
	recordSchema   string
	recordPacking  string
}

func parseRequest(r *http.Request) request {
	q := r.URL.Query()
	req := request{
		operation:      q.Get("operation"),
		version:        q.Get("version"),
		query:          q.Get("query"),
		recordSchema:   q.Get("recordSchema"),
		recordPacking:  q.Get("recordPacking"),
		startRecord:    1,
		maximumRecords: 10,
	}
	if v, err := strconv.Atoi(q.Get("startRecord")); err == nil && v > 0 {
		req.startRecord = v
	}
	if v, err := strconv.Atoi(q.Get("maximumRecords")); err == nil && v > 0 {
		req.maximumRecords = min(v, DefaultMaxRecs)
	}
	if req.recordPacking == "" {
		req.recordPacking = "xml"
	}
	return req
}

// --- internal CQL parsing ---

type cqlQuery struct {
	index string
	value string
}

func parseCQL(query string) (cqlQuery, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return cqlQuery{}, fmt.Errorf("empty query")
	}
	if before, after, ok := strings.Cut(query, "="); ok {
		index := strings.TrimSpace(before)
		value := strings.TrimSpace(after)
		value = strings.Trim(value, `"`)
		if index == "" || value == "" {
			return cqlQuery{}, fmt.Errorf("invalid CQL: empty index or value")
		}
		return cqlQuery{index: index, value: value}, nil
	}
	return cqlQuery{value: query}, nil
}

// --- XML response types ---

type searchRetrieveResponse struct {
	XMLName             xml.Name                     `xml:"searchRetrieveResponse"`
	XMLNS               string                       `xml:"xmlns,attr"`
	Version             string                       `xml:"version"`
	NumberOfRecords     int                          `xml:"numberOfRecords"`
	NextRecordPosition  int                          `xml:"nextRecordPosition,omitempty"`
	Records             *records                     `xml:"records,omitempty"`
	Diagnostics         *diagnostics                 `xml:"diagnostics,omitempty"`
	EchoedRequest       *echoedSearchRetrieveRequest `xml:"echoedSearchRetrieveRequest,omitempty"`
}

type echoedSearchRetrieveRequest struct {
	Version        string `xml:"version"`
	Query          string `xml:"query"`
	StartRecord    int    `xml:"startRecord"`
	MaximumRecords int    `xml:"maximumRecords"`
	RecordPacking  string `xml:"recordPacking"`
	RecordSchema   string `xml:"recordSchema,omitempty"`
}

type records struct {
	Record []record `xml:"record"`
}

type record struct {
	RecordSchema   string     `xml:"recordSchema"`
	RecordPacking  string     `xml:"recordPacking"`
	RecordData     recordData `xml:"recordData"`
	RecordPosition int        `xml:"recordPosition"`
}

type recordData struct {
	XML string `xml:",innerxml"`
}

type diagnostics struct {
	Diagnostic []diagnostic `xml:"diagnostic"`
}

type diagnostic struct {
	XMLNS   string `xml:"xmlns,attr"`
	URI     string `xml:"uri"`
	Details string `xml:"details,omitempty"`
	Message string `xml:"message"`
}

type explainResponse struct {
	XMLName xml.Name `xml:"explainResponse"`
	XMLNS   string   `xml:"xmlns,attr"`
	Version string   `xml:"version"`
	Record  record   `xml:"record"`
}

// --- response builders ---

func newSearchResponse(req request, total int, recs []record) *searchRetrieveResponse {
	resp := &searchRetrieveResponse{
		XMLNS:           NS,
		Version:         Version,
		NumberOfRecords: total,
		EchoedRequest: &echoedSearchRetrieveRequest{
			Version:        Version,
			Query:          req.query,
			StartRecord:    req.startRecord,
			MaximumRecords: req.maximumRecords,
			RecordPacking:  req.recordPacking,
			RecordSchema:   req.recordSchema,
		},
	}
	if len(recs) > 0 {
		resp.Records = &records{Record: recs}
	}
	nextPos := req.startRecord + len(recs)
	if nextPos <= total {
		resp.NextRecordPosition = nextPos
	}
	return resp
}

func newDiagnostic(uri, message, details string) *searchRetrieveResponse {
	return &searchRetrieveResponse{
		XMLNS:           NS,
		Version:         Version,
		NumberOfRecords: 0,
		Diagnostics: &diagnostics{
			Diagnostic: []diagnostic{{
				XMLNS:   DiagNS,
				URI:     uri,
				Message: message,
				Details: details,
			}},
		},
	}
}

func newRecord(schema, packing string, position int, data []byte) record {
	content := string(data)
	if packing == "string" {
		content = html.EscapeString(content)
	}
	return record{
		RecordSchema:   schema,
		RecordPacking:  packing,
		RecordData:     recordData{XML: content},
		RecordPosition: position,
	}
}

func buildExplain(cfg ServerConfig) *explainResponse {
	var indexXML string
	for _, idx := range cfg.Indexes {
		indexXML += fmt.Sprintf(`<index><title>%s</title><map><name set="cql">%s</name></map></index>`, idx.Title, idx.CQLName)
	}

	var schemaXML string
	for _, s := range cfg.Schemas {
		schemaXML += fmt.Sprintf(`<schema identifier="%s" name="%s"><title>%s</title></schema>`, s.Identifier, s.Name, s.Title)
	}

	explainXML := fmt.Sprintf(`<explain xmlns="http://explain.z3950.org/dtd/2.0/">
  <serverInfo>
    <host></host>
    <port></port>
    <database>%s</database>
  </serverInfo>
  <databaseInfo>
    <title>%s</title>
  </databaseInfo>
  <indexInfo>%s</indexInfo>
  <schemaInfo>%s</schemaInfo>
</explain>`, cfg.Database, cfg.Title, indexXML, schemaXML)

	return &explainResponse{
		XMLNS:   NS,
		Version: Version,
		Record: record{
			RecordSchema:   "http://explain.z3950.org/dtd/2.0/",
			RecordPacking:  "xml",
			RecordData:     recordData{XML: explainXML},
			RecordPosition: 1,
		},
	}
}

func writeXML(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	enc.Encode(v)
}

// Standard SRU diagnostic URIs.
const (
	DiagGeneralSystemError        = "info:srw/diagnostic/1/1"
	DiagUnsupportedOperation      = "info:srw/diagnostic/1/4"
	DiagUnsupportedVersion        = "info:srw/diagnostic/1/5"
	DiagQuerySyntaxError          = "info:srw/diagnostic/1/10"
	DiagUnsupportedIndex          = "info:srw/diagnostic/1/16"
	DiagUnknownSchemaForRetrieval = "info:srw/diagnostic/1/66"
)
