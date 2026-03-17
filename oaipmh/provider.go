package oaipmh

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"
)

const (
	xmlnsXsi          = "http://www.w3.org/2001/XMLSchema-instance"
	xsiSchemaLocation = "http://www.openarchives.org/OAI/2.0/ http://www.openarchives.org/OAI/2.0/OAI-PMH.xsd"
)

// Well-known metadata format.
var OAIDC = MetadataFormat{
	MetadataPrefix:    "oai_dc",
	Schema:            "http://www.openarchives.org/OAI/2.0/oai_dc.xsd",
	MetadataNamespace: "http://www.openarchives.org/OAI/2.0/oai_dc/",
}

// OAI-PMH errors returned by RecordProvider implementations.
var (
	ErrCannotDisseminateFormat = &Error{Code: "cannotDisseminateFormat", Value: "the metadata format identified by the value given for the metadataPrefix argument is not supported by the item or by the repository"}
	ErrBadResumptionToken      = &Error{Code: "badResumptionToken", Value: "the value of the resumptionToken argument is invalid or expired"}
	ErrIDDoesNotExist          = &Error{Code: "idDoesNotExist", Value: "identifier is unknown or illegal"}
	ErrNoRecordsMatch          = &Error{Code: "noRecordsMatch", Value: "no records match"}
)

// Internal protocol errors.
var (
	errBadVerb                  = &Error{Code: "badVerb", Value: "verb is invalid"}
	errVerbMissing              = &Error{Code: "badVerb", Value: "verb is missing"}
	errVerbRepeated             = &Error{Code: "badVerb", Value: "verb can't be repeated"}
	errNoSetHierarchy           = &Error{Code: "noSetHierarchy", Value: "sets are not supported"}
	errResumptiontokenExclusive = &Error{Code: "badArgument", Value: "resumptionToken cannot be combined with other attributes"}
	errMetadataPrefixMissing    = &Error{Code: "badArgument", Value: "metadataPrefix is missing"}
	errIdentifierMissing        = &Error{Code: "badArgument", Value: "identifier is missing"}
	errFromInvalid              = &Error{Code: "badArgument", Value: "from is not a valid datestamp"}
	errUntilInvalid             = &Error{Code: "badArgument", Value: "until is not a valid datestamp"}
	errSetDoesNotExist          = &Error{Code: "badArgument", Value: "set is unknown"}
)

// --- RecordProvider interface ---

// Query holds the parameters for a ListRecords/ListIdentifiers request.
type Query struct {
	MetadataPrefix string
	Set            string
	From           time.Time
	Until          time.Time
	Cursor         string // opaque, from previous Page
	Limit          int
}

// Page is a page of records returned by RecordProvider.ListRecords.
type Page struct {
	Records []*Record
	Cursor  string // empty = last page
	Total   int    // 0 = unknown (maps to completeListSize)
}

// RecordProvider is the required interface for serving OAI-PMH records.
type RecordProvider interface {
	GetEarliestDatestamp(ctx context.Context) (time.Time, error)
	ListRecords(ctx context.Context, q Query) (*Page, error)
	GetRecord(ctx context.Context, identifier, metadataPrefix string) (*Record, error)
}

// IdentifierProvider is an optional interface. If the RecordProvider implements it,
// ListIdentifiers uses it instead of ListRecords (avoids encoding metadata).
type IdentifierProvider interface {
	ListIdentifiers(ctx context.Context, q Query) (*IdentifierPage, error)
}

// IdentifierPage is a page of headers returned by IdentifierProvider.
type IdentifierPage struct {
	Headers []*Header
	Cursor  string
	Total   int
}

// SetProvider is an optional interface. If the RecordProvider implements it, sets are supported.
type SetProvider interface {
	ListSets(ctx context.Context) ([]Set, error)
	HasSet(ctx context.Context, spec string) (bool, error)
}

// --- Config ---

type Config struct {
	RepositoryName  string
	BaseURL         string
	AdminEmails     []string
	Descriptions    []string // raw XML for <description> elements in Identify
	MetadataFormats []MetadataFormat
	Granularity     string // "YYYY-MM-DD" or "YYYY-MM-DDThh:mm:ssZ" (default)
	Compression     string
	DeletedRecord   string // "no", "transient", or "persistent" (default)
	PageSize        int    // records per page (default 50)
	StyleSheet      string
	ErrorHandler    func(error)
	RecordProvider  RecordProvider
}

// --- Provider ---

type Provider struct {
	cfg        Config
	dateFormat string
	pageSize   int
	formats    map[string]bool
}

func NewProvider(cfg Config) (*Provider, error) {
	p := &Provider{cfg: cfg}

	if p.cfg.Granularity == "" {
		p.cfg.Granularity = "YYYY-MM-DDThh:mm:ssZ"
	}
	if p.cfg.DeletedRecord == "" {
		p.cfg.DeletedRecord = "persistent"
	}
	p.pageSize = p.cfg.PageSize
	if p.pageSize <= 0 {
		p.pageSize = 50
	}

	switch p.cfg.Granularity {
	case "YYYY-MM-DD":
		p.dateFormat = "2006-01-02"
	case "YYYY-MM-DDThh:mm:ssZ":
		p.dateFormat = "2006-01-02T15:04:05Z"
	default:
		return nil, errors.New("OAI-PMH granularity should be YYYY-MM-DD or YYYY-MM-DDThh:mm:ssZ")
	}

	p.formats = make(map[string]bool)
	for _, f := range p.cfg.MetadataFormats {
		p.formats[f.MetadataPrefix] = true
	}

	return p, nil
}

func (p *Provider) setProvider() SetProvider {
	if sp, ok := p.cfg.RecordProvider.(SetProvider); ok {
		return sp
	}
	return nil
}

// --- Verb definitions ---

type handleFunc func(context.Context, *Provider, *oaiRequest) error

type verbSpec struct {
	allowed []string // allowed query params (besides "verb")
	chain   []handleFunc
}

var verbSpecs = map[string]verbSpec{
	"Identify": {
		chain: []handleFunc{identify},
	},
	"ListMetadataFormats": {
		allowed: []string{"identifier"},
		chain:   []handleFunc{listMetadataFormats},
	},
	"ListSets": {
		allowed: []string{"resumptionToken"},
		chain:   []handleFunc{validateResumptionToken, listSets},
	},
	"ListIdentifiers": {
		allowed: []string{"resumptionToken", "metadataPrefix", "set", "from", "until"},
		chain:   []handleFunc{validateResumptionToken, requireMetadataPrefix, validateSet, validateFromUntil, listIdentifiers},
	},
	"ListRecords": {
		allowed: []string{"resumptionToken", "metadataPrefix", "set", "from", "until"},
		chain:   []handleFunc{validateResumptionToken, requireMetadataPrefix, validateSet, validateFromUntil, doListRecords},
	},
	"GetRecord": {
		allowed: []string{"metadataPrefix", "identifier"},
		chain:   []handleFunc{requireMetadataPrefix, requireIdentifier, getRecord},
	},
}

// --- HTTP handler ---

func (p *Provider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := &oaiRequest{baseURL: p.cfg.BaseURL}

	ctx := r.Context()
	args := r.URL.Query()

	verbs, ok := args["verb"]
	if !ok {
		req.addError(errVerbMissing)
		p.writeResponse(w, req)
		return
	}
	if len(verbs) > 1 {
		req.addError(errVerbRepeated)
		p.writeResponse(w, req)
		return
	}

	req.verb = verbs[0]
	def, ok := verbSpecs[req.verb]
	if !ok {
		req.addError(errBadVerb)
		p.writeResponse(w, req)
		return
	}

	// Parse all raw args into oaiRequest + check for illegal params.
	req.parseArgs(args)
	p.checkAllowedArgs(req, args, def.allowed)

	for i, h := range def.chain {
		if i == len(def.chain)-1 && len(req.errors) > 0 {
			break
		}
		if err := h(ctx, p, req); err != nil {
			p.handleError(w, err)
			return
		}
	}

	p.writeResponse(w, req)
}

// --- internal request state ---

type oaiRequest struct {
	baseURL string
	verb    string

	// Raw values parsed from query params.
	metadataPrefix  string
	identifier      string
	set             string
	from            string
	until           string
	resumptionToken string

	// Parsed time values.
	fromTime  time.Time
	untilTime time.Time

	errors []*Error
	body   any
}

func (r *oaiRequest) addError(err *Error) {
	r.errors = append(r.errors, err)
}

// parseArgs extracts all known OAI-PMH params from the query string,
// validating that none are repeated or empty.
func (r *oaiRequest) parseArgs(q map[string][]string) {
	r.metadataPrefix = r.extractArg(q, "metadataPrefix")
	r.identifier = r.extractArg(q, "identifier")
	r.set = r.extractArg(q, "set")
	r.from = r.extractArg(q, "from")
	r.until = r.extractArg(q, "until")
	r.resumptionToken = r.extractArg(q, "resumptionToken")
}

func (r *oaiRequest) extractArg(q map[string][]string, attr string) string {
	vals, ok := q[attr]
	if !ok {
		return ""
	}
	if len(vals) > 1 {
		r.addError(&Error{Code: "badArgument", Value: fmt.Sprintf("%s can't be repeated", attr)})
		return ""
	}
	if vals[0] == "" {
		r.addError(&Error{Code: "badArgument", Value: fmt.Sprintf("%s is missing", attr)})
		return ""
	}
	return vals[0]
}

func (p *Provider) checkAllowedArgs(req *oaiRequest, args map[string][]string, allowed []string) {
	for key := range args {
		if key == "verb" {
			continue
		}
		if !slices.Contains(allowed, key) {
			req.addError(&Error{Code: "badArgument", Value: fmt.Sprintf("argument %s is illegal", key)})
		}
	}
}

// --- XML types ---

type xmlResponse struct {
	XMLName           xml.Name `xml:"http://www.openarchives.org/OAI/2.0/ OAI-PMH"`
	XmlnsXsi          string   `xml:"xmlns:xsi,attr"`
	XsiSchemaLocation string   `xml:"xsi:schemaLocation,attr"`
	ResponseDate      string   `xml:"responseDate"`
	Request           xmlRequest
	Errors            []*Error
	Body              any
}

type xmlRequest struct {
	XMLName         xml.Name `xml:"request"`
	URL             string   `xml:",chardata"`
	Verb            string   `xml:"verb,attr,omitempty"`
	MetadataPrefix  string   `xml:"metadataPrefix,attr,omitempty"`
	Identifier      string   `xml:"identifier,attr,omitempty"`
	Set             string   `xml:"set,attr,omitempty"`
	From            string   `xml:"from,attr,omitempty"`
	Until           string   `xml:"until,attr,omitempty"`
	ResumptionToken string   `xml:"resumptionToken,attr,omitempty"`
}

type Error struct {
	XMLName xml.Name `xml:"error"`
	Code    string   `xml:"code,attr"`
	Value   string   `xml:",chardata"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Value)
}

type xmlIdentify struct {
	XMLName           xml.Name   `xml:"Identify"`
	RepositoryName    string     `xml:"repositoryName"`
	BaseURL           string     `xml:"baseURL"`
	ProtocolVersion   string     `xml:"protocolVersion"`
	AdminEmails       []string   `xml:"adminEmail"`
	Granularity       string     `xml:"granularity"`
	EarliestDatestamp string     `xml:"earliestDatestamp"`
	Compression       string     `xml:"compression,omitempty"`
	DeletedRecord     string     `xml:"deletedRecord"`
	Descriptions      []*Payload `xml:"description,omitempty"`
}

type xmlListMetadataFormats struct {
	XMLName         xml.Name         `xml:"ListMetadataFormats"`
	MetadataFormats []MetadataFormat `xml:"metadataFormat"`
}

type xmlListSets struct {
	XMLName         xml.Name         `xml:"ListSets"`
	Sets            []xmlSet         `xml:"set"`
	ResumptionToken *ResumptionToken `xml:"resumptionToken"`
}

type xmlGetRecord struct {
	XMLName xml.Name `xml:"GetRecord"`
	Record  *Record  `xml:"record"`
}

type xmlListIdentifiers struct {
	XMLName         xml.Name         `xml:"ListIdentifiers"`
	Headers         []*Header        `xml:"header"`
	ResumptionToken *ResumptionToken `xml:"resumptionToken"`
}

type xmlListRecords struct {
	XMLName         xml.Name         `xml:"ListRecords"`
	Records         []*Record        `xml:"record"`
	ResumptionToken *ResumptionToken `xml:"resumptionToken"`
}

// --- Public types used by RecordProvider ---

type MetadataFormat struct {
	MetadataPrefix    string `xml:"metadataPrefix"`
	Schema            string `xml:"schema"`
	MetadataNamespace string `xml:"metadataNamespace"`
}

type Set struct {
	Spec        string
	Name        string
	Description string // raw XML, optional
}

type xmlSet struct {
	SetSpec        string   `xml:"setSpec"`
	SetName        string   `xml:"setName"`
	SetDescription *Payload `xml:"setDescription,omitempty"`
}

type Header struct {
	Status     string   `xml:"status,attr,omitempty"`
	Identifier string   `xml:"identifier"`
	Datestamp  string   `xml:"datestamp"`
	SetSpecs   []string `xml:"setSpec"`
}

type Payload struct {
	XML string `xml:",innerxml"`
}

type Record struct {
	Header   *Header  `xml:"header"`
	Metadata *Payload `xml:"metadata"`
}

type ResumptionToken struct {
	CompleteListSize int    `xml:"completeListSize,attr,omitempty"`
	Value            string `xml:",chardata"`
	ExpirationDate   string `xml:"expirationDate,attr,omitempty"`
	Cursor           *int   `xml:"cursor,attr,omitempty"`
}

// --- Response writing ---

func (p *Provider) writeResponse(w http.ResponseWriter, req *oaiRequest) {
	xmlReq := xmlRequest{URL: req.baseURL}
	if len(req.errors) == 0 {
		xmlReq.Verb = req.verb
		xmlReq.MetadataPrefix = req.metadataPrefix
		xmlReq.Identifier = req.identifier
		xmlReq.Set = req.set
		xmlReq.From = req.from
		xmlReq.Until = req.until
		xmlReq.ResumptionToken = req.resumptionToken
	} else {
		xmlReq.Verb = req.verb
	}

	res := &xmlResponse{
		XmlnsXsi:          xmlnsXsi,
		XsiSchemaLocation: xsiSchemaLocation,
		ResponseDate:      time.Now().UTC().Format(time.RFC3339),
		Request:           xmlReq,
		Errors:            req.errors,
		Body:              req.body,
	}

	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(200)
	w.Write([]byte(xml.Header))
	if p.cfg.StyleSheet != "" {
		w.Write([]byte(`<?xml-stylesheet type="text/xsl" href="` + p.cfg.StyleSheet + `"?>` + "\n"))
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	enc.Encode(res)
}

func (p *Provider) handleError(w http.ResponseWriter, err error) {
	if p.cfg.ErrorHandler != nil {
		p.cfg.ErrorHandler(err)
	}
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

// --- Verb handlers ---

func identify(ctx context.Context, p *Provider, req *oaiRequest) error {
	t, err := p.cfg.RecordProvider.GetEarliestDatestamp(ctx)
	if err != nil {
		return err
	}

	var descriptions []*Payload
	for _, d := range p.cfg.Descriptions {
		descriptions = append(descriptions, &Payload{XML: d})
	}

	req.body = &xmlIdentify{
		RepositoryName:    p.cfg.RepositoryName,
		BaseURL:           p.cfg.BaseURL,
		ProtocolVersion:   "2.0",
		AdminEmails:       p.cfg.AdminEmails,
		Granularity:       p.cfg.Granularity,
		Compression:       p.cfg.Compression,
		DeletedRecord:     p.cfg.DeletedRecord,
		EarliestDatestamp: t.Format(p.dateFormat),
		Descriptions:      descriptions,
	}
	return nil
}

func listMetadataFormats(ctx context.Context, p *Provider, req *oaiRequest) error {
	if len(p.cfg.MetadataFormats) == 0 {
		req.addError(&Error{Code: "noMetadataFormats", Value: "there are no metadata formats available"})
		return nil
	}
	if req.identifier != "" {
		_, err := p.cfg.RecordProvider.GetRecord(ctx, req.identifier, p.cfg.MetadataFormats[0].MetadataPrefix)
		if err != nil {
			var oaiErr *Error
			if errors.As(err, &oaiErr) && oaiErr.Code == "idDoesNotExist" {
				req.addError(ErrIDDoesNotExist)
				return nil
			}
			if !errors.As(err, &oaiErr) {
				return err
			}
		}
	}
	req.body = &xmlListMetadataFormats{MetadataFormats: p.cfg.MetadataFormats}
	return nil
}

func listSets(ctx context.Context, p *Provider, req *oaiRequest) error {
	sp := p.setProvider()
	if sp == nil {
		req.addError(errNoSetHierarchy)
		return nil
	}
	sets, err := sp.ListSets(ctx)
	if err != nil {
		return err
	}
	if len(sets) == 0 {
		req.addError(errNoSetHierarchy)
		return nil
	}
	xmlSets := make([]xmlSet, len(sets))
	for i, s := range sets {
		xmlSets[i] = xmlSet{SetSpec: s.Spec, SetName: s.Name}
		if s.Description != "" {
			xmlSets[i].SetDescription = &Payload{XML: s.Description}
		}
	}
	req.body = &xmlListSets{Sets: xmlSets}
	return nil
}

func listIdentifiers(ctx context.Context, p *Provider, req *oaiRequest) error {
	if il, ok := p.cfg.RecordProvider.(IdentifierProvider); ok {
		page, err := il.ListIdentifiers(ctx, p.buildQuery(req))
		if err != nil {
			return p.handleProviderError(req, err)
		}
		if len(page.Headers) == 0 {
			req.addError(ErrNoRecordsMatch)
			return nil
		}
		var token *ResumptionToken
		if page.Cursor != "" {
			token = &ResumptionToken{Value: page.Cursor, CompleteListSize: page.Total}
		}
		req.body = &xmlListIdentifiers{Headers: page.Headers, ResumptionToken: token}
		return nil
	}

	page, err := p.cfg.RecordProvider.ListRecords(ctx, p.buildQuery(req))
	if err != nil {
		return p.handleProviderError(req, err)
	}
	if len(page.Records) == 0 {
		req.addError(ErrNoRecordsMatch)
		return nil
	}
	headers := make([]*Header, len(page.Records))
	for i, r := range page.Records {
		headers[i] = r.Header
	}
	var token *ResumptionToken
	if page.Cursor != "" {
		token = &ResumptionToken{Value: page.Cursor, CompleteListSize: page.Total}
	}
	req.body = &xmlListIdentifiers{Headers: headers, ResumptionToken: token}
	return nil
}

func doListRecords(ctx context.Context, p *Provider, req *oaiRequest) error {
	page, err := p.cfg.RecordProvider.ListRecords(ctx, p.buildQuery(req))
	if err != nil {
		return p.handleProviderError(req, err)
	}
	if len(page.Records) == 0 {
		req.addError(ErrNoRecordsMatch)
		return nil
	}
	var token *ResumptionToken
	if page.Cursor != "" {
		token = &ResumptionToken{Value: page.Cursor, CompleteListSize: page.Total}
	}
	req.body = &xmlListRecords{Records: page.Records, ResumptionToken: token}
	return nil
}

func getRecord(ctx context.Context, p *Provider, req *oaiRequest) error {
	rec, err := p.cfg.RecordProvider.GetRecord(ctx, req.identifier, req.metadataPrefix)
	if err != nil {
		return p.handleProviderError(req, err)
	}
	req.body = &xmlGetRecord{Record: rec}
	return nil
}

func (p *Provider) buildQuery(req *oaiRequest) Query {
	return Query{
		MetadataPrefix: req.metadataPrefix,
		Set:            req.set,
		From:           req.fromTime,
		Until:          req.untilTime,
		Cursor:         req.resumptionToken,
		Limit:          p.pageSize,
	}
}

func (p *Provider) handleProviderError(req *oaiRequest, err error) error {
	var oaiErr *Error
	if errors.As(err, &oaiErr) {
		req.addError(oaiErr)
		return nil
	}
	return err
}

// --- Validation middleware ---

func validateResumptionToken(ctx context.Context, p *Provider, req *oaiRequest) error {
	if req.resumptionToken != "" && (req.metadataPrefix != "" || req.set != "" || req.from != "" || req.until != "") {
		req.addError(errResumptiontokenExclusive)
	}
	return nil
}

func requireMetadataPrefix(ctx context.Context, p *Provider, req *oaiRequest) error {
	if req.resumptionToken != "" {
		return nil
	}
	if req.metadataPrefix == "" {
		req.addError(errMetadataPrefixMissing)
		return nil
	}
	if !p.formats[req.metadataPrefix] {
		req.addError(ErrCannotDisseminateFormat)
	}
	return nil
}

func requireIdentifier(ctx context.Context, p *Provider, req *oaiRequest) error {
	if req.identifier == "" {
		req.addError(errIdentifierMissing)
	}
	return nil
}

func validateSet(ctx context.Context, p *Provider, req *oaiRequest) error {
	if req.resumptionToken != "" || req.set == "" {
		return nil
	}
	sp := p.setProvider()
	if sp == nil {
		req.addError(errNoSetHierarchy)
		return nil
	}
	exists, err := sp.HasSet(ctx, req.set)
	if err != nil {
		return err
	}
	if !exists {
		req.addError(errSetDoesNotExist)
	}
	return nil
}

func validateFromUntil(ctx context.Context, p *Provider, req *oaiRequest) error {
	if req.resumptionToken != "" {
		return nil
	}
	if req.from != "" {
		f := req.from
		if p.cfg.Granularity == "YYYY-MM-DDThh:mm:ssZ" && len(f) == 10 {
			f += "T00:00:00Z"
		}
		if t, err := time.Parse(p.dateFormat, f); err == nil {
			req.from = f
			req.fromTime = t
		} else {
			req.addError(errFromInvalid)
		}
	}
	if req.until != "" {
		u := req.until
		if p.cfg.Granularity == "YYYY-MM-DDThh:mm:ssZ" && len(u) == 10 {
			u += "T00:00:00Z"
		}
		if t, err := time.Parse(p.dateFormat, u); err == nil {
			req.until = u
			req.untilTime = t
		} else {
			req.addError(errUntilInvalid)
		}
	}
	return nil
}
