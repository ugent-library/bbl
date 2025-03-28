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
var WorkSubKinds = map[string][]string{}

func LoadWorkProfile(rec *Work) error {
	if subKinds, ok := WorkProfiles[rec.Kind]; ok {
		if !ok {
			return fmt.Errorf("invalid work kind %s", rec.Kind)
		}
		p, ok := subKinds[rec.SubKind]
		if !ok {
			return fmt.Errorf("%s: invalid work sub kind %s", rec.Kind, rec.SubKind)
		}
		rec.Profile = p
	}
	return nil
}

func init() {
	var profiles []struct {
		Kind       string          `json:"kind"`
		RawProfile json.RawMessage `json:"profile"`
		SubKinds   []struct {
			SubKind    string          `json:"sub_kind"`
			RawProfile json.RawMessage `json:"profile"`
		} `json:"sub_kinds"`
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
		WorkKinds = append(WorkKinds, p.Kind)
		WorkProfiles[p.Kind] = map[string]*WorkProfile{"": &kp}
		for _, pp := range p.SubKinds {
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
			WorkSubKinds[p.Kind] = append(WorkSubKinds[p.Kind], pp.SubKind)
			WorkProfiles[p.Kind][pp.SubKind] = &skp
		}
	}
}

type WorkProfile struct {
	Identifiers  *CodeAttrProfile `json:"identifiers,omitempty"`
	Titles       *AttrProfile     `json:"titles,omitempty"`
	Abstracts    *AttrProfile     `json:"abstracts,omitempty"`
	LaySummaries *AttrProfile     `json:"lay_summaries,omitempty"`
	Keywords     *AttrProfile     `json:"keywords,omitempty"`
	Conference   *AttrProfile     `json:"conference,omitempty"`
	Contributors *AttrProfile     `json:"contributors,omitempty"`
	Rels         *AttrProfile     `json:"rels,omitempty"`

	IdentifierSchemes []string `json:"-"`
}
