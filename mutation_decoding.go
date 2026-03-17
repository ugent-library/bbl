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
	case "delete_work_article_number":
		m = &DeleteWorkArticleNumber{}
	case "set_work_book_title":
		m = &SetWorkBookTitle{}
	case "delete_work_book_title":
		m = &DeleteWorkBookTitle{}
	case "set_work_conference":
		m = &SetWorkConference{}
	case "delete_work_conference":
		m = &DeleteWorkConference{}
	case "set_work_edition":
		m = &SetWorkEdition{}
	case "delete_work_edition":
		m = &DeleteWorkEdition{}
	case "set_work_issue":
		m = &SetWorkIssue{}
	case "delete_work_issue":
		m = &DeleteWorkIssue{}
	case "set_work_issue_title":
		m = &SetWorkIssueTitle{}
	case "delete_work_issue_title":
		m = &DeleteWorkIssueTitle{}
	case "set_work_journal_abbreviation":
		m = &SetWorkJournalAbbreviation{}
	case "delete_work_journal_abbreviation":
		m = &DeleteWorkJournalAbbreviation{}
	case "set_work_journal_title":
		m = &SetWorkJournalTitle{}
	case "delete_work_journal_title":
		m = &DeleteWorkJournalTitle{}
	case "set_work_pages":
		m = &SetWorkPages{}
	case "delete_work_pages":
		m = &DeleteWorkPages{}
	case "set_work_place_of_publication":
		m = &SetWorkPlaceOfPublication{}
	case "delete_work_place_of_publication":
		m = &DeleteWorkPlaceOfPublication{}
	case "set_work_publication_status":
		m = &SetWorkPublicationStatus{}
	case "delete_work_publication_status":
		m = &DeleteWorkPublicationStatus{}
	case "set_work_publication_year":
		m = &SetWorkPublicationYear{}
	case "delete_work_publication_year":
		m = &DeleteWorkPublicationYear{}
	case "set_work_publisher":
		m = &SetWorkPublisher{}
	case "delete_work_publisher":
		m = &DeleteWorkPublisher{}
	case "set_work_report_number":
		m = &SetWorkReportNumber{}
	case "delete_work_report_number":
		m = &DeleteWorkReportNumber{}
	case "set_work_series_title":
		m = &SetWorkSeriesTitle{}
	case "delete_work_series_title":
		m = &DeleteWorkSeriesTitle{}
	case "set_work_total_pages":
		m = &SetWorkTotalPages{}
	case "delete_work_total_pages":
		m = &DeleteWorkTotalPages{}
	case "set_work_volume":
		m = &SetWorkVolume{}
	case "delete_work_volume":
		m = &DeleteWorkVolume{}
	// Work relations
	case "set_work_titles":
		m = &SetWorkTitles{}
	case "set_work_abstracts":
		m = &SetWorkAbstracts{}
	case "delete_work_abstracts":
		m = &DeleteWorkAbstracts{}
	case "set_work_lay_summaries":
		m = &SetWorkLaySummaries{}
	case "delete_work_lay_summaries":
		m = &DeleteWorkLaySummaries{}
	case "set_work_notes":
		m = &SetWorkNotes{}
	case "delete_work_notes":
		m = &DeleteWorkNotes{}
	case "set_work_keywords":
		m = &SetWorkKeywords{}
	case "delete_work_keywords":
		m = &DeleteWorkKeywords{}
	case "set_work_identifiers":
		m = &SetWorkIdentifiers{}
	case "delete_work_identifiers":
		m = &DeleteWorkIdentifiers{}
	case "set_work_classifications":
		m = &SetWorkClassifications{}
	case "delete_work_classifications":
		m = &DeleteWorkClassifications{}
	case "set_work_contributors":
		m = &SetWorkContributors{}
	case "delete_work_contributors":
		m = &DeleteWorkContributors{}
	case "set_work_projects":
		m = &SetWorkProjects{}
	case "delete_work_projects":
		m = &DeleteWorkProjects{}
	case "set_work_organizations":
		m = &SetWorkOrganizations{}
	case "delete_work_organizations":
		m = &DeleteWorkOrganizations{}
	case "set_work_rels":
		m = &SetWorkRels{}
	case "delete_work_rels":
		m = &DeleteWorkRels{}
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
	case "delete_person_given_name":
		m = &DeletePersonGivenName{}
	case "set_person_middle_name":
		m = &SetPersonMiddleName{}
	case "delete_person_middle_name":
		m = &DeletePersonMiddleName{}
	case "set_person_family_name":
		m = &SetPersonFamilyName{}
	case "delete_person_family_name":
		m = &DeletePersonFamilyName{}
	case "set_person_identifiers":
		m = &SetPersonIdentifiers{}
	case "delete_person_identifiers":
		m = &DeletePersonIdentifiers{}
	case "set_person_organizations":
		m = &SetPersonOrganizations{}
	case "delete_person_organizations":
		m = &DeletePersonOrganizations{}
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
	case "delete_project_descriptions":
		m = &DeleteProjectDescriptions{}
	case "set_project_identifiers":
		m = &SetProjectIdentifiers{}
	case "delete_project_identifiers":
		m = &DeleteProjectIdentifiers{}
	case "set_project_people":
		m = &SetProjectPeople{}
	case "delete_project_people":
		m = &DeleteProjectPeople{}
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
	case "delete_organization_identifiers":
		m = &DeleteOrganizationIdentifiers{}
	case "set_organization_rels":
		m = &SetOrganizationRels{}
	case "delete_organization_rels":
		m = &DeleteOrganizationRels{}
	default:
		return nil, fmt.Errorf("unknown mutation %q", envelope.Mutation)
	}

	if err := json.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("decode %s: %w", envelope.Mutation, err)
	}
	return m, nil
}
