package bbl

import (
	"encoding/json"
	"slices"
)

const (
	SuggestionStatus = "suggestion"
	DraftStatus      = "draft"
	PublicStatus     = "public"
	DeletedStatus    = "deleted"
)

var WorkStatuses = []string{
	SuggestionStatus,
	DraftStatus,
	PublicStatus,
	DeletedStatus,
}

const (
	AuthorCreditRole     = "author"
	SupervisorCreditRole = "supervisor"
)

var CreditRoles = []string{
	AuthorCreditRole,
	SupervisorCreditRole,
}

type Work struct {
	RecHeader
	Permissions  []Permission      `json:"permissions,omitempty"` // TODO move to header?
	Profile      *WorkProfile      `json:"-"`
	Kind         string            `json:"kind"`
	Subkind      string            `json:"subkind,omitempty"`
	Status       string            `json:"status"`
	Identifiers  []Code            `json:"identifiers,omitempty"`
	Contributors []WorkContributor `json:"contributors,omitempty"`
	Files        []WorkFile        `json:"files,omitempty"`
	Rels         []WorkRel         `json:"rels,omitempty"`
	Attrs        WorkAttrs         `json:"attrs"`
}

type WorkAttrs struct {
	Classifications     []Code     `json:"classifications,omitempty"`
	Titles              []Text     `json:"titles,omitempty"`
	Abstracts           []Text     `json:"abstracts,omitempty"`
	LaySummaries        []Text     `json:"lay_summaries,omitempty"`
	Keywords            []string   `json:"keywords,omitempty"`
	Conference          Conference `json:"conference,omitzero"`
	ArticleNumber       string     `json:"article_number,omitempty"`
	ReportNumber        string     `json:"report_number,omitempty"`
	Volume              string     `json:"volume,omitempty"`
	Issue               string     `json:"issue,omitempty"`
	IssueTitle          string     `json:"issue_title,omitempty"`
	Edition             string     `json:"edition,omitempty"`
	TotalPages          string     `json:"total_pages,omitempty"`
	Pages               Extent     `json:"pages,omitzero"`
	PlaceOfPublication  string     `json:"place_of_publication,omitempty"`
	Publisher           string     `json:"publisher,omitempty"`
	PublicationYear     string     `json:"publication_year,omitempty"`
	JournalTitle        string     `json:"journal_title,omitempty"`
	JournalAbbreviation string     `json:"journal_abbreviation,omitempty"`
	BookTitle           string     `json:"book_title,omitempty"`
	SeriesTitle         string     `json:"series_title,omitempty"`
}

type WorkContributor struct {
	Attrs    WorkContributorAttrs `json:"attrs"`
	PersonID string               `json:"person_id,omitempty"`
	Person   *Person              `json:"person,omitempty"`
}

type WorkContributorAttrs struct {
	CreditRoles []string `json:"credit_roles,omitempty"`
	Name        string   `json:"name,omitempty"`
	GivenName   string   `json:"given_name,omitempty"`
	MiddleName  string   `json:"middle_name,omitempty"`
	FamilyName  string   `json:"family_name,omitempty"`
}

func (c *WorkContributor) GetName() string {
	if c.Attrs.Name != "" {
		return c.Attrs.Name
	}
	if c.Person != nil {
		return c.Person.Attrs.Name
	}
	return ""
}

type WorkFile struct {
	ObjectID    string `json:"object_id"`
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
}

type WorkRel struct {
	Kind   string `json:"kind"`
	WorkID string `json:"work_id"`
	Work   *Work  `json:"work,omitempty"`
}

func (rec *Work) Validate() error {
	return nil
}

func (rec *Work) Clone() (*Work, error) {
	clone := &Work{Profile: rec.Profile}
	b, err := json.Marshal(rec)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, clone); err != nil {
		return nil, err
	}
	return clone, nil
}

func (rec *Work) Diff(rec2 *Work) map[string]any {
	changes := map[string]any{}
	if !slices.Equal(rec.Permissions, rec2.Permissions) { // TODO should we include this in changes?
		changes["permissions"] = rec.Permissions
	}
	if rec.Kind != rec2.Kind {
		changes["kind"] = rec.Kind
	}
	if rec.Subkind != rec2.Subkind {
		changes["subkind"] = rec.Subkind
	}
	if rec.Status != rec2.Status {
		changes["status"] = rec.Status
	}
	if !slices.Equal(rec.Identifiers, rec2.Identifiers) {
		changes["identifiers"] = rec.Identifiers
	}
	if !slices.EqualFunc(rec.Contributors, rec2.Contributors, func(c1, c2 WorkContributor) bool {
		return c1.PersonID == c2.PersonID &&
			slices.Equal(c1.Attrs.CreditRoles, c2.Attrs.CreditRoles) &&
			c1.Attrs.Name == c2.Attrs.Name &&
			c1.Attrs.GivenName == c2.Attrs.GivenName &&
			c1.Attrs.MiddleName == c2.Attrs.MiddleName &&
			c1.Attrs.FamilyName == c2.Attrs.FamilyName
	}) {
		changes["contributors"] = rec.Contributors
	}
	if !slices.Equal(rec.Files, rec2.Files) {
		changes["files"] = rec.Files
	}
	if !slices.EqualFunc(rec.Rels, rec2.Rels, func(r1, r2 WorkRel) bool {
		return r1.Kind == r2.Kind && r1.WorkID == r2.WorkID
	}) {
		changes["rels"] = rec.Rels
	}
	if !slices.Equal(rec.Attrs.Classifications, rec2.Attrs.Classifications) {
		changes["classifications"] = rec.Attrs.Classifications
	}
	if !slices.Equal(rec.Attrs.Titles, rec2.Attrs.Titles) {
		changes["titles"] = rec.Attrs.Titles
	}
	if !slices.Equal(rec.Attrs.Abstracts, rec2.Attrs.Abstracts) {
		changes["abstracts"] = rec.Attrs.Abstracts
	}
	if !slices.Equal(rec.Attrs.LaySummaries, rec2.Attrs.LaySummaries) {
		changes["lay_summaries"] = rec.Attrs.LaySummaries
	}
	if !slices.Equal(rec.Attrs.Keywords, rec2.Attrs.Keywords) {
		changes["keywords"] = rec.Attrs.Keywords
	}
	if rec.Attrs.Conference != rec2.Attrs.Conference {
		changes["conference"] = rec.Attrs.Conference
	}
	if rec.Attrs.ArticleNumber != rec2.Attrs.ArticleNumber {
		changes["article_number"] = rec.Attrs.ArticleNumber
	}
	if rec.Attrs.ReportNumber != rec2.Attrs.ReportNumber {
		changes["report_number"] = rec.Attrs.ReportNumber
	}
	if rec.Attrs.Volume != rec2.Attrs.Volume {
		changes["volume"] = rec.Attrs.Volume
	}
	if rec.Attrs.Issue != rec2.Attrs.Issue {
		changes["issue"] = rec.Attrs.Issue
	}
	if rec.Attrs.IssueTitle != rec2.Attrs.IssueTitle {
		changes["issue_title"] = rec.Attrs.IssueTitle
	}
	if rec.Attrs.Edition != rec2.Attrs.Edition {
		changes["edition"] = rec.Attrs.Edition
	}
	if rec.Attrs.TotalPages != rec2.Attrs.TotalPages {
		changes["total_pages"] = rec.Attrs.TotalPages
	}
	if rec.Attrs.Pages != rec2.Attrs.Pages {
		changes["pages"] = rec.Attrs.Pages
	}
	if rec.Attrs.PlaceOfPublication != rec2.Attrs.PlaceOfPublication {
		changes["place_of_publication"] = rec.Attrs.PlaceOfPublication
	}
	if rec.Attrs.Publisher != rec2.Attrs.Publisher {
		changes["publisher"] = rec.Attrs.Publisher
	}
	if rec.Attrs.PublicationYear != rec2.Attrs.PublicationYear {
		changes["publication_year"] = rec.Attrs.PublicationYear
	}
	if rec.Attrs.JournalTitle != rec2.Attrs.JournalTitle {
		changes["journal_title"] = rec.Attrs.JournalTitle
	}
	if rec.Attrs.JournalAbbreviation != rec2.Attrs.JournalAbbreviation {
		changes["journal_abbreviation"] = rec.Attrs.JournalAbbreviation
	}
	if rec.Attrs.BookTitle != rec2.Attrs.BookTitle {
		changes["book_title"] = rec.Attrs.BookTitle
	}
	if rec.Attrs.SeriesTitle != rec2.Attrs.SeriesTitle {
		changes["series_title"] = rec.Attrs.SeriesTitle
	}
	return changes
}

func (rec *Work) Title() string {
	if len(rec.Attrs.Titles) > 0 {
		return rec.Attrs.Titles[0].Val
	}
	return ""
}

func (rec *Work) ContributorsWithCreditRole(role string) []WorkContributor {
	var s []WorkContributor
	for _, con := range rec.Contributors {
		if slices.Contains(con.Attrs.CreditRoles, role) {
			s = append(s, con)
		}
	}
	return s
}

func (rec *Work) Authors() []WorkContributor {
	return rec.ContributorsWithCreditRole(AuthorCreditRole)
}

func (rec *Work) Supervisors() []WorkContributor {
	return rec.ContributorsWithCreditRole(SupervisorCreditRole)
}
