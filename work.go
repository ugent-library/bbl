package bbl

import (
	"context"
	"slices"
	"time"
)

var WorkStatuses = []string{"suggestion", "draft", "public", "deleted"}

type Work struct {
	RecHeader
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
	Titles       []Text     `json:"titles,omitempty"`
	Abstracts    []Text     `json:"abstracts,omitempty"`
	LaySummaries []Text     `json:"lay_summaries,omitempty"`
	Keywords     []string   `json:"keywords,omitempty"`
	Conference   Conference `json:"conference,omitzero"`
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

func (rec *Work) Diff(rec2 *Work) map[string]any {
	changes := map[string]any{}
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
	return changes
}

func (rec *Work) Title() string {
	if len(rec.Attrs.Titles) > 0 {
		return rec.Attrs.Titles[0].Val
	}
	return ""
}

type WorkEncoder = func(context.Context, *Work) ([]byte, error)

type WorkRepresentation struct {
	WorkID    string    `json:"work_id"`
	Scheme    string    `json:"scheme"`
	Record    []byte    `json:"record"`
	UpdatedAt time.Time `json:"updated_at"`
}
