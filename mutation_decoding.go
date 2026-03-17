package bbl

import (
	"encoding/json"
	"fmt"
)

// DecodeMutation decodes a JSON-encoded mutation into a concrete mutation type.
// The JSON must have a "mutation" field with the snake_case mutation name.
func DecodeMutation(data []byte) (any, error) {
	var envelope struct {
		Mutation string `json:"mutation"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("decode mutation: %w", err)
	}

	var m any
	switch envelope.Mutation {
	// Work lifecycle
	case "create_work":
		m = &CreateWork{}
	case "delete_work":
		m = &DeleteWork{}
	// Work scalar fields
	case "set_work_article_number":
		m = &SetWorkArticleNumber{}
	case "unset_work_article_number":
		m = &UnsetWorkArticleNumber{}
	case "set_work_book_title":
		m = &SetWorkBookTitle{}
	case "unset_work_book_title":
		m = &UnsetWorkBookTitle{}
	case "set_work_conference":
		m = &SetWorkConference{}
	case "unset_work_conference":
		m = &UnsetWorkConference{}
	case "set_work_edition":
		m = &SetWorkEdition{}
	case "unset_work_edition":
		m = &UnsetWorkEdition{}
	case "set_work_issue":
		m = &SetWorkIssue{}
	case "unset_work_issue":
		m = &UnsetWorkIssue{}
	case "set_work_issue_title":
		m = &SetWorkIssueTitle{}
	case "unset_work_issue_title":
		m = &UnsetWorkIssueTitle{}
	case "set_work_journal_abbreviation":
		m = &SetWorkJournalAbbreviation{}
	case "unset_work_journal_abbreviation":
		m = &UnsetWorkJournalAbbreviation{}
	case "set_work_journal_title":
		m = &SetWorkJournalTitle{}
	case "unset_work_journal_title":
		m = &UnsetWorkJournalTitle{}
	case "set_work_pages":
		m = &SetWorkPages{}
	case "unset_work_pages":
		m = &UnsetWorkPages{}
	case "set_work_place_of_publication":
		m = &SetWorkPlaceOfPublication{}
	case "unset_work_place_of_publication":
		m = &UnsetWorkPlaceOfPublication{}
	case "set_work_publication_status":
		m = &SetWorkPublicationStatus{}
	case "unset_work_publication_status":
		m = &UnsetWorkPublicationStatus{}
	case "set_work_publication_year":
		m = &SetWorkPublicationYear{}
	case "unset_work_publication_year":
		m = &UnsetWorkPublicationYear{}
	case "set_work_publisher":
		m = &SetWorkPublisher{}
	case "unset_work_publisher":
		m = &UnsetWorkPublisher{}
	case "set_work_report_number":
		m = &SetWorkReportNumber{}
	case "unset_work_report_number":
		m = &UnsetWorkReportNumber{}
	case "set_work_series_title":
		m = &SetWorkSeriesTitle{}
	case "unset_work_series_title":
		m = &UnsetWorkSeriesTitle{}
	case "set_work_total_pages":
		m = &SetWorkTotalPages{}
	case "unset_work_total_pages":
		m = &UnsetWorkTotalPages{}
	case "set_work_volume":
		m = &SetWorkVolume{}
	case "unset_work_volume":
		m = &UnsetWorkVolume{}
	// Work relations
	case "set_work_titles":
		m = &SetWorkTitles{}
	case "set_work_abstracts":
		m = &SetWorkAbstracts{}
	case "unset_work_abstracts":
		m = &UnsetWorkAbstracts{}
	case "set_work_lay_summaries":
		m = &SetWorkLaySummaries{}
	case "unset_work_lay_summaries":
		m = &UnsetWorkLaySummaries{}
	case "set_work_notes":
		m = &SetWorkNotes{}
	case "unset_work_notes":
		m = &UnsetWorkNotes{}
	case "set_work_keywords":
		m = &SetWorkKeywords{}
	case "unset_work_keywords":
		m = &UnsetWorkKeywords{}
	case "set_work_identifiers":
		m = &SetWorkIdentifiers{}
	case "unset_work_identifiers":
		m = &UnsetWorkIdentifiers{}
	case "set_work_classifications":
		m = &SetWorkClassifications{}
	case "unset_work_classifications":
		m = &UnsetWorkClassifications{}
	case "set_work_contributors":
		m = &SetWorkContributors{}
	case "unset_work_contributors":
		m = &UnsetWorkContributors{}
	case "set_work_projects":
		m = &SetWorkProjects{}
	case "unset_work_projects":
		m = &UnsetWorkProjects{}
	case "set_work_organizations":
		m = &SetWorkOrganizations{}
	case "unset_work_organizations":
		m = &UnsetWorkOrganizations{}
	case "set_work_rels":
		m = &SetWorkRels{}
	case "unset_work_rels":
		m = &UnsetWorkRels{}
	// Person lifecycle
	case "create_person":
		m = &CreatePerson{}
	case "delete_person":
		m = &DeletePerson{}
	// Person fields
	case "set_person_name":
		m = &SetPersonName{}
	case "set_person_given_name":
		m = &SetPersonGivenName{}
	case "unset_person_given_name":
		m = &UnsetPersonGivenName{}
	case "set_person_middle_name":
		m = &SetPersonMiddleName{}
	case "unset_person_middle_name":
		m = &UnsetPersonMiddleName{}
	case "set_person_family_name":
		m = &SetPersonFamilyName{}
	case "unset_person_family_name":
		m = &UnsetPersonFamilyName{}
	case "set_person_identifiers":
		m = &SetPersonIdentifiers{}
	case "unset_person_identifiers":
		m = &UnsetPersonIdentifiers{}
	case "set_person_organizations":
		m = &SetPersonOrganizations{}
	case "unset_person_organizations":
		m = &UnsetPersonOrganizations{}
	// Project lifecycle
	case "create_project":
		m = &CreateProject{}
	case "delete_project":
		m = &DeleteProject{}
	// Project fields
	case "set_project_titles":
		m = &SetProjectTitles{}
	case "set_project_descriptions":
		m = &SetProjectDescriptions{}
	case "unset_project_descriptions":
		m = &UnsetProjectDescriptions{}
	case "set_project_identifiers":
		m = &SetProjectIdentifiers{}
	case "unset_project_identifiers":
		m = &UnsetProjectIdentifiers{}
	case "set_project_people":
		m = &SetProjectPeople{}
	case "unset_project_people":
		m = &UnsetProjectPeople{}
	// Organization lifecycle
	case "create_organization":
		m = &CreateOrganization{}
	case "delete_organization":
		m = &DeleteOrganization{}
	// Organization fields
	case "set_organization_names":
		m = &SetOrganizationNames{}
	case "set_organization_identifiers":
		m = &SetOrganizationIdentifiers{}
	case "unset_organization_identifiers":
		m = &UnsetOrganizationIdentifiers{}
	case "set_organization_rels":
		m = &SetOrganizationRels{}
	case "unset_organization_rels":
		m = &UnsetOrganizationRels{}
	default:
		return nil, fmt.Errorf("unknown mutation %q", envelope.Mutation)
	}

	if err := json.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("decode %s: %w", envelope.Mutation, err)
	}
	return m, nil
}
