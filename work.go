package bbl

import (
	"slices"
	"time"
)

type Work struct {
	Profile   *WorkProfile `json:"-"`
	ID        string       `json:"id,omitempty"`
	Kind      string       `json:"kind"`
	SubKind   string       `json:"sub_kind,omitempty"`
	Attrs     WorkAttrs    `json:"attrs"`
	Rels      []WorkRel    `json:"rels,omitempty"`
	CreatedAt time.Time    `json:"created_at,omitzero"`
	UpdatedAt time.Time    `json:"updated_at,omitzero"`
}

type WorkAttrs struct {
	Identifiers  []Code     `json:"identifiers,omitempty"`
	Titles       []Text     `json:"titles,omitempty"`
	Abstracts    []Text     `json:"abstracts,omitempty"`
	LaySummaries []Text     `json:"lay_summaries,omitempty"`
	Keywords     []string   `json:"keywords,omitempty"`
	Conference   Conference `json:"conference,omitzero"`
}

type WorkRel struct {
	ID     string `json:"id,omitempty"`
	Kind   string `json:"kind"`
	WorkID string `json:"work_id"`
	Work   *Work  `json:"work,omitempty"`
}

func (rec *Work) Diff(otherRec *Work) map[string]any {
	changes := map[string]any{}
	if rec.Kind != otherRec.Kind {
		changes["kind"] = rec.Kind
	}
	if rec.SubKind != otherRec.SubKind {
		changes["sub_kind"] = rec.SubKind
	}
	if !slices.Equal(rec.Attrs.Identifiers, otherRec.Attrs.Identifiers) {
		changes["identifiers"] = rec.Attrs.Identifiers
	}
	if !slices.Equal(rec.Attrs.Titles, otherRec.Attrs.Titles) {
		changes["titles"] = rec.Attrs.Titles
	}
	if !slices.Equal(rec.Attrs.Abstracts, otherRec.Attrs.Abstracts) {
		changes["abstracts"] = rec.Attrs.Abstracts
	}
	if !slices.Equal(rec.Attrs.LaySummaries, otherRec.Attrs.LaySummaries) {
		changes["lay_summaries"] = rec.Attrs.LaySummaries
	}
	if !slices.Equal(rec.Attrs.Keywords, otherRec.Attrs.Keywords) {
		changes["keywords"] = rec.Attrs.Keywords
	}
	if rec.Attrs.Conference != otherRec.Attrs.Conference {
		changes["conference"] = rec.Attrs.Conference
	}
	if !slices.EqualFunc(rec.Rels, otherRec.Rels, func(rel, otherRel WorkRel) bool {
		return rel.Kind == otherRel.Kind && rel.WorkID == otherRel.WorkID
	}) {
		changes["rels"] = rec.Rels
	}
	return changes
}
