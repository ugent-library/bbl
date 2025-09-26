package orcid

import (
	"encoding/xml"
	"time"
)

type Address struct {
	ElementSummary
	Country string `xml:"http://www.orcid.org/ns/address country"`
}

type Addresses struct {
	LastModifiedDate *DateTime `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
	Address          []Address `xml:"http://www.orcid.org/ns/address address,omitempty"`
	Path             string    `xml:"path,attr,omitempty"`
}

// TODO unfinished
type AffiliationGroup struct {
	LastModifiedDate   *DateTime            `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
	ExternalIds        ExternalIds          `xml:"http://www.orcid.org/ns/common external-ids"`
	DistinctionSummary []AffiliationSummary `xml:"http://www.orcid.org/ns/distinction distinction-summary,omitempty"`
	EducationSummary   []AffiliationSummary `xml:"http://www.orcid.org/ns/education education-summary,omitempty"`
	EmploymentSummary  []AffiliationSummary `xml:"http://www.orcid.org/ns/employment employment-summary,omitempty"`
	MembershipSummary  []AffiliationSummary `xml:"http://www.orcid.org/ns/membership membership-summary,omitempty"`
	ServiceSummary     []AffiliationSummary `xml:"http://www.orcid.org/ns/service service-summary,omitempty"`
}

type AffiliationSummary struct {
	ElementSummary
	DepartmentName string       `xml:"http://www.orcid.org/ns/common department-name,omitempty"`
	RoleTitle      string       `xml:"http://www.orcid.org/ns/common role-title,omitempty"`
	StartDate      *FuzzyDate   `xml:"http://www.orcid.org/ns/common start-date,omitempty"`
	EndDate        *FuzzyDate   `xml:"http://www.orcid.org/ns/common end-date,omitempty"`
	Organization   Organization `xml:"http://www.orcid.org/ns/common organization"`
	Url            string       `xml:"http://www.orcid.org/ns/common url,omitempty"`
	ExternalIds    *ExternalIds `xml:"http://www.orcid.org/ns/common external-ids,omitempty"`
}

type Biography struct {
	CreatedDate      *DateTime `xml:"http://www.orcid.org/ns/common created-date,omitempty"`
	LastModifiedDate *DateTime `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
	Content          string    `xml:"http://www.orcid.org/ns/personal-details content"`
	Visibility       string    `xml:"visibility,attr,omitempty"`
	Path             string    `xml:"path,attr,omitempty"`
}

type ClientId struct {
	Uri  string `xml:"http://www.orcid.org/ns/common uri,omitempty"`
	Path string `xml:"http://www.orcid.org/ns/common path,omitempty"`
	Host string `xml:"http://www.orcid.org/ns/common host,omitempty"`
}

type DisambiguatedOrganization struct {
	DisambiguatedOrganizationIdentifier string `xml:"http://www.orcid.org/ns/common disambiguated-organization-identifier"`
	DisambiguationSource                string `xml:"http://www.orcid.org/ns/common disambiguation-source"`
}

type Distinctions struct {
	LastModifiedDate *DateTime          `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
	AffiliationGroup []AffiliationGroup `xml:"http://www.orcid.org/ns/activities affiliation-group,omitempty"`
	Path             string             `xml:"path,attr,omitempty"`
}

type Educations struct {
	LastModifiedDate *DateTime          `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
	AffiliationGroup []AffiliationGroup `xml:"http://www.orcid.org/ns/activities affiliation-group,omitempty"`
	Path             string             `xml:"path,attr,omitempty"`
}

type ElementSummary struct {
	CreatedDate      *DateTime `xml:"http://www.orcid.org/ns/common created-date,omitempty"`
	LastModifiedDate *DateTime `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
	Source           *Source   `xml:"http://www.orcid.org/ns/common source,omitempty"`
	PutCode          int       `xml:"put-code,attr,omitempty"`
	Visibility       string    `xml:"visibility,attr"`
	DisplayIndex     string    `xml:"display-index,attr,omitempty"`
	Path             string    `xml:"path,attr,omitempty"`
}

type Email struct {
	ElementSummary
	Email    string `xml:"http://www.orcid.org/ns/email email"`
	Primary  bool   `xml:"primary,attr"`
	Verified bool   `xml:"verified,attr"`
}

type Emails struct {
	LastModifiedDate *DateTime `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
	Email            []Email   `xml:"http://www.orcid.org/ns/email email,omitempty"`
	Path             string    `xml:"path,attr,omitempty"`
}

type Employments struct {
	LastModifiedDate *DateTime          `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
	AffiliationGroup []AffiliationGroup `xml:"http://www.orcid.org/ns/activities affiliation-group,omitempty"`
	Path             string             `xml:"path,attr,omitempty"`
}

type ExternalId struct {
	ElementSummary
	ExternalIdType            string          `xml:"http://www.orcid.org/ns/common external-id-type"`
	ExternalIdValue           string          `xml:"http://www.orcid.org/ns/common external-id-value"`
	ExternalIdNormalized      TransientString `xml:"http://www.orcid.org/ns/common external-id-normalized,omitempty"`
	ExternalIdNormalizedError TransientError  `xml:"http://www.orcid.org/ns/common external-id-normalized-error,omitempty"`
	ExternalIdUrl             string          `xml:"http://www.orcid.org/ns/common external-id-url,omitempty"`
	ExternalIdRelationship    string          `xml:"http://www.orcid.org/ns/common external-id-relationship,omitempty"`
}

type ExternalIds struct {
	ExternalId []ExternalId `xml:"http://www.orcid.org/ns/common external-id,omitempty"`
}

type FuzzyDate struct {
	Year  int `xml:"http://www.orcid.org/ns/common year"`
	Month int `xml:"http://www.orcid.org/ns/common month,omitempty"`
	Day   int `xml:"http://www.orcid.org/ns/common day,omitempty"`
}

type Keyword struct {
	ElementSummary
	Content string `xml:"http://www.orcid.org/ns/keyword content"`
}

type Keywords struct {
	LastModifiedDate *DateTime `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
	Keyword          []Keyword `xml:"http://www.orcid.org/ns/keyword keyword,omitempty"`
	Path             string    `xml:"path,attr,omitempty"`
}

type Memberships struct {
	LastModifiedDate *DateTime          `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
	AffiliationGroup []AffiliationGroup `xml:"http://www.orcid.org/ns/activities affiliation-group,omitempty"`
	Path             string             `xml:"path,attr,omitempty"`
}

type OrcidId struct {
	Uri  string `xml:"http://www.orcid.org/ns/common uri,omitempty"`
	Path string `xml:"http://www.orcid.org/ns/common path,omitempty"`
	Host string `xml:"http://www.orcid.org/ns/common host,omitempty"`
}

type Organization struct {
	Name                      string                     `xml:"http://www.orcid.org/ns/common name"`
	Address                   OrganizationAddress        `xml:"http://www.orcid.org/ns/common address"`
	DisambiguatedOrganization *DisambiguatedOrganization `xml:"http://www.orcid.org/ns/common disambiguated-organization,omitempty"`
}

type OrganizationAddress struct {
	City    string `xml:"http://www.orcid.org/ns/common city"`
	Region  string `xml:"http://www.orcid.org/ns/common region,omitempty"`
	Country string `xml:"http://www.orcid.org/ns/common country"`
}

type Services struct {
	LastModifiedDate *DateTime          `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
	AffiliationGroup []AffiliationGroup `xml:"http://www.orcid.org/ns/activities affiliation-group,omitempty"`
	Path             string             `xml:"path,attr,omitempty"`
}

type Source struct {
	SourceOrcid             *OrcidId  `xml:"http://www.orcid.org/ns/common source-orcid,omitempty"`
	SourceClientId          *ClientId `xml:"http://www.orcid.org/ns/common source-client-id,omitempty"`
	SourceName              string    `xml:"http://www.orcid.org/ns/common source-name,omitempty"`
	AssertionOriginOrcid    *OrcidId  `xml:"http://www.orcid.org/ns/common assertion-origin-orcid,omitempty"`
	AssertionOriginClientId *ClientId `xml:"http://www.orcid.org/ns/common assertion-origin-client-id,omitempty"`
	AssertionOriginName     string    `xml:"http://www.orcid.org/ns/common assertion-origin-name,omitempty"`
}

type TransientError struct {
	ErrorCode    string `xml:"http://www.orcid.org/ns/common error-code,omitempty"`
	ErrorMessage string `xml:"http://www.orcid.org/ns/common error-message,omitempty"`
	Transient    bool   `xml:"transient,attr"`
}

type TransientString struct {
	Value     string `xml:",chardata"`
	Transient bool   `xml:"transient,attr"`
}

type WorkGroup struct {
	LastModifiedDate *DateTime     `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
	ExternalIds      ExternalIds   `xml:"http://www.orcid.org/ns/common external-ids"`
	WorkSummary      []WorkSummary `xml:"http://www.orcid.org/ns/work work-summary,omitempty"`
}

type Works struct {
	LastModifiedDate *DateTime   `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
	Group            []WorkGroup `xml:"http://www.orcid.org/ns/activities group,omitempty"`
	Path             string      `xml:"path,attr,omitempty"`
}

type WorkSummary struct {
	ElementSummary
	Title           WorkTitle   `xml:"http://www.orcid.org/ns/work title"`
	ExternalIds     ExternalIds `xml:"http://www.orcid.org/ns/common external-ids"`
	Url             string      `xml:"http://www.orcid.org/ns/common url,omitempty"`
	Type            string      `xml:"http://www.orcid.org/ns/work type"`
	PublicationDate *FuzzyDate  `xml:"http://www.orcid.org/ns/common publication-date,omitempty"`
	JournalTitle    string      `xml:"http://www.orcid.org/ns/common journal-title,omitempty"`
}

type WorkTitle struct {
	Title           string `xml:"http://www.orcid.org/ns/common title"`
	Subtitle        string `xml:"http://www.orcid.org/ns/common subtitle,omitempty"`
	TranslatedTitle string `xml:"http://www.orcid.org/ns/common translated-title,omitempty"`
}

type DateTime struct {
	time.Time
}

func (dt *DateTime) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if dt.Time.IsZero() {
		return nil
	}
	v := dt.Format("2006-01-02T15:04:05.999999999Z07:00")
	return e.EncodeElement(v, start)
}

func (dt *DateTime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var v string
	d.DecodeElement(&v, &start)
	t, err := time.Parse("2006-01-02T15:04:05.999999999", v)
	if _, ok := err.(*time.ParseError); ok {
		t, err = time.Parse("2006-01-02T15:04:05.999999999Z07:00", v)
	}
	if err != nil {
		return err
	}
	*dt = DateTime{t}
	return nil
}

// type ActivitiesSummary struct {
// 	LastModifiedDate  string            `xml:"http://www.orcid.org/ns/activities last-modified-date,omitempty"`
// 	Distinctions      Distinctions      `xml:"http://www.orcid.org/ns/activities distinctions,omitempty"`
// 	Educations        Educations        `xml:"http://www.orcid.org/ns/activities educations,omitempty"`
// 	Employments       Employments       `xml:"http://www.orcid.org/ns/activities employments,omitempty"`
// 	Fundings          Fundings          `xml:"http://www.orcid.org/ns/activities fundings,omitempty"`
// 	InvitedPositions  InvitedPositions  `xml:"http://www.orcid.org/ns/activities invited-positions,omitempty"`
// 	Memberships       Memberships       `xml:"http://www.orcid.org/ns/activities memberships,omitempty"`
// 	PeerReviews       PeerReviews       `xml:"http://www.orcid.org/ns/activities peer-reviews,omitempty"`
// 	Qualifications    Qualifications    `xml:"http://www.orcid.org/ns/activities qualifications,omitempty"`
// 	ResearchResources ResearchResources `xml:"http://www.orcid.org/ns/activities research-resources,omitempty"`
// 	Services          Services          `xml:"http://www.orcid.org/ns/activities services,omitempty"`
// 	Works             Works             `xml:"http://www.orcid.org/ns/activities works,omitempty"`
// 	Path              string            `xml:"path,attr,omitempty"`
// }

// type Address struct {
// 	CreatedDate      string `xml:"http://www.orcid.org/ns/common created-date,omitempty"`
// 	LastModifiedDate string `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
// 	Source           string `xml:"http://www.orcid.org/ns/common source,omitempty"`
// 	Country          string `xml:"http://www.orcid.org/ns/common country"`
// 	PutCode          int    `xml:"put-code,attr,omitempty"`
// 	Visibility       string `xml:"visibility,attr,omitempty"`
// 	DisplayIndex     string `xml:"display-index,attr,omitempty"`
// 	Path             string `xml:"path,attr,omitempty"`
// }

// type Addresses struct {
// 	LastModifiedDate string    `xml:"http://www.orcid.org/ns/address last-modified-date,omitempty"`
// 	Address          []Address `xml:"http://www.orcid.org/ns/address address,omitempty"`
// 	Path             string    `xml:"path,attr,omitempty"`
// }

// type Affiliation struct {
// 	CreatedDate      string       `xml:"http://www.orcid.org/ns/common created-date,omitempty"`
// 	LastModifiedDate string       `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
// 	Source           string       `xml:"http://www.orcid.org/ns/common source,omitempty"`
// 	DepartmentName   string       `xml:"http://www.orcid.org/ns/common department-name,omitempty"`
// 	RoleTitle        string       `xml:"http://www.orcid.org/ns/common role-title,omitempty"`
// 	StartDate        string       `xml:"http://www.orcid.org/ns/common start-date,omitempty"`
// 	EndDate          string       `xml:"http://www.orcid.org/ns/common end-date,omitempty"`
// 	Organization     Organization `xml:"http://www.orcid.org/ns/common organization,omitempty"`
// 	Url              string       `xml:"http://www.orcid.org/ns/common url,omitempty"`
// 	ExternalIds      string       `xml:"http://www.orcid.org/ns/common external-ids,omitempty"`
// 	PutCode          int          `xml:"put-code,attr,omitempty"`
// 	Visibility       string       `xml:"visibility,attr,omitempty"`
// 	DisplayIndex     string       `xml:"display-index,attr,omitempty"`
// 	Path             string       `xml:"path,attr,omitempty"`
// }

// // The funding amount.
// type Amount struct {
// 	Value        string `xml:",chardata"`
// 	CurrencyCode string `xml:"currency-code,attr"`
// }

// // Utilitary schema that allow the creation of multiple works in a single request
// type Bulk struct {
// 	Work  string `xml:"http://www.orcid.org/ns/bulk work,omitempty"`
// 	Error string `xml:"http://www.orcid.org/ns/bulk error,omitempty"`
// }

// // Container for a work citation. Citations may be
// // fielded (e.g., RIS, BibTeX - preferred citation type), or may be
// // textual (APA, MLA, Chicago, etc.) The required work-citation-type
// // element indicates the format of the citation.
// type Citation struct {
// 	CitationType  string `xml:"http://www.orcid.org/ns/work citation-type,omitempty"`
// 	CitationValue string `xml:"http://www.orcid.org/ns/work citation-value"`
// }

// A collaborator or other contributor to a work or
// other orcid-activity
// type Contributor struct {
// 	ContributorOrcId      string                `xml:"http://www.orcid.org/ns/work contributor-orcid,omitempty"`
// 	CreditName            string                `xml:"http://www.orcid.org/ns/common credit-name,omitempty"`
// 	ContributorEmail      string                `xml:"http://www.orcid.org/ns/work contributor-email,omitempty"`
// 	ContributorAttributes ContributorAttributes `xml:"http://www.orcid.org/ns/work contributor-attributes,omitempty"`
// }

// // Provides detail of the nature of the contribution
// // by the collaborator or other contirbutor.
// type ContributorAttributes struct {
// 	ContributorSequence string `xml:"http://www.orcid.org/ns/work contributor-sequence,omitempty"`
// 	ContributorRole     string `xml:"http://www.orcid.org/ns/common contributor-role,omitempty"`
// }

// // Container for the contributors of a funding.
// type Contributors struct {
// 	Contributor []Contributor `xml:"http://www.orcid.org/ns/funding contributor,omitempty"`
// }

// // Country represented by its ISO 3611 code. The
// // visibility attribute (private, limited or public) can be set at
// // record creation, and indicates who can see this section of
// // information.
// type Country struct {
// 	Value      string `xml:",chardata"`
// 	Visibility string `xml:"visibility,attr,omitempty"`
// }

// type CreditName struct {
// 	Value string `xml:",chardata"`
// }

// type EducationQualification struct {
// 	PutCode          int64  `xml:"http://www.orcid.org/ns/summary put-code"`
// 	Type             string `xml:"http://www.orcid.org/ns/summary type"`
// 	OrganizationName string `xml:"http://www.orcid.org/ns/common organization-name"`
// 	Role             string `xml:"http://www.orcid.org/ns/common role"`
// 	Url              string `xml:"http://www.orcid.org/ns/summary url,omitempty"`
// 	StartDate        string `xml:"http://www.orcid.org/ns/summary start-date,omitempty"`
// 	EndDate          string `xml:"http://www.orcid.org/ns/summary end-date,omitempty"`
// 	Validated        bool   `xml:"http://www.orcid.org/ns/summary validated"`
// }

// type EducationQualifications struct {
// 	Count                  int      `xml:"http://www.orcid.org/ns/summary count"`
// 	EducationQualification []string `xml:"http://www.orcid.org/ns/summary education-qualification"`
// }

// type EmailDomain struct {
// 	Value            string    `xml:"http://www.orcid.org/ns/common value"`
// 	VerificationDate time.Time `xml:"http://www.orcid.org/ns/summary verification-date,omitempty"`
// 	CreatedDate      string    `xml:"http://www.orcid.org/ns/summary created-date,omitempty"`
// 	LastModifiedDate string    `xml:"http://www.orcid.org/ns/summary last-modified-date,omitempty"`
// }

// type EmailDomains struct {
// 	Count       int      `xml:"http://www.orcid.org/ns/summary count"`
// 	EmailDomain []string `xml:"http://www.orcid.org/ns/summary email-domain"`
// }

// type Employment struct {
// 	PutCode          int64  `xml:"http://www.orcid.org/ns/summary put-code"`
// 	Type             string `xml:"http://www.orcid.org/ns/summary type,omitempty"`
// 	OrganizationName string `xml:"http://www.orcid.org/ns/common organization-name"`
// 	Role             string `xml:"http://www.orcid.org/ns/common role"`
// 	Url              string `xml:"http://www.orcid.org/ns/summary url,omitempty"`
// 	StartDate        string `xml:"http://www.orcid.org/ns/summary start-date,omitempty"`
// 	EndDate          string `xml:"http://www.orcid.org/ns/summary end-date,omitempty"`
// 	Validated        bool   `xml:"http://www.orcid.org/ns/summary validated"`
// }

// A single expanded search result when performing a
// search on the
// ORCID Registry.
// type ExpandedResult struct {
// 	OrcidId         string   `xml:"http://www.orcid.org/ns/expanded-search orcid-id"`
// 	GivenNames      string   `xml:"http://www.orcid.org/ns/expanded-search given-names,omitempty"`
// 	FamilyNames     string   `xml:"http://www.orcid.org/ns/expanded-search family-names,omitempty"`
// 	CreditName      string   `xml:"http://www.orcid.org/ns/expanded-search credit-name,omitempty"`
// 	OtherName       []string `xml:"http://www.orcid.org/ns/expanded-search other-name,omitempty"`
// 	Email           []string `xml:"http://www.orcid.org/ns/expanded-search email,omitempty"`
// 	InstitutionName []string `xml:"http://www.orcid.org/ns/expanded-search institution-name,omitempty"`
// }

// The container element for the results when
// performing a search on the ORCID Registry. the num-found attribute
// indicates the number of successful matches.
// type ExpandedSearch struct {
// 	ExpandedResult []ExpandedResult `xml:"http://www.orcid.org/ns/expanded-search expanded-result,omitempty"`
// 	NumFound       int              `xml:"num-found,attr,omitempty"`
// }

// type Funding struct {
// 	CreatedDate             string       `xml:"http://www.orcid.org/ns/common created-date,omitempty"`
// 	LastModifiedDate        string       `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
// 	Source                  string       `xml:"http://www.orcid.org/ns/common source,omitempty"`
// 	Type                    string       `xml:"http://www.orcid.org/ns/funding type"`
// 	OrganizationDefinedType string       `xml:"http://www.orcid.org/ns/common organization-defined-type,omitempty"`
// 	Title                   FundingTitle `xml:"http://www.orcid.org/ns/funding title,omitempty"`
// 	ShortDescription        string       `xml:"http://www.orcid.org/ns/common short-description,omitempty"`
// 	Amount                  Amount       `xml:"http://www.orcid.org/ns/common amount,omitempty"`
// 	Url                     string       `xml:"http://www.orcid.org/ns/funding url,omitempty"`
// 	StartDate               string       `xml:"http://www.orcid.org/ns/funding start-date,omitempty"`
// 	EndDate                 string       `xml:"http://www.orcid.org/ns/funding end-date,omitempty"`
// 	ExternalIds             string       `xml:"http://www.orcid.org/ns/funding external-ids,omitempty"`
// 	Contributors            Contributors `xml:"http://www.orcid.org/ns/funding contributors,omitempty"`
// 	Organization            string       `xml:"http://www.orcid.org/ns/funding organization"`
// 	PutCode                 int          `xml:"put-code,attr,omitempty"`
// 	Visibility              string       `xml:"visibility,attr,omitempty"`
// 	DisplayIndex            string       `xml:"display-index,attr,omitempty"`
// 	Path                    string       `xml:"path,attr,omitempty"`
// }

// type FundingGroup struct {
// 	LastModifiedDate string   `xml:"http://www.orcid.org/ns/activities last-modified-date,omitempty"`
// 	ExternalIds      string   `xml:"http://www.orcid.org/ns/activities external-ids"`
// 	FundingSummary   []string `xml:"http://www.orcid.org/ns/activities funding-summary,omitempty"`
// }

// type Fundings struct {
// 	SelfAssertedCount int `xml:"http://www.orcid.org/ns/summary self-asserted-count"`
// 	ValidatedCount    int `xml:"http://www.orcid.org/ns/summary validated-count"`
// }

// type FundingSummary struct {
// 	CreatedDate      string       `xml:"http://www.orcid.org/ns/common created-date,omitempty"`
// 	LastModifiedDate string       `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
// 	Source           string       `xml:"http://www.orcid.org/ns/common source,omitempty"`
// 	Title            FundingTitle `xml:"http://www.orcid.org/ns/funding title"`
// 	ExternalIds      string       `xml:"http://www.orcid.org/ns/funding external-ids,omitempty"`
// 	Url              string       `xml:"http://www.orcid.org/ns/funding url,omitempty"`
// 	Type             string       `xml:"http://www.orcid.org/ns/funding type"`
// 	StartDate        string       `xml:"http://www.orcid.org/ns/funding start-date,omitempty"`
// 	EndDate          string       `xml:"http://www.orcid.org/ns/funding end-date,omitempty"`
// 	Organization     string       `xml:"http://www.orcid.org/ns/funding organization"`
// 	PutCode          int          `xml:"put-code,attr,omitempty"`
// 	Visibility       string       `xml:"visibility,attr,omitempty"`
// 	DisplayIndex     string       `xml:"display-index,attr,omitempty"`
// 	Path             string       `xml:"path,attr,omitempty"`
// }

// // Container for titles of the funding.
// type FundingTitle struct {
// 	Title           string `xml:"http://www.orcid.org/ns/funding title"`
// 	TranslatedTitle string `xml:"http://www.orcid.org/ns/funding translated-title,omitempty"`
// }

// The history of the researcher's ORCID record
// type History struct {
// 	CreationMethod       string    `xml:"http://www.orcid.org/ns/history creation-method,omitempty"`
// 	CompletionDate       time.Time `xml:"http://www.orcid.org/ns/history completion-date,omitempty"`
// 	SubmissionDate       time.Time `xml:"http://www.orcid.org/ns/history submission-date,omitempty"`
// 	LastModifiedDate     string    `xml:"http://www.orcid.org/ns/history last-modified-date,omitempty"`
// 	Claimed              bool      `xml:"http://www.orcid.org/ns/history claimed,omitempty"`
// 	Source               string    `xml:"http://www.orcid.org/ns/history source,omitempty"`
// 	DeactivationDate     time.Time `xml:"http://www.orcid.org/ns/history deactivation-date,omitempty"`
// 	VerifiedEmail        bool      `xml:"http://www.orcid.org/ns/history verified-email"`
// 	VerifiedPrimaryEmail bool      `xml:"http://www.orcid.org/ns/history verified-primary-email"`
// 	Visibility           string    `xml:"visibility,attr,omitempty"`
// }

// Container for host and proposal organisations
// type Hosts struct {
// 	Organization []string `xml:"http://www.orcid.org/ns/research-resource organization"`
// }

// type InvitedPositions struct {
// 	LastModifiedDate string   `xml:"http://www.orcid.org/ns/activities last-modified-date,omitempty"`
// 	AffiliationGroup []string `xml:"http://www.orcid.org/ns/activities affiliation-group,omitempty"`
// 	Path             string   `xml:"path,attr,omitempty"`
// }

// type Keyword struct {
// 	CreatedDate      string `xml:"http://www.orcid.org/ns/common created-date,omitempty"`
// 	LastModifiedDate string `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
// 	Source           string `xml:"http://www.orcid.org/ns/common source,omitempty"`
// 	Content          string `xml:"http://www.orcid.org/ns/common content"`
// 	PutCode          int    `xml:"put-code,attr,omitempty"`
// 	Visibility       string `xml:"visibility,attr,omitempty"`
// 	DisplayIndex     string `xml:"display-index,attr,omitempty"`
// 	Path             string `xml:"path,attr,omitempty"`
// }

// // Keyworks container
// type Keywords struct {
// 	LastModifiedDate string    `xml:"http://www.orcid.org/ns/keyword last-modified-date,omitempty"`
// 	Keyword          []Keyword `xml:"http://www.orcid.org/ns/keyword keyword,omitempty"`
// 	Path             string    `xml:"path,attr,omitempty"`
// }

// // Container for the researcher's first and last name.
// type Name struct {
// 	CreatedDate      string `xml:"http://www.orcid.org/ns/personal-details created-date,omitempty"`
// 	LastModifiedDate string `xml:"http://www.orcid.org/ns/personal-details last-modified-date,omitempty"`
// 	GivenNames       string `xml:"http://www.orcid.org/ns/personal-details given-names,omitempty"`
// 	FamilyName       string `xml:"http://www.orcid.org/ns/personal-details family-name,omitempty"`
// 	CreditName       string `xml:"http://www.orcid.org/ns/personal-details credit-name,omitempty"`
// 	Visibility       string `xml:"visibility,attr,omitempty"`
// 	Path             string `xml:"path,attr,omitempty"`
// }

// type OtherName struct {
// 	CreatedDate      string `xml:"http://www.orcid.org/ns/common created-date,omitempty"`
// 	LastModifiedDate string `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
// 	Source           string `xml:"http://www.orcid.org/ns/common source,omitempty"`
// 	Content          string `xml:"http://www.orcid.org/ns/common content"`
// 	PutCode          int    `xml:"put-code,attr,omitempty"`
// 	Visibility       string `xml:"visibility,attr,omitempty"`
// 	DisplayIndex     string `xml:"display-index,attr,omitempty"`
// 	Path             string `xml:"path,attr,omitempty"`
// }

// // Container for other names.
// type OtherNames struct {
// 	LastModifiedDate string      `xml:"http://www.orcid.org/ns/other-name last-modified-date,omitempty"`
// 	OtherName        []OtherName `xml:"http://www.orcid.org/ns/other-name other-name,omitempty"`
// 	Path             string      `xml:"path,attr,omitempty"`
// }

// type PeerReview struct {
// 	CreatedDate               string       `xml:"http://www.orcid.org/ns/common created-date,omitempty"`
// 	LastModifiedDate          string       `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
// 	Source                    string       `xml:"http://www.orcid.org/ns/common source,omitempty"`
// 	ReviewerRole              string       `xml:"http://www.orcid.org/ns/peer-review reviewer-role"`
// 	ReviewIdentifiers         ExternalIds  `xml:"http://www.orcid.org/ns/common review-identifiers"`
// 	ReviewUrl                 string       `xml:"http://www.orcid.org/ns/common review-url,omitempty"`
// 	ReviewType                string       `xml:"http://www.orcid.org/ns/peer-review review-type"`
// 	ReviewCompletionDate      FuzzyDate    `xml:"http://www.orcid.org/ns/common review-completion-date"`
// 	ReviewGroupId             string       `xml:"http://www.orcid.org/ns/common review-group-id"`
// 	SubjectExternalIdentifier ExternalId   `xml:"http://www.orcid.org/ns/common subject-external-identifier,omitempty"`
// 	SubjectContainerName      string       `xml:"http://www.orcid.org/ns/common subject-container-name,omitempty"`
// 	SubjectType               string       `xml:"http://www.orcid.org/ns/peer-review subject-type,omitempty"`
// 	SubjectName               SubjectName  `xml:"http://www.orcid.org/ns/peer-review subject-name,omitempty"`
// 	SubjectUrl                string       `xml:"http://www.orcid.org/ns/common subject-url,omitempty"`
// 	ConveningOrganization     Organization `xml:"http://www.orcid.org/ns/common convening-organization"`
// 	PutCode                   int          `xml:"put-code,attr,omitempty"`
// 	Visibility                string       `xml:"visibility,attr,omitempty"`
// 	DisplayIndex              string       `xml:"display-index,attr,omitempty"`
// 	Path                      string       `xml:"path,attr,omitempty"`
// }

// type PeerReviewDuplicates struct {
// 	LastModifiedDate  string   `xml:"http://www.orcid.org/ns/activities last-modified-date,omitempty"`
// 	ExternalIds       string   `xml:"http://www.orcid.org/ns/activities external-ids"`
// 	PeerReviewSummary []string `xml:"http://www.orcid.org/ns/activities peer-review-summary,omitempty"`
// }

// type PeerreviewGroup struct {
// 	LastModifiedDate string                 `xml:"http://www.orcid.org/ns/activities last-modified-date,omitempty"`
// 	ExternalIds      string                 `xml:"http://www.orcid.org/ns/activities external-ids"`
// 	PeerReviewGroup  []PeerReviewDuplicates `xml:"http://www.orcid.org/ns/activities peer-review-group,omitempty"`
// }

// type PeerReviews struct {
// 	PeerReviewPublicationGrants int `xml:"http://www.orcid.org/ns/summary peer-review-publication-grants"`
// 	SelfAssertedCount           int `xml:"http://www.orcid.org/ns/summary self-asserted-count,omitempty"`
// 	Total                       int `xml:"http://www.orcid.org/ns/summary total"`
// }

// type PeerReviewSummary struct {
// 	CreatedDate           string       `xml:"http://www.orcid.org/ns/common created-date,omitempty"`
// 	LastModifiedDate      string       `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
// 	Source                string       `xml:"http://www.orcid.org/ns/common source,omitempty"`
// 	ReviewerRole          string       `xml:"http://www.orcid.org/ns/peer-review reviewer-role"`
// 	ExternalIds           string       `xml:"http://www.orcid.org/ns/peer-review external-ids,omitempty"`
// 	ReviewUrl             string       `xml:"http://www.orcid.org/ns/common review-url,omitempty"`
// 	ReviewType            string       `xml:"http://www.orcid.org/ns/peer-review review-type"`
// 	CompletionDate        FuzzyDate    `xml:"http://www.orcid.org/ns/common completion-date"`
// 	ReviewGroupId         string       `xml:"http://www.orcid.org/ns/common review-group-id"`
// 	ConveningOrganization Organization `xml:"http://www.orcid.org/ns/common convening-organization"`
// 	PutCode               int          `xml:"put-code,attr,omitempty"`
// 	Visibility            string       `xml:"visibility,attr,omitempty"`
// 	DisplayIndex          string       `xml:"display-index,attr,omitempty"`
// 	Path                  string       `xml:"path,attr,omitempty"`
// }

// type Person struct {
// 	LastModifiedDate    string    `xml:"http://www.orcid.org/ns/person last-modified-date,omitempty"`
// 	Name                Name      `xml:"http://www.orcid.org/ns/personal-details name,omitempty"`
// 	OtherNames          string    `xml:"http://www.orcid.org/ns/person other-names,omitempty"`
// 	Biography           Biography `xml:"http://www.orcid.org/ns/personal-details biography,omitempty"`
// 	ResearcherUrls      string    `xml:"http://www.orcid.org/ns/person researcher-urls,omitempty"`
// 	Emails              string    `xml:"http://www.orcid.org/ns/person emails"`
// 	Addresses           string    `xml:"http://www.orcid.org/ns/person addresses,omitempty"`
// 	Keywords            string    `xml:"http://www.orcid.org/ns/person keywords,omitempty"`
// 	ExternalIdentifiers string    `xml:"http://www.orcid.org/ns/person external-identifiers,omitempty"`
// 	Path                string    `xml:"path,attr,omitempty"`
// }

// type PersonalDetails struct {
// 	LastModifiedDate string    `xml:"http://www.orcid.org/ns/personal-details last-modified-date,omitempty"`
// 	Name             Name      `xml:"http://www.orcid.org/ns/personal-details name,omitempty"`
// 	OtherNames       string    `xml:"http://www.orcid.org/ns/personal-details other-names,omitempty"`
// 	Biography        Biography `xml:"http://www.orcid.org/ns/personal-details biography,omitempty"`
// 	Path             string    `xml:"path,attr,omitempty"`
// }

// // Preferences set by the researcher or contributor.
// // (currently language preference)
// type Preferences struct {
// 	Locale string `xml:"http://www.orcid.org/ns/common locale"`
// }

// type ProfessionalActivities struct {
// 	Count                int      `xml:"http://www.orcid.org/ns/summary count"`
// 	ProfessionalActivity []string `xml:"http://www.orcid.org/ns/summary professional-activity"`
// }

// type ProfessionalActivity struct {
// 	PutCode          int64  `xml:"http://www.orcid.org/ns/summary put-code"`
// 	Type             string `xml:"http://www.orcid.org/ns/summary type"`
// 	OrganizationName string `xml:"http://www.orcid.org/ns/common organization-name"`
// 	Role             string `xml:"http://www.orcid.org/ns/common role"`
// 	Url              string `xml:"http://www.orcid.org/ns/summary url,omitempty"`
// 	StartDate        string `xml:"http://www.orcid.org/ns/summary start-date,omitempty"`
// 	EndDate          string `xml:"http://www.orcid.org/ns/summary end-date,omitempty"`
// 	Validated        bool   `xml:"http://www.orcid.org/ns/summary validated"`
// }

// // Container for proposal that led to access
// type Proposal struct {
// 	Title       ResearchResourceTitle `xml:"http://www.orcid.org/ns/research-resource title"`
// 	Hosts       Hosts                 `xml:"http://www.orcid.org/ns/research-resource hosts"`
// 	ExternalIds string                `xml:"http://www.orcid.org/ns/research-resource external-ids"`
// 	StartDate   string                `xml:"http://www.orcid.org/ns/research-resource start-date,omitempty"`
// 	EndDate     string                `xml:"http://www.orcid.org/ns/research-resource end-date,omitempty"`
// 	Url         string                `xml:"http://www.orcid.org/ns/research-resource url,omitempty"`
// }

// type Qualifications struct {
// 	LastModifiedDate string   `xml:"http://www.orcid.org/ns/activities last-modified-date,omitempty"`
// 	AffiliationGroup []string `xml:"http://www.orcid.org/ns/activities affiliation-group,omitempty"`
// 	Path             string   `xml:"path,attr,omitempty"`
// }

// // The container element for a researcher or
// // contributor ORCID Record.
// // * The type attribute can only be set by
// // ORCID, and indicates the type of ORCID Record the information
// // refers to. In most cases the value will be "user" to indicate an ORCID iD holder.
// // * The client type attribute is set by ORCID, and is
// // present when the type attribute is "group" or "client". This
// // attribute indicates the API privileges held by the group as
// // indicated by their ORCID Membership Agreement.
// type Record struct {
// 	OrcidIdentifier   string `xml:"http://www.orcid.org/ns/record orcid-identifier,omitempty"`
// 	Preferences       string `xml:"http://www.orcid.org/ns/record preferences,omitempty"`
// 	History           string `xml:"http://www.orcid.org/ns/record history,omitempty"`
// 	Person            string `xml:"http://www.orcid.org/ns/record person,omitempty"`
// 	ActivitiesSummary string `xml:"http://www.orcid.org/ns/record activities-summary,omitempty"`
// 	Path              string `xml:"path,attr,omitempty"`
// }

// type RecordSummary struct {
// 	CreatedDate             string `xml:"http://www.orcid.org/ns/summary created-date,omitempty"`
// 	LastModifiedDate        string `xml:"http://www.orcid.org/ns/summary last-modified-date,omitempty"`
// 	CreditName              string `xml:"http://www.orcid.org/ns/summary credit-name,omitempty"`
// 	OrcidIdentifier         string `xml:"http://www.orcid.org/ns/summary orcid-identifier"`
// 	ExternalIdentifiers     string `xml:"http://www.orcid.org/ns/summary external-identifiers,omitempty"`
// 	Employments             string `xml:"http://www.orcid.org/ns/summary employments,omitempty"`
// 	ProfessionalActivities  string `xml:"http://www.orcid.org/ns/summary professional-activities,omitempty"`
// 	Fundings                string `xml:"http://www.orcid.org/ns/summary fundings,omitempty"`
// 	Works                   string `xml:"http://www.orcid.org/ns/summary works,omitempty"`
// 	Peerreviews             string `xml:"http://www.orcid.org/ns/summary peer-reviews,omitempty"`
// 	EducationQualifications string `xml:"http://www.orcid.org/ns/summary education-qualifications,omitempty"`
// 	ResearchResources       string `xml:"http://www.orcid.org/ns/summary research-resources,omitempty"`
// 	EmailDomains            string `xml:"http://www.orcid.org/ns/summary email-domains,omitempty"`
// }

// type ResearcherUrl struct {
// 	CreatedDate      string `xml:"http://www.orcid.org/ns/common created-date,omitempty"`
// 	LastModifiedDate string `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
// 	Source           string `xml:"http://www.orcid.org/ns/common source,omitempty"`
// 	UrlName          string `xml:"http://www.orcid.org/ns/common url-name,omitempty"`
// 	Url              string `xml:"http://www.orcid.org/ns/common url"`
// 	PutCode          int    `xml:"put-code,attr,omitempty"`
// 	Visibility       string `xml:"visibility,attr,omitempty"`
// 	DisplayIndex     string `xml:"display-index,attr,omitempty"`
// 	Path             string `xml:"path,attr,omitempty"`
// }

// // Container for URLs of websites about or related to the researcher.
// type ResearcherUrls struct {
// 	LastModifiedDate string          `xml:"http://www.orcid.org/ns/researcher-url last-modified-date,omitempty"`
// 	ResearcherUrl    []ResearcherUrl `xml:"http://www.orcid.org/ns/researcher-url researcher-url,omitempty"`
// 	Path             string          `xml:"path,attr,omitempty"`
// }

// type ResearchResource struct {
// 	CreatedDate      string        `xml:"http://www.orcid.org/ns/common created-date,omitempty"`
// 	LastModifiedDate string        `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
// 	Source           string        `xml:"http://www.orcid.org/ns/common source,omitempty"`
// 	Proposal         Proposal      `xml:"http://www.orcid.org/ns/research-resource proposal"`
// 	ResourceItems    ResourceItems `xml:"http://www.orcid.org/ns/research-resource resource-items"`
// 	PutCode          int           `xml:"put-code,attr,omitempty"`
// 	Visibility       string        `xml:"visibility,attr,omitempty"`
// 	DisplayIndex     string        `xml:"display-index,attr,omitempty"`
// 	Path             string        `xml:"path,attr,omitempty"`
// }

// type ResearchResourceGroup struct {
// 	LastModifiedDate        string   `xml:"http://www.orcid.org/ns/activities last-modified-date,omitempty"`
// 	ExternalIds             string   `xml:"http://www.orcid.org/ns/activities external-ids"`
// 	ResearchResourceSummary []string `xml:"http://www.orcid.org/ns/activities research-resource-summary,omitempty"`
// }

// type ResearchResources struct {
// 	SelfAssertedCount int `xml:"http://www.orcid.org/ns/summary self-asserted-count"`
// 	ValidatedCount    int `xml:"http://www.orcid.org/ns/summary validated-count"`
// }

// type ResearchResourceSummary struct {
// 	CreatedDate      string   `xml:"http://www.orcid.org/ns/common created-date,omitempty"`
// 	LastModifiedDate string   `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
// 	Source           string   `xml:"http://www.orcid.org/ns/common source,omitempty"`
// 	Proposal         Proposal `xml:"http://www.orcid.org/ns/research-resource proposal"`
// 	PutCode          int      `xml:"put-code,attr,omitempty"`
// 	Visibility       string   `xml:"visibility,attr,omitempty"`
// 	DisplayIndex     string   `xml:"display-index,attr,omitempty"`
// 	Path             string   `xml:"path,attr,omitempty"`
// }

// // Container for titles of the proposal or resource-item.
// type ResearchResourceTitle struct {
// 	Title           string `xml:"http://www.orcid.org/ns/research-resource title"`
// 	TranslatedTitle string `xml:"http://www.orcid.org/ns/research-resource translated-title,omitempty"`
// }

// // Actual resources used
// type ResourceItem struct {
// 	ResourceName string `xml:"http://www.orcid.org/ns/common resource-name"`
// 	ResourceType string `xml:"http://www.orcid.org/ns/research-resource resource-type"`
// 	Hosts        Hosts  `xml:"http://www.orcid.org/ns/research-resource hosts"`
// 	ExternalIds  string `xml:"http://www.orcid.org/ns/research-resource external-ids"`
// 	Url          string `xml:"http://www.orcid.org/ns/research-resource url,omitempty"`
// }

// // Container for resources
// type ResourceItems struct {
// 	ResourceItem []ResourceItem `xml:"http://www.orcid.org/ns/research-resource resource-item"`
// }

// // A single result when performing a search on the
// // ORCID Registry.
// type Result struct {
// 	OrcidIdentifier string `xml:"http://www.orcid.org/ns/search orcid-identifier"`
// }

// // The container element for the results when
// // performing a search on the ORCID Registry. the num-found attribute
// // indicates the number of successful matches.
// type Search struct {
// 	Result   []Result `xml:"http://www.orcid.org/ns/search result,omitempty"`
// 	NumFound int      `xml:"num-found,attr,omitempty"`
// }

// // Container for peer-review subject name.
// type SubjectName struct {
// 	Title           string `xml:"http://www.orcid.org/ns/peer-review title"`
// 	Subtitle        string `xml:"http://www.orcid.org/ns/peer-review subtitle,omitempty"`
// 	TranslatedTitle string `xml:"http://www.orcid.org/ns/peer-review translated-title,omitempty"`
// }

// // The main title of the work or funding translated
// // into another language. The translated language will be included in
// // the <language-code> attribute.
// type TranslatedTitle struct {
// 	Value        string `xml:",chardata"`
// 	LanguageCode string `xml:"language-code,attr"`
// }

// // A work is any research output that the researcher produced or contributed to
// // * The put-code attribute is used only when reading this
// // element. When updating the item, the put-code attribute must be
// // included to indicate the specific record to be updated.
// type Work struct {
// 	CreatedDate      string           `xml:"http://www.orcid.org/ns/common created-date,omitempty"`
// 	LastModifiedDate string           `xml:"http://www.orcid.org/ns/common last-modified-date,omitempty"`
// 	Source           string           `xml:"http://www.orcid.org/ns/common source,omitempty"`
// 	Title            WorkTitle        `xml:"http://www.orcid.org/ns/work title"`
// 	JournalTitle     string           `xml:"http://www.orcid.org/ns/common journal-title,omitempty"`
// 	ShortDescription string           `xml:"http://www.orcid.org/ns/common short-description,omitempty"`
// 	Citation         Citation         `xml:"http://www.orcid.org/ns/work citation,omitempty"`
// 	Type             string           `xml:"http://www.orcid.org/ns/work type"`
// 	PublicationDate  string           `xml:"http://www.orcid.org/ns/work publication-date,omitempty"`
// 	ExternalIds      string           `xml:"http://www.orcid.org/ns/work external-ids,omitempty"`
// 	Url              string           `xml:"http://www.orcid.org/ns/work url,omitempty"`
// 	Contributors     WorkContributors `xml:"http://www.orcid.org/ns/work contributors,omitempty"`
// 	LanguageCode     string           `xml:"http://www.orcid.org/ns/work language-code,omitempty"`
// 	Country          string           `xml:"http://www.orcid.org/ns/work country,omitempty"`
// 	PutCode          int              `xml:"put-code,attr,omitempty"`
// 	Visibility       string           `xml:"visibility,attr,omitempty"`
// 	DisplayIndex     string           `xml:"display-index,attr,omitempty"`
// 	Path             string           `xml:"path,attr,omitempty"`
// }

// // Container for the contributors of a Work.
// type WorkContributors struct {
// 	Contributor []Contributor `xml:"http://www.orcid.org/ns/work contributor,omitempty"`
// }

// type ExternalIdentifier struct {
// 	PutCode         int64  `xml:"http://www.orcid.org/ns/summary put-code"`
// 	ExternalIdType  string `xml:"http://www.orcid.org/ns/common external-id-type"`
// 	ExternalIdValue string `xml:"http://www.orcid.org/ns/common external-id-value,omitempty"`
// 	ExternalIdUrl   string `xml:"http://www.orcid.org/ns/summary external-id-url,omitempty"`
// 	Validated       bool   `xml:"http://www.orcid.org/ns/summary validated"`
// }

// type ExternalIdentifiers struct {
// 	ExternalIdentifier []string `xml:"http://www.orcid.org/ns/summary external-identifier"`
// }
