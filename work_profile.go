package bbl

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed work_profiles.json
var workProfilesFile []byte

var WorkProfiles = map[string]map[string]*WorkProfile{}
var WorkKinds []string
var WorkSubkinds = map[string][]string{}

type WorkProfile struct {
	Identifiers         *CodeAttrProfile `json:"identifiers,omitempty"`
	Contributors        *AttrProfile     `json:"contributors,omitempty"`
	Rels                *AttrProfile     `json:"rels,omitempty"`
	Classifications     *CodeAttrProfile `json:"classifications,omitempty"`
	Titles              *AttrProfile     `json:"titles,omitempty"`
	Abstracts           *AttrProfile     `json:"abstracts,omitempty"`
	LaySummaries        *AttrProfile     `json:"lay_summaries,omitempty"`
	Keywords            *AttrProfile     `json:"keywords,omitempty"`
	Conference          *AttrProfile     `json:"conference,omitempty"`
	ArticleNumber       *AttrProfile     `json:"article_number,omitempty"`
	ReportNumber        *AttrProfile     `json:"report_number,omitempty"`
	Volume              *AttrProfile     `json:"volume,omitempty"`
	Issue               *AttrProfile     `json:"issue,omitempty"`
	IssueTitle          *AttrProfile     `json:"issue_title,omitempty"`
	Edition             *AttrProfile     `json:"edition,omitempty"`
	TotalPages          *AttrProfile     `json:"total_pages,omitempty"`
	Pages               *AttrProfile     `json:"pages,omitempty"`
	PlaceOfPublication  *AttrProfile     `json:"place_of_publication,omitempty"`
	Publisher           *AttrProfile     `json:"publisher,omitempty"`
	PublicationYear     *AttrProfile     `json:"publication_year,omitempty"`
	JournalTitle        *AttrProfile     `json:"journal_title,omitempty"`
	JournalAbbreviation *AttrProfile     `json:"journal_abbreviation,omitempty"`
	BookTitle           *AttrProfile     `json:"book_title,omitempty"`
	SeriesTitle         *AttrProfile     `json:"series_title,omitempty"`

	IdentifierSchemes     []string `json:"-"`
	ClassificationSchemes []string `json:"-"`
}

func LoadWorkProfile(rec *Work) error {
	if subKinds, ok := WorkProfiles[rec.Kind]; ok {
		if !ok {
			return fmt.Errorf("invalid work kind %s", rec.Kind)
		}
		p, ok := subKinds[rec.Subkind]
		if !ok {
			return fmt.Errorf("%s: invalid work sub kind %s", rec.Kind, rec.Subkind)
		}
		rec.Profile = p
	}
	return nil
}

func init() {
	var profiles []struct {
		Kind       string          `json:"kind"`
		RawProfile json.RawMessage `json:"profile"`
		Subkinds   []struct {
			Subkind    string          `json:"subkind"`
			RawProfile json.RawMessage `json:"profile"`
		} `json:"subkinds"`
	}
	if err := json.Unmarshal(workProfilesFile, &profiles); err != nil {
		panic(err)
	}

	for _, p := range profiles {
		var kp WorkProfile
		if err := json.Unmarshal(p.RawProfile, &kp); err != nil {
			panic(err)
		}
		if kp.Identifiers != nil {
			for _, scheme := range kp.Identifiers.Schemes {
				kp.IdentifierSchemes = append(kp.IdentifierSchemes, scheme.Scheme)
			}
		}
		if kp.Classifications != nil {
			for _, scheme := range kp.Classifications.Schemes {
				kp.ClassificationSchemes = append(kp.ClassificationSchemes, scheme.Scheme)
			}
		}
		WorkKinds = append(WorkKinds, p.Kind)
		WorkProfiles[p.Kind] = map[string]*WorkProfile{"": &kp}
		for _, pp := range p.Subkinds {
			var skp WorkProfile
			if err := json.Unmarshal(p.RawProfile, &skp); err != nil {
				panic(err)
			}
			if pp.RawProfile != nil {
				if err := json.Unmarshal(pp.RawProfile, &skp); err != nil {
					panic(err)
				}
			}
			if skp.Identifiers != nil {
				for _, scheme := range skp.Identifiers.Schemes {
					skp.IdentifierSchemes = append(skp.IdentifierSchemes, scheme.Scheme)
				}
			}
			if skp.Classifications != nil {
				for _, scheme := range skp.Classifications.Schemes {
					skp.ClassificationSchemes = append(skp.ClassificationSchemes, scheme.Scheme)
				}
			}
			WorkSubkinds[p.Kind] = append(WorkSubkinds[p.Kind], pp.Subkind)
			WorkProfiles[p.Kind][pp.Subkind] = &skp
		}
	}
}
