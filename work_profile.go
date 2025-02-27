package bbl

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"
)

//go:embed work_profiles.json
var workProfilesFile []byte

// TODO this shouldn't be global
var (
	workProfiles        = map[string]*WorkProfile{"work": {Kind: "work"}}
	visibleWorkProfiles = []*WorkProfile{}
)

func WorkProfiles() []*WorkProfile {
	return visibleWorkProfiles
}

func getWorkProfile(kind string) *WorkProfile {
	return workProfiles[kind]
}

func init() {
	var rawProfiles []json.RawMessage
	if err := json.Unmarshal(workProfilesFile, &rawProfiles); err != nil {
		panic(fmt.Errorf("loadWorkProfiles: load raw: %w", err))
	}

	for _, raw := range rawProfiles {
		p := &WorkProfile{json: raw}
		kind := gjson.GetBytes(raw, "kind").String()

		// merge parent profiles
		for i, c := range kind {
			if c != '.' {
				continue
			}
			parentKind := kind[:i]
			if parentProfile, ok := workProfiles[parentKind]; ok {
				if parentProfile.json != nil { // TODO because of default work profile above
					if err := json.Unmarshal([]byte(parentProfile.json), p); err != nil {
						panic(fmt.Errorf("loadWorkProfiles: merge parent: %w", err))
					}
				}
			}
		}

		// merge profile itself
		if err := json.Unmarshal(raw, p); err != nil {
			panic(fmt.Errorf("loadWorkProfiles: merge: %w", err))
		}

		// set implied values
		if p.Abstracts.Required {
			p.Abstracts.Use = true
		}
		if p.ArticleNumber.Required {
			p.ArticleNumber.Use = true
		}
		if len(p.Classifications.Schemes) > 0 {
			p.Classifications.Use = true
		}
		if p.Conference.Required {
			p.Conference.Use = true
		}
		if len(p.Contributors.CreditRoles) > 0 {
			p.Contributors.Use = true
		}
		if p.Edition.Required {
			p.Edition.Use = true
		}
		if len(p.Identifiers.Schemes) > 0 {
			p.Identifiers.Use = true
		}
		if p.Issue.Required {
			p.Issue.Use = true
		}
		if p.IssueTitle.Required {
			p.IssueTitle.Use = true
		}
		if p.Keywords.Required {
			p.Keywords.Use = true
		}
		if p.LaySummaries.Required {
			p.LaySummaries.Use = true
		}
		if p.Pages.Required {
			p.Pages.Use = true
		}
		if p.PlaceOfPublication.Required {
			p.PlaceOfPublication.Use = true
		}
		if p.Publisher.Required {
			p.Publisher.Use = true
		}
		if p.RelatedProjects.Required {
			p.RelatedProjects.Use = true
		}
		if len(p.RelatedWorks.Relations) > 0 {
			p.RelatedWorks.Use = true
		}
		if p.ReportNumber.Required {
			p.ReportNumber.Use = true
		}
		if p.Titles.Required {
			p.Titles.Use = true
		}
		if p.TotalPages.Required {
			p.TotalPages.Use = true
		}
		if p.Volume.Required {
			p.Volume.Use = true
		}

		// memoize
		p.classificationSchemes = make([]string, len(p.Classifications.Schemes))
		for i, s := range p.Classifications.Schemes {
			p.classificationSchemes[i] = s.Scheme
		}
		p.creditRoles = make([]string, len(p.Contributors.CreditRoles))
		for i, c := range p.Contributors.CreditRoles {
			p.creditRoles[i] = c.CreditRole
		}
		p.identifierSchemes = make([]string, len(p.Identifiers.Schemes))
		for i, s := range p.Identifiers.Schemes {
			p.identifierSchemes[i] = s.Scheme
		}
		p.relatedWorkRelations = make([]string, len(p.RelatedWorks.Relations))
		for i, r := range p.RelatedWorks.Relations {
			p.relatedWorkRelations[i] = r.Relation
		}

		workProfiles[p.Kind] = p
		visibleWorkProfiles = append(visibleWorkProfiles, p)
	}
}

type WorkProfileScheme struct {
	Scheme   string `json:"scheme"`
	Required bool   `json:"required"`
	Multiple bool   `json:"multiple"`
}

type WorkProfile struct {
	json []byte

	// memoize
	identifierSchemes     []string
	classificationSchemes []string
	creditRoles           []string
	relatedWorkRelations  []string

	Kind      string `json:"kind"`
	Abstracts struct {
		Use      bool `json:"use"`
		Required bool `json:"required"`
	} `json:"abstracts"`
	ArticleNumber struct {
		Use      bool `json:"use"`
		Required bool `json:"required"`
	} `json:"article_number"`
	Classifications struct {
		Use     bool                `json:"use"`
		Schemes []WorkProfileScheme `json:"schemes"`
	} `json:"classifications"`
	Conference struct {
		Use      bool `json:"use"`
		Required bool `json:"required"`
	} `json:"conference"`
	Contributors struct {
		Use         bool `json:"use"`
		CreditRoles []struct {
			CreditRole string `json:"credit_role"`
			Required   bool   `json:"required"`
		} `json:"credit_roles"`
	} `json:"contributors"`
	Edition struct {
		Use      bool `json:"use"`
		Required bool `json:"required"`
	} `json:"edition"`
	Identifiers struct {
		Use     bool                `json:"use"`
		Schemes []WorkProfileScheme `json:"schemes"`
	} `json:"identifiers"`
	Issue struct {
		Use      bool `json:"use"`
		Required bool `json:"required"`
	} `json:"issue"`
	IssueTitle struct {
		Use      bool `json:"use"`
		Required bool `json:"required"`
	} `json:"issue_title"`
	Keywords struct {
		Use      bool `json:"use"`
		Required bool `json:"required"`
	} `json:"keywords"`
	LaySummaries struct {
		Use      bool `json:"use"`
		Required bool `json:"required"`
	} `json:"lay_summaries"`
	Pages struct {
		Use      bool `json:"use"`
		Required bool `json:"required"`
	} `json:"pages"`
	PlaceOfPublication struct {
		Use      bool `json:"use"`
		Required bool `json:"required"`
	} `json:"place_of_publication"`
	Publisher struct {
		Use      bool `json:"use"`
		Required bool `json:"required"`
	} `json:"publisher"`
	RelatedProjects struct {
		Use      bool `json:"use"`
		Required bool `json:"required"`
	} `json:"related_projects"`
	RelatedWorks struct {
		Use       bool `json:"use"`
		Relations []struct {
			Relation string `json:"relation"`
			Required bool   `json:"required"`
		} `json:"relations"`
	} `json:"related_works"`
	ReportNumber struct {
		Use      bool `json:"use"`
		Required bool `json:"required"`
	} `json:"report_number"`
	Titles struct {
		Use      bool `json:"use"`
		Required bool `json:"required"`
	} `json:"titles"`
	TotalPages struct {
		Use      bool `json:"use"`
		Required bool `json:"required"`
	} `json:"total_pages"`
	Volume struct {
		Use      bool `json:"use"`
		Required bool `json:"required"`
	} `json:"volume"`
}

func (p *WorkProfile) UsesCreditRole(creditRole string) bool {
	for _, s := range p.Contributors.CreditRoles {
		if s.CreditRole == creditRole {
			return true
		}
	}
	return false
}

func (p *WorkProfile) CreditRoles() []string {
	return p.creditRoles
}

func (p *WorkProfile) UsesClassificationScheme(scheme string) bool {
	for _, s := range p.Classifications.Schemes {
		if s.Scheme == scheme {
			return true
		}
	}
	return false
}

func (p *WorkProfile) ClassificationSchemes() []string {
	return p.classificationSchemes
}

func (p *WorkProfile) UsesIdentifierScheme(scheme string) bool {
	for _, s := range p.Identifiers.Schemes {
		if s.Scheme == scheme {
			return true
		}
	}
	return false
}

func (p *WorkProfile) IdentifierSchemes() []string {
	return p.identifierSchemes
}

func (p *WorkProfile) UsesRelatedWorkRelation(relation string) bool {
	for _, r := range p.RelatedWorks.Relations {
		if r.Relation == relation {
			return true
		}
	}
	return false
}

func (p *WorkProfile) RelatedWorkRelations() []string {
	return p.relatedWorkRelations
}
