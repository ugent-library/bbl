package bbl

import (
	"encoding/json"
	"slices"
	"time"
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
	Contributors []WorkContributor `json:"contributors,omitempty"`
	Files        []WorkFile        `json:"files,omitempty"`
	Identifiers  []Code            `json:"identifiers,omitempty"`
	Kind         string            `json:"kind"`
	Permissions  []Permission      `json:"permissions,omitempty"` // TODO move to header?
	Profile      *WorkProfile      `json:"-"`
	Rels         []WorkRel         `json:"rels,omitempty"`
	Status       string            `json:"status"`
	Subkind      string            `json:"subkind,omitempty"`
	WorkAttrs
}

type WorkAttrs struct {
	Abstracts           []Text     `json:"abstracts,omitempty"`
	ArticleNumber       string     `json:"article_number,omitempty"`
	BookTitle           string     `json:"book_title,omitempty"`
	Classifications     []Code     `json:"classifications,omitempty"`
	Conference          Conference `json:"conference,omitzero"`
	Edition             string     `json:"edition,omitempty"`
	Issue               string     `json:"issue,omitempty"`
	IssueTitle          string     `json:"issue_title,omitempty"`
	JournalAbbreviation string     `json:"journal_abbreviation,omitempty"`
	JournalTitle        string     `json:"journal_title,omitempty"`
	Keywords            []string   `json:"keywords,omitempty"`
	LaySummaries        []Text     `json:"lay_summaries,omitempty"`
	Pages               Extent     `json:"pages,omitzero"`
	PlaceOfPublication  string     `json:"place_of_publication,omitempty"`
	PublicationYear     string     `json:"publication_year,omitempty"`
	Publisher           string     `json:"publisher,omitempty"`
	ReportNumber        string     `json:"report_number,omitempty"`
	SeriesTitle         string     `json:"series_title,omitempty"`
	Titles              []Text     `json:"titles,omitempty"`
	TotalPages          string     `json:"total_pages,omitempty"`
	Volume              string     `json:"volume,omitempty"`
}

type WorkContributor struct {
	PersonID string  `json:"person_id,omitempty"`
	Person   *Person `json:"person,omitempty"`
	WorkContributorAttrs
}

type WorkContributorAttrs struct {
	CreditRoles []string `json:"credit_roles,omitempty"`
	Name        string   `json:"name,omitempty"`
	GivenName   string   `json:"given_name,omitempty"`
	MiddleName  string   `json:"middle_name,omitempty"`
	FamilyName  string   `json:"family_name,omitempty"`
}

func (c *WorkContributor) GetName() string {
	if c.Name != "" {
		return c.Name
	}
	if c.Person != nil {
		return c.Person.Name
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

type WorkChange struct {
	RevID     string    `json:"rev_id"`
	CreatedAt time.Time `json:"created_at"`
	UserID    string    `json:"user_id,omitempty"`
	User      *User     `json:"user,omitempty"`
	Diff      WorkDiff  `json:"diff"`
}

type WorkDiff struct {
	Abstracts           *[]Text            `json:"abstracts,omitempty"`
	ArticleNumber       *string            `json:"article_number,omitempty"`
	BookTitle           *string            `json:"book_title,omitempty"`
	Classifications     *[]Code            `json:"classifications,omitempty"`
	Conference          *Conference        `json:"conference,omitempty"`
	Contributors        *[]WorkContributor `json:"contributors,omitempty"`
	Edition             *string            `json:"edition,omitempty"`
	Files               *[]WorkFile        `json:"files,omitempty"`
	Identifiers         *[]Code            `json:"identifiers,omitempty"`
	Issue               *string            `json:"issue,omitempty"`
	IssueTitle          *string            `json:"issue_title,omitempty"`
	JournalAbbreviation *string            `json:"journal_abbreviation,omitempty"`
	JournalTitle        *string            `json:"journal_title,omitempty"`
	Keywords            *[]string          `json:"keywords,omitempty"`
	Kind                *string            `json:"kind,omitempty"`
	LaySummaries        *[]Text            `json:"lay_summaries,omitempty"`
	Pages               *Extent            `json:"pages,omitempty"`
	Permissions         *[]Permission      `json:"permissions,omitempty"` // TODO should we include this in changes?
	PlaceOfPublication  *string            `json:"place_of_publication,omitempty"`
	PublicationYear     *string            `json:"publication_year,omitempty"`
	Publisher           *string            `json:"publisher,omitempty"`
	Rels                *[]WorkRel         `json:"rels,omitempty"`
	ReportNumber        *string            `json:"report_number,omitempty"`
	SeriesTitle         *string            `json:"series_title,omitempty"`
	Status              *string            `json:"status,omitempty"`
	SubKind             *string            `json:"subkind,omitempty"`
	Titles              *[]Text            `json:"titles,omitempty"`
	TotalPages          *string            `json:"total_pages,omitempty"`
	Volume              *string            `json:"volume,omitempty"`
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

func (rec *Work) Diff(rec2 *Work) *WorkDiff {
	diff := &WorkDiff{}

	if !slices.Equal(rec.Abstracts, rec2.Abstracts) {
		diff.Abstracts = &rec.Abstracts
	}
	if rec.ArticleNumber != rec2.ArticleNumber {
		diff.ArticleNumber = &rec.ArticleNumber
	}
	if rec.BookTitle != rec2.BookTitle {
		diff.BookTitle = &rec.BookTitle
	}
	if !slices.Equal(rec.Classifications, rec2.Classifications) {
		diff.Classifications = &rec.Classifications
	}
	if rec.Conference != rec2.Conference {
		diff.Conference = &rec.Conference
	}
	if !slices.EqualFunc(rec.Contributors, rec2.Contributors, func(c1, c2 WorkContributor) bool {
		return c1.PersonID == c2.PersonID &&
			slices.Equal(c1.CreditRoles, c2.CreditRoles) &&
			c1.Name == c2.Name &&
			c1.GivenName == c2.GivenName &&
			c1.MiddleName == c2.MiddleName &&
			c1.FamilyName == c2.FamilyName
	}) {
		diff.Contributors = &rec.Contributors
	}
	if rec.Edition != rec2.Edition {
		diff.Edition = &rec.Edition
	}
	if !slices.Equal(rec.Files, rec2.Files) {
		diff.Files = &rec.Files
	}
	if !slices.Equal(rec.Identifiers, rec2.Identifiers) {
		diff.Identifiers = &rec.Identifiers
	}
	if rec.Issue != rec2.Issue {
		diff.Issue = &rec.Issue
	}
	if rec.IssueTitle != rec2.IssueTitle {
		diff.IssueTitle = &rec.IssueTitle
	}
	if rec.JournalAbbreviation != rec2.JournalAbbreviation {
		diff.JournalAbbreviation = &rec.JournalAbbreviation
	}
	if rec.JournalTitle != rec2.JournalTitle {
		diff.JournalTitle = &rec.JournalTitle
	}
	if !slices.Equal(rec.Keywords, rec2.Keywords) {
		diff.Keywords = &rec.Keywords
	}
	if rec.Kind != rec2.Kind {
		diff.Kind = &rec.Kind
	}
	if !slices.Equal(rec.LaySummaries, rec2.LaySummaries) {
		diff.LaySummaries = &rec.LaySummaries
	}
	if rec.Pages != rec2.Pages {
		diff.Pages = &rec.Pages
	}
	if !slices.Equal(rec.Permissions, rec2.Permissions) { // TODO should we include this in changes?
		diff.Permissions = &rec.Permissions
	}
	if rec.PlaceOfPublication != rec2.PlaceOfPublication {
		diff.PlaceOfPublication = &rec.PlaceOfPublication
	}
	if rec.PublicationYear != rec2.PublicationYear {
		diff.PublicationYear = &rec.PublicationYear
	}
	if rec.Publisher != rec2.Publisher {
		diff.Publisher = &rec.Publisher
	}
	if !slices.EqualFunc(rec.Rels, rec2.Rels, func(r1, r2 WorkRel) bool {
		return r1.Kind == r2.Kind && r1.WorkID == r2.WorkID
	}) {
		diff.Rels = &rec.Rels
	}
	if rec.ReportNumber != rec2.ReportNumber {
		diff.ReportNumber = &rec.ReportNumber
	}
	if rec.SeriesTitle != rec2.SeriesTitle {
		diff.SeriesTitle = &rec.SeriesTitle
	}
	if rec.Status != rec2.Status {
		diff.Status = &rec.Status
	}
	if rec.Subkind != rec2.Subkind {
		diff.SubKind = &rec.Subkind
	}
	if !slices.Equal(rec.Titles, rec2.Titles) {
		diff.Titles = &rec.Titles
	}
	if rec.TotalPages != rec2.TotalPages {
		diff.TotalPages = &rec.TotalPages
	}
	if rec.Volume != rec2.Volume {
		diff.Volume = &rec.Volume
	}

	return diff
}

func (rec *Work) GetTitle() string {
	if len(rec.Titles) > 0 {
		return rec.Titles[0].Val
	}
	return ""
}

func (rec *Work) ContributorsWithCreditRole(role string) []WorkContributor {
	var s []WorkContributor
	for _, con := range rec.Contributors {
		if slices.Contains(con.CreditRoles, role) {
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
