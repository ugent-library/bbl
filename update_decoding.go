package bbl

import (
	"encoding/json"
	"fmt"
)

// DecodeUpdate decodes a JSON-encoded update into a concrete update type.
//
// Wire format: {"verb": "target", ...payload}
//
//	{"set": "work_volume", "work_id": "01J...", "val": "42"}
//	{"hide": "work_volume", "work_id": "01J..."}
//	{"unset": "work_volume", "work_id": "01J..."}
//	{"create": "work", "work_id": "01J...", "kind": "journal_article"}
//	{"delete": "work", "work_id": "01J..."}
func DecodeUpdate(data []byte) (any, error) {
	var envelope struct {
		Set    string `json:"set"`
		Hide   string `json:"hide"`
		Unset  string `json:"unset"`
		Create string `json:"create"`
		Delete string `json:"delete"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("decode update: %w", err)
	}

	var op, target string
	switch {
	case envelope.Set != "":
		op, target = "set", envelope.Set
	case envelope.Hide != "":
		op, target = "hide", envelope.Hide
	case envelope.Unset != "":
		op, target = "unset", envelope.Unset
	case envelope.Create != "":
		op, target = "create", envelope.Create
	case envelope.Delete != "":
		op, target = "delete", envelope.Delete
	default:
		return nil, fmt.Errorf("decode update: missing operation (set/hide/unset/create/delete)")
	}

	key := op + ":" + target
	var m any
	switch key {
	// Work lifecycle
	case "create:work":
		m = &CreateWork{}
	case "delete:work":
		m = &DeleteWork{}
	// Work scalar fields
	case "set:work_article_number":
		m = &SetWorkArticleNumber{}
	case "unset:work_article_number":
		m = &UnsetWorkArticleNumber{}
	case "set:work_book_title":
		m = &SetWorkBookTitle{}
	case "unset:work_book_title":
		m = &UnsetWorkBookTitle{}
	case "set:work_conference":
		m = &SetWorkConference{}
	case "unset:work_conference":
		m = &UnsetWorkConference{}
	case "set:work_edition":
		m = &SetWorkEdition{}
	case "unset:work_edition":
		m = &UnsetWorkEdition{}
	case "set:work_issue":
		m = &SetWorkIssue{}
	case "unset:work_issue":
		m = &UnsetWorkIssue{}
	case "set:work_issue_title":
		m = &SetWorkIssueTitle{}
	case "unset:work_issue_title":
		m = &UnsetWorkIssueTitle{}
	case "set:work_journal_abbreviation":
		m = &SetWorkJournalAbbreviation{}
	case "unset:work_journal_abbreviation":
		m = &UnsetWorkJournalAbbreviation{}
	case "set:work_journal_title":
		m = &SetWorkJournalTitle{}
	case "unset:work_journal_title":
		m = &UnsetWorkJournalTitle{}
	case "set:work_pages":
		m = &SetWorkPages{}
	case "unset:work_pages":
		m = &UnsetWorkPages{}
	case "set:work_place_of_publication":
		m = &SetWorkPlaceOfPublication{}
	case "unset:work_place_of_publication":
		m = &UnsetWorkPlaceOfPublication{}
	case "set:work_publication_status":
		m = &SetWorkPublicationStatus{}
	case "unset:work_publication_status":
		m = &UnsetWorkPublicationStatus{}
	case "set:work_publication_year":
		m = &SetWorkPublicationYear{}
	case "unset:work_publication_year":
		m = &UnsetWorkPublicationYear{}
	case "set:work_publisher":
		m = &SetWorkPublisher{}
	case "unset:work_publisher":
		m = &UnsetWorkPublisher{}
	case "set:work_report_number":
		m = &SetWorkReportNumber{}
	case "unset:work_report_number":
		m = &UnsetWorkReportNumber{}
	case "set:work_series_title":
		m = &SetWorkSeriesTitle{}
	case "unset:work_series_title":
		m = &UnsetWorkSeriesTitle{}
	case "set:work_total_pages":
		m = &SetWorkTotalPages{}
	case "unset:work_total_pages":
		m = &UnsetWorkTotalPages{}
	case "set:work_volume":
		m = &SetWorkVolume{}
	case "unset:work_volume":
		m = &UnsetWorkVolume{}
	// Work scalar hides
	case "hide:work_article_number":
		m = &HideWorkArticleNumber{}
	case "hide:work_book_title":
		m = &HideWorkBookTitle{}
	case "hide:work_conference":
		m = &HideWorkConference{}
	case "hide:work_edition":
		m = &HideWorkEdition{}
	case "hide:work_issue":
		m = &HideWorkIssue{}
	case "hide:work_issue_title":
		m = &HideWorkIssueTitle{}
	case "hide:work_journal_abbreviation":
		m = &HideWorkJournalAbbreviation{}
	case "hide:work_journal_title":
		m = &HideWorkJournalTitle{}
	case "hide:work_pages":
		m = &HideWorkPages{}
	case "hide:work_place_of_publication":
		m = &HideWorkPlaceOfPublication{}
	case "hide:work_publication_status":
		m = &HideWorkPublicationStatus{}
	case "hide:work_publication_year":
		m = &HideWorkPublicationYear{}
	case "hide:work_publisher":
		m = &HideWorkPublisher{}
	case "hide:work_report_number":
		m = &HideWorkReportNumber{}
	case "hide:work_series_title":
		m = &HideWorkSeriesTitle{}
	case "hide:work_total_pages":
		m = &HideWorkTotalPages{}
	case "hide:work_volume":
		m = &HideWorkVolume{}
	// Work collectives
	case "set:work_titles":
		m = &SetWorkTitles{}
	case "set:work_abstracts":
		m = &SetWorkAbstracts{}
	case "unset:work_abstracts":
		m = &UnsetWorkAbstracts{}
	case "set:work_lay_summaries":
		m = &SetWorkLaySummaries{}
	case "unset:work_lay_summaries":
		m = &UnsetWorkLaySummaries{}
	case "set:work_notes":
		m = &SetWorkNotes{}
	case "unset:work_notes":
		m = &UnsetWorkNotes{}
	case "set:work_keywords":
		m = &SetWorkKeywords{}
	case "unset:work_keywords":
		m = &UnsetWorkKeywords{}
	case "set:work_identifiers":
		m = &SetWorkIdentifiers{}
	case "unset:work_identifiers":
		m = &UnsetWorkIdentifiers{}
	case "set:work_classifications":
		m = &SetWorkClassifications{}
	case "unset:work_classifications":
		m = &UnsetWorkClassifications{}
	case "set:work_contributors":
		m = &SetWorkContributors{}
	case "unset:work_contributors":
		m = &UnsetWorkContributors{}
	case "set:work_projects":
		m = &SetWorkProjects{}
	case "unset:work_projects":
		m = &UnsetWorkProjects{}
	case "set:work_organizations":
		m = &SetWorkOrganizations{}
	case "unset:work_organizations":
		m = &UnsetWorkOrganizations{}
	case "set:work_rels":
		m = &SetWorkRels{}
	case "unset:work_rels":
		m = &UnsetWorkRels{}
	// Work collective hides
	case "hide:work_abstracts":
		m = &HideWorkAbstracts{}
	case "hide:work_lay_summaries":
		m = &HideWorkLaySummaries{}
	case "hide:work_notes":
		m = &HideWorkNotes{}
	case "hide:work_keywords":
		m = &HideWorkKeywords{}
	case "hide:work_identifiers":
		m = &HideWorkIdentifiers{}
	case "hide:work_classifications":
		m = &HideWorkClassifications{}
	case "hide:work_contributors":
		m = &HideWorkContributors{}
	case "hide:work_projects":
		m = &HideWorkProjects{}
	case "hide:work_organizations":
		m = &HideWorkOrganizations{}
	case "hide:work_rels":
		m = &HideWorkRels{}
	// Person lifecycle
	case "create:person":
		m = &CreatePerson{}
	case "delete:person":
		m = &DeletePerson{}
	// Person fields
	case "set:person_name":
		m = &SetPersonName{}
	case "set:person_given_name":
		m = &SetPersonGivenName{}
	case "unset:person_given_name":
		m = &UnsetPersonGivenName{}
	case "set:person_middle_name":
		m = &SetPersonMiddleName{}
	case "unset:person_middle_name":
		m = &UnsetPersonMiddleName{}
	case "set:person_family_name":
		m = &SetPersonFamilyName{}
	case "unset:person_family_name":
		m = &UnsetPersonFamilyName{}
	case "set:person_identifiers":
		m = &SetPersonIdentifiers{}
	case "unset:person_identifiers":
		m = &UnsetPersonIdentifiers{}
	case "set:person_affiliations":
		m = &SetPersonAffiliations{}
	case "unset:person_affiliations":
		m = &UnsetPersonAffiliations{}
	// Person hides
	case "hide:person_given_name":
		m = &HidePersonGivenName{}
	case "hide:person_middle_name":
		m = &HidePersonMiddleName{}
	case "hide:person_family_name":
		m = &HidePersonFamilyName{}
	case "hide:person_identifiers":
		m = &HidePersonIdentifiers{}
	case "hide:person_affiliations":
		m = &HidePersonAffiliations{}
	// Project lifecycle
	case "create:project":
		m = &CreateProject{}
	case "delete:project":
		m = &DeleteProject{}
	// Project fields
	case "set:project_titles":
		m = &SetProjectTitles{}
	case "set:project_descriptions":
		m = &SetProjectDescriptions{}
	case "unset:project_descriptions":
		m = &UnsetProjectDescriptions{}
	case "set:project_identifiers":
		m = &SetProjectIdentifiers{}
	case "unset:project_identifiers":
		m = &UnsetProjectIdentifiers{}
	case "set:project_participants":
		m = &SetProjectParticipants{}
	case "unset:project_participants":
		m = &UnsetProjectParticipants{}
	// Project hides
	case "hide:project_descriptions":
		m = &HideProjectDescriptions{}
	case "hide:project_identifiers":
		m = &HideProjectIdentifiers{}
	case "hide:project_participants":
		m = &HideProjectParticipants{}
	// Organization lifecycle
	case "create:organization":
		m = &CreateOrganization{}
	case "delete:organization":
		m = &DeleteOrganization{}
	// Organization fields
	case "set:organization_names":
		m = &SetOrganizationNames{}
	case "set:organization_identifiers":
		m = &SetOrganizationIdentifiers{}
	case "unset:organization_identifiers":
		m = &UnsetOrganizationIdentifiers{}
	case "set:organization_rels":
		m = &SetOrganizationRels{}
	case "unset:organization_rels":
		m = &UnsetOrganizationRels{}
	// Organization hides
	case "hide:organization_identifiers":
		m = &HideOrganizationIdentifiers{}
	case "hide:organization_rels":
		m = &HideOrganizationRels{}
	default:
		return nil, fmt.Errorf("unknown update %q", key)
	}

	if err := json.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("decode %s: %w", key, err)
	}
	return m, nil
}
