package orcid

import (
	"encoding/xml"
	"time"
)

type ActivitiesSummary struct {
	LastModifiedDate  *DateTime          `xml:"last-modified-date,omitempty"`
	Distinctions      *Affiliations      `xml:"distinctions,omitempty"`
	Educations        *Affiliations      `xml:"educations,omitempty"`
	Employments       *Affiliations      `xml:"employments,omitempty"`
	Fundings          *Fundings          `xml:"fundings,omitempty"`
	InvitedPositions  *Affiliations      `xml:"invited-positions,omitempty"`
	Memberships       *Affiliations      `xml:"memberships,omitempty"`
	PeerReviews       *PeerReviews       `xml:"peer-reviews,omitempty"`
	Qualifications    *Affiliations      `xml:"qualifications,omitempty"`
	ResearchResources *ResearchResources `xml:"research-resources,omitempty"`
	Services          *Affiliations      `xml:"services,omitempty"`
	Works             *Works             `xml:"works,omitempty"`
	Path              string             `xml:"path,attr,omitempty"`
}

type Address struct {
	ElementSummary
	Country string `xml:"country"`
}

type Addresses struct {
	LastModifiedDate *DateTime `xml:"last-modified-date,omitempty"`
	Address          []Address `xml:"address,omitempty"`
	Path             string    `xml:"path,attr,omitempty"`
}

type AffiliationGroup struct {
	LastModifiedDate        *DateTime                 `xml:"last-modified-date,omitempty"`
	ExternalIds             ExternalIds               `xml:"external-ids"`
	DistinctionSummary      []AffiliationSummary      `xml:"distinction-summary,omitempty"`
	EducationSummary        []AffiliationSummary      `xml:"education-summary,omitempty"`
	EmploymentSummary       []AffiliationSummary      `xml:"employment-summary,omitempty"`
	InvitedPositionSummary  []AffiliationSummary      `xml:"invited-position-summary,omitempty"`
	MembershipSummary       []AffiliationSummary      `xml:"membership-summary,omitempty"`
	QualificationSummary    []AffiliationSummary      `xml:"qualification-summary,omitempty"`
	ResearchResourceSummary []ResearchResourceSummary `xml:"research-resource-summary,omitempty"`
	ServiceSummary          []AffiliationSummary      `xml:"service-summary,omitempty"`
}

type Affiliations struct {
	LastModifiedDate *DateTime          `xml:"last-modified-date,omitempty"`
	AffiliationGroup []AffiliationGroup `xml:"affiliation-group,omitempty"`
	Path             string             `xml:"path,attr,omitempty"`
}

type AffiliationSummary struct {
	ElementSummary
	DepartmentName string       `xml:"department-name,omitempty"`
	RoleTitle      string       `xml:"role-title,omitempty"`
	StartDate      *FuzzyDate   `xml:"start-date,omitempty"`
	EndDate        *FuzzyDate   `xml:"end-date,omitempty"`
	Organization   Organization `xml:"organization"`
	Url            string       `xml:"url,omitempty"`
	ExternalIds    *ExternalIds `xml:"external-ids,omitempty"`
}

type Amount struct {
	Value        string `xml:",chardata"`
	CurrencyCode string `xml:"currency-code,attr"`
}

type Biography struct {
	CreatedDate      *DateTime `xml:"created-date,omitempty"`
	LastModifiedDate *DateTime `xml:"last-modified-date,omitempty"`
	Content          string    `xml:"content"`
	Visibility       string    `xml:"visibility,attr,omitempty"`
	Path             string    `xml:"path,attr,omitempty"`
}

type Bulk struct {
	Work  []Work  `xml:"work,omitempty"`
	Error []Error `xml:"error,omitempty"`
}

type Citation struct {
	CitationType  string `xml:"citation-type"`
	CitationValue string `xml:"citation-value"`
}

type ClientId struct {
	Uri  string `xml:"uri,omitempty"`
	Path string `xml:"path,omitempty"`
	Host string `xml:"host,omitempty"`
}

type DisambiguatedOrganization struct {
	DisambiguatedOrganizationIdentifier string `xml:"disambiguated-organization-identifier"`
	DisambiguationSource                string `xml:"disambiguation-source"`
}

type ElementSummary struct {
	CreatedDate      *DateTime `xml:"created-date,omitempty"`
	LastModifiedDate *DateTime `xml:"last-modified-date,omitempty"`
	Source           *Source   `xml:"source,omitempty"`
	PutCode          int       `xml:"put-code,attr,omitempty"`
	Visibility       string    `xml:"visibility,attr"`
	DisplayIndex     string    `xml:"display-index,attr,omitempty"`
	Path             string    `xml:"path,attr,omitempty"`
}

type Email struct {
	ElementSummary
	Email    string `xml:"email"`
	Primary  bool   `xml:"primary,attr"`
	Verified bool   `xml:"verified,attr"`
}

type Emails struct {
	LastModifiedDate *DateTime `xml:"last-modified-date,omitempty"`
	Email            []Email   `xml:"email,omitempty"`
	Path             string    `xml:"path,attr,omitempty"`
}

type Error struct {
	ResponseCode     int    `xml:"response-code"`
	DeveloperMessage string `xml:"developer-message"`
	UserMessage      string `xml:"user-message,omitempty"`
	ErrorCode        int    `xml:"error-code,omitempty"`
	MoreInfo         string `xml:"more-info,omitempty"`
}

type ExternalId struct {
	ElementSummary
	ExternalIdType            string          `xml:"external-id-type"`
	ExternalIdValue           string          `xml:"external-id-value"`
	ExternalIdNormalized      TransientString `xml:"external-id-normalized,omitempty"`
	ExternalIdNormalizedError TransientError  `xml:"external-id-normalized-error,omitempty"`
	ExternalIdUrl             string          `xml:"external-id-url,omitempty"`
	ExternalIdRelationship    string          `xml:"external-id-relationship,omitempty"`
}

type ExternalIdentifiers struct {
	LastModifiedDate   *DateTime    `xml:"last-modified-date,omitempty"`
	ExternalIdentifier []ExternalId `xml:"external-identifier,omitempty"`
	Path               string       `xml:"path,attr,omitempty"`
}

type ExternalIds struct {
	ExternalId []ExternalId `xml:"external-id,omitempty"`
}

type Funding struct {
	ElementSummary
	Type                    string               `xml:"type"`
	OrganizationDefinedType string               `xml:"organization-defined-type,omitempty"`
	Title                   FundingTitle         `xml:"title"`
	ShortDescription        string               `xml:"short-description,omitempty"`
	Amount                  *Amount              `xml:"amount,omitempty"`
	Url                     string               `xml:"url,omitempty"`
	StartDate               *FuzzyDate           `xml:"start-date,omitempty"`
	EndDate                 *FuzzyDate           `xml:"end-date,omitempty"`
	ExternalIds             *ExternalIds         `xml:"external-ids,omitempty"`
	Organization            Organization         `xml:"organization"`
	Contributors            *FundingContributors `xml:"contributors,omitempty"`
}

type FundingContributor struct {
	ContributorOrcId      *OrcidId                      `xml:"contributor-orcid,omitempty"`
	CreditName            string                        `xml:"credit-name,omitempty"`
	ContributorEmail      string                        `xml:"contributor-email,omitempty"`
	ContributorAttributes *FundingContributorAttributes `xml:"contributor-attributes,omitempty"`
}

type FundingContributorAttributes struct {
	ContributorRole string `xml:"contributor-role,omitempty"`
}

type FundingContributors struct {
	FundingContributor []FundingContributor `xml:"contributor,omitempty"`
}

type FundingGroup struct {
	LastModifiedDate *DateTime        `xml:"last-modified-date,omitempty"`
	ExternalIds      ExternalIds      `xml:"external-ids"`
	FundingSummary   []FundingSummary `xml:"funding-summary,omitempty"`
}

type Fundings struct {
	LastModifiedDate *DateTime      `xml:"last-modified-date,omitempty"`
	Group            []FundingGroup `xml:"group,omitempty"`
	Path             string         `xml:"path,attr,omitempty"`
}

type FundingSummary struct {
	ElementSummary
	Title        FundingTitle `xml:"title"`
	ExternalIds  *ExternalIds `xml:"external-ids,omitempty"`
	Url          string       `xml:"url,omitempty"`
	Type         string       `xml:"type"`
	StartDate    *FuzzyDate   `xml:"start-date,omitempty"`
	EndDate      *FuzzyDate   `xml:"end-date,omitempty"`
	Organization Organization `xml:"organization"`
}

type FundingTitle struct {
	Title           string `xml:"title"`
	TranslatedTitle string `xml:"translated-title,omitempty"`
}

type FuzzyDate struct {
	Year  int `xml:"year"`
	Month int `xml:"month,omitempty"`
	Day   int `xml:"day,omitempty"`
}

type History struct {
	CreationMethod       string    `xml:"creation-method,omitempty"`
	CompletionDate       *DateTime `xml:"completion-date,omitempty"`
	SubmissionDate       *DateTime `xml:"submission-date,omitempty"`
	LastModifiedDate     *DateTime `xml:"last-modified-date,omitempty"`
	Claimed              bool      `xml:"claimed"`
	Source               *Source   `xml:"source,omitempty"`
	DeactivationDate     *DateTime `xml:"deactivation-date,omitempty"`
	VerifiedEmail        bool      `xml:"verified-email"`
	VerifiedPrimaryEmail bool      `xml:"verified-primary-email"`
	Visibility           string    `xml:"visibility,attr,omitempty"`
}

type Hosts struct {
	Organization []Organization `xml:"organization"`
}

type Keyword struct {
	ElementSummary
	Content string `xml:"content"`
}

type Keywords struct {
	LastModifiedDate *DateTime `xml:"last-modified-date,omitempty"`
	Keyword          []Keyword `xml:"keyword,omitempty"`
	Path             string    `xml:"path,attr,omitempty"`
}

type Name struct {
	CreatedDate      *DateTime `xml:"created-date,omitempty"`
	LastModifiedDate *DateTime `xml:"last-modified-date,omitempty"`
	GivenNames       string    `xml:"given-names,omitempty"`
	FamilyName       string    `xml:"family-name,omitempty"`
	CreditName       string    `xml:"credit-name,omitempty"`
	Visibility       string    `xml:"visibility,attr,omitempty"`
	Path             string    `xml:"path,attr,omitempty"`
}

type OrcidId struct {
	Uri  string `xml:"uri,omitempty"`
	Path string `xml:"path,omitempty"`
	Host string `xml:"host,omitempty"`
}

type Organization struct {
	Name                      string                     `xml:"name"`
	Address                   OrganizationAddress        `xml:"address"`
	DisambiguatedOrganization *DisambiguatedOrganization `xml:"disambiguated-organization,omitempty"`
}

type OrganizationAddress struct {
	City    string `xml:"city"`
	Region  string `xml:"region,omitempty"`
	Country string `xml:"country"`
}

type OtherName struct {
	ElementSummary
	Content string `xml:"content"`
}

type OtherNames struct {
	LastModifiedDate *DateTime   `xml:"last-modified-date,omitempty"`
	OtherName        []OtherName `xml:"other-name,omitempty"`
	Path             string      `xml:"path,attr,omitempty"`
}

type PeerReview struct {
	ElementSummary
	ReviewerRole              string       `xml:"reviewer-role"`
	ReviewIdentifiers         ExternalIds  `xml:"review-identifiers"`
	ReviewUrl                 string       `xml:"review-url,omitempty"`
	ReviewType                string       `xml:"review-type"`
	ReviewCompletionDate      FuzzyDate    `xml:"review-completion-date"`
	ReviewGroupId             string       `xml:"review-group-id"`
	SubjectExternalIdentifier *ExternalId  `xml:"subject-external-identifier,omitempty"`
	SubjectContainerName      string       `xml:"subject-container-name,omitempty"`
	SubjectType               string       `xml:"subject-type,omitempty"`
	SubjectName               *SubjectName `xml:"subject-name,omitempty"`
	SubjectUrl                string       `xml:"subject-url,omitempty"`
	ConveningOrganization     Organization `xml:"convening-organization"`
}

type PeerReviewGroup struct {
	LastModifiedDate  *DateTime           `xml:"last-modified-date,omitempty"`
	ExternalIds       ExternalIds         `xml:"external-ids"`
	PeerReviewSummary []PeerReviewSummary `xml:"peer-review-summary,omitempty"`
}

type PeerReviews struct {
	LastModifiedDate *DateTime         `xml:"last-modified-date,omitempty"`
	Group            []PeerReviewGroup `xml:"group,omitempty"`
	Path             string            `xml:"path,attr,omitempty"`
}

type PeerReviewSummary struct {
	ElementSummary
	ReviewerRole          string       `xml:"reviewer-role"`
	ExternalIds           *ExternalIds `xml:"external-ids,omitempty"`
	ReviewUrl             string       `xml:"review-url,omitempty"`
	ReviewType            string       `xml:"review-type"`
	CompletionDate        FuzzyDate    `xml:"completion-date"`
	ReviewGroupId         string       `xml:"review-group-id"`
	ConveningOrganization Organization `xml:"convening-organization"`
}

type Person struct {
	LastModifiedDate    *DateTime           `xml:"last-modified-date,omitempty"`
	Name                Name                `xml:"name,omitempty"`
	OtherNames          OtherNames          `xml:"other-names,omitempty"`
	Biography           Biography           `xml:"biography,omitempty"`
	ResearcherUrls      ResearcherUrls      `xml:"researcher-urls,omitempty"`
	Emails              Emails              `xml:"emails"`
	Addresses           Addresses           `xml:"addresses,omitempty"`
	Keywords            Keywords            `xml:"keywords,omitempty"`
	ExternalIdentifiers ExternalIdentifiers `xml:"external-identifiers,omitempty"`
	Path                string              `xml:"path,attr,omitempty"`
}

type PersonalDetails struct {
	LastModifiedDate *DateTime  `xml:"last-modified-date,omitempty"`
	Name             Name       `xml:"name,omitempty"`
	OtherNames       OtherNames `xml:"other-names,omitempty"`
	Biography        Biography  `xml:"biography,omitempty"`
	Path             string     `xml:"path,attr,omitempty"`
}

type Preferences struct {
	Locale string `xml:"locale"`
}

type Proposal struct {
	Title       ResearchResourceTitle `xml:"title"`
	Hosts       Hosts                 `xml:"hosts"`
	ExternalIds ExternalIds           `xml:"external-ids"`
	StartDate   *FuzzyDate            `xml:"start-date,omitempty"`
	EndDate     *FuzzyDate            `xml:"end-date,omitempty"`
	Url         string                `xml:"url,omitempty"`
}

type Record struct {
	ActivitiesSummary *ActivitiesSummary `xml:"activities-summary,omitempty"`
	History           *History           `xml:"history,omitempty"`
	OrcidIdentifier   *OrcidId           `xml:"orcid-identifier,omitempty"`
	Path              string             `xml:"path,attr,omitempty"`
	Person            *Person            `xml:"person,omitempty"`
	Preferences       *Preferences       `xml:"preferences,omitempty"`
}

type ResearchResource struct {
	ElementSummary
	Proposal      *Proposal      `xml:"proposal"`
	ResourceItems *ResourceItems `xml:"resource-items,omitempty"`
}

type ResearchResourceGroup struct {
	LastModifiedDate        *DateTime                 `xml:"last-modified-date,omitempty"`
	ExternalIds             ExternalIds               `xml:"external-ids"`
	ResearchResourceSummary []ResearchResourceSummary `xml:"research-resource-summary,omitempty"`
}

type ResearchResources struct {
	LastModifiedDate *DateTime               `xml:"last-modified-date,omitempty"`
	Group            []ResearchResourceGroup `xml:"group,omitempty"`
	Path             string                  `xml:"path,attr,omitempty"`
}

type ResearchResourceSummary struct {
	ElementSummary
	Proposal *Proposal `xml:"proposal"`
}

type ResearchResourceTitle struct {
	Title           string `xml:"title"`
	TranslatedTitle string `xml:"translated-title,omitempty"`
}

type ResearcherUrl struct {
	ElementSummary
	UrlName string `xml:"url-name,omitempty"`
	Url     string `xml:"url"`
}

type ResearcherUrls struct {
	LastModifiedDate *DateTime       `xml:"last-modified-date,omitempty"`
	ResearcherUrl    []ResearcherUrl `xml:"researcher-url,omitempty"`
	Path             string          `xml:"path,attr,omitempty"`
}

type ResourceItem struct {
	ResourceName string      `xml:"resource-name"`
	ResourceType string      `xml:"resource-type"`
	Hosts        Hosts       `xml:"hosts"`
	ExternalIds  ExternalIds `xml:"external-ids"`
	Url          string      `xml:"url,omitempty"`
}

type ResourceItems struct {
	ResourceItem []ResourceItem `xml:"resource-item"`
}

type Result struct {
	OrcidIdentifier OrcidId `xml:"orcid-identifier"`
}

type Search struct {
	Result   []Result `xml:"result,omitempty"`
	NumFound int      `xml:"num-found,attr"`
}

type Source struct {
	SourceOrcid             *OrcidId  `xml:"source-orcid,omitempty"`
	SourceClientId          *ClientId `xml:"source-client-id,omitempty"`
	SourceName              string    `xml:"source-name,omitempty"`
	AssertionOriginOrcid    *OrcidId  `xml:"assertion-origin-orcid,omitempty"`
	AssertionOriginClientId *ClientId `xml:"assertion-origin-client-id,omitempty"`
	AssertionOriginName     string    `xml:"assertion-origin-name,omitempty"`
}

type SubjectName struct {
	Title           string `xml:"title"`
	Subtitle        string `xml:"subtitle,omitempty"`
	TranslatedTitle string `xml:"translated-title,omitempty"`
}

type TransientError struct {
	ErrorCode    string `xml:"error-code,omitempty"`
	ErrorMessage string `xml:"error-message,omitempty"`
	Transient    bool   `xml:"transient,attr"`
}

type TransientString struct {
	Value     string `xml:",chardata"`
	Transient bool   `xml:"transient,attr"`
}

type Work struct {
	ElementSummary
	Title            WorkTitle         `xml:"title"`
	JournalTitle     string            `xml:"journal-title,omitempty"`
	ShortDescription string            `xml:"short-description,omitempty"`
	Citation         *Citation         `xml:"citation,omitempty"`
	Type             string            `xml:"type"`
	PublicationDate  *FuzzyDate        `xml:"publication-date,omitempty"`
	ExternalIds      *ExternalIds      `xml:"external-ids,omitempty"`
	Url              string            `xml:"url,omitempty"`
	Contributors     *WorkContributors `xml:"contributors,omitempty"`
	LanguageCode     string            `xml:"language-code,omitempty"`
	Country          string            `xml:"country,omitempty"`
}

type WorkContributor struct {
	ContributorOrcId      *OrcidId                   `xml:"contributor-orcid,omitempty"`
	CreditName            string                     `xml:"credit-name,omitempty"`
	ContributorEmail      string                     `xml:"contributor-email,omitempty"`
	ContributorAttributes *WorkContributorAttributes `xml:"contributor-attributes,omitempty"`
}

type WorkContributorAttributes struct {
	ContributorSequence string `xml:"contributor-sequence,omitempty"`
	ContributorRole     string `xml:"contributor-role,omitempty"`
}

type WorkContributors struct {
	Contributor []WorkContributor `xml:"contributor,omitempty"`
}

type WorkGroup struct {
	LastModifiedDate *DateTime     `xml:"last-modified-date,omitempty"`
	ExternalIds      ExternalIds   `xml:"external-ids"`
	WorkSummary      []WorkSummary `xml:"work-summary,omitempty"`
}

type Works struct {
	LastModifiedDate *DateTime   `xml:"last-modified-date,omitempty"`
	Group            []WorkGroup `xml:"group,omitempty"`
	Path             string      `xml:"path,attr,omitempty"`
}

type WorkSummary struct {
	ElementSummary
	Title           WorkTitle   `xml:"title"`
	ExternalIds     ExternalIds `xml:"external-ids"`
	Url             string      `xml:"url,omitempty"`
	Type            string      `xml:"type"`
	PublicationDate *FuzzyDate  `xml:"publication-date,omitempty"`
	JournalTitle    string      `xml:"journal-title,omitempty"`
}

type WorkTitle struct {
	Title           string `xml:"title"`
	Subtitle        string `xml:"subtitle,omitempty"`
	TranslatedTitle string `xml:"translated-title,omitempty"`
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
