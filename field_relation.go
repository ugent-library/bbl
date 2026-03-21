package bbl

import (
	"fmt"
	"strings"
)

// Relation builders return SQL + args for batch-inserting FK rows
// into extension tables. Each receives the assertion IDs and the
// original Go slice value. Extension tables are the single source of
// truth for FK data; fetchState reads them back via relation.enrichVal.

func buildWorkContributorRelation(assertionIDs []int64, val any) (string, []any) {
	items := val.([]WorkContributor)
	if len(assertionIDs) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("INSERT INTO bbl_work_assertion_contributors (assertion_id, person_id, organization_id) VALUES ")
	args := make([]any, 0, len(assertionIDs)*3)
	for i, c := range items {
		if i > 0 {
			b.WriteString(", ")
		}
		p := i * 3
		fmt.Fprintf(&b, "($%d, $%d, $%d)", p+1, p+2, p+3)
		args = append(args, assertionIDs[i], c.PersonID, nil)
	}
	return b.String(), args
}

func buildWorkProjectRelation(assertionIDs []int64, val any) (string, []any) {
	ids := val.([]ID)
	if len(assertionIDs) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("INSERT INTO bbl_work_assertion_projects (assertion_id, project_id) VALUES ")
	args := make([]any, 0, len(assertionIDs)*2)
	for i, id := range ids {
		if i > 0 {
			b.WriteString(", ")
		}
		p := i * 2
		fmt.Fprintf(&b, "($%d, $%d)", p+1, p+2)
		args = append(args, assertionIDs[i], id)
	}
	return b.String(), args
}

func buildWorkOrganizationRelation(assertionIDs []int64, val any) (string, []any) {
	ids := val.([]ID)
	if len(assertionIDs) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("INSERT INTO bbl_work_assertion_organizations (assertion_id, organization_id) VALUES ")
	args := make([]any, 0, len(assertionIDs)*2)
	for i, id := range ids {
		if i > 0 {
			b.WriteString(", ")
		}
		p := i * 2
		fmt.Fprintf(&b, "($%d, $%d)", p+1, p+2)
		args = append(args, assertionIDs[i], id)
	}
	return b.String(), args
}

func buildWorkRelRelation(assertionIDs []int64, val any) (string, []any) {
	items := val.([]WorkRel)
	if len(assertionIDs) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("INSERT INTO bbl_work_assertion_rels (assertion_id, related_work_id, kind) VALUES ")
	args := make([]any, 0, len(assertionIDs)*3)
	for i, r := range items {
		if i > 0 {
			b.WriteString(", ")
		}
		p := i * 3
		fmt.Fprintf(&b, "($%d, $%d, $%d)", p+1, p+2, p+3)
		args = append(args, assertionIDs[i], r.RelatedWorkID, r.Kind)
	}
	return b.String(), args
}

func buildPersonAffiliationRelation(assertionIDs []int64, val any) (string, []any) {
	items := val.([]PersonAffiliation)
	if len(assertionIDs) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("INSERT INTO bbl_person_assertion_affiliations (assertion_id, organization_id) VALUES ")
	args := make([]any, 0, len(assertionIDs)*2)
	for i, a := range items {
		if i > 0 {
			b.WriteString(", ")
		}
		p := i * 2
		fmt.Fprintf(&b, "($%d, $%d)", p+1, p+2)
		args = append(args, assertionIDs[i], a.OrganizationID)
	}
	return b.String(), args
}

func buildProjectParticipantRelation(assertionIDs []int64, val any) (string, []any) {
	items := val.([]ProjectParticipant)
	if len(assertionIDs) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("INSERT INTO bbl_project_assertion_participants (assertion_id, person_id, role) VALUES ")
	args := make([]any, 0, len(assertionIDs)*3)
	for i, pp := range items {
		if i > 0 {
			b.WriteString(", ")
		}
		p := i * 3
		fmt.Fprintf(&b, "($%d, $%d, $%d)", p+1, p+2, p+3)
		args = append(args, assertionIDs[i], pp.PersonID, nilIfEmpty(pp.Role))
	}
	return b.String(), args
}

func buildOrganizationRelRelation(assertionIDs []int64, val any) (string, []any) {
	items := val.([]OrganizationRel)
	if len(assertionIDs) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("INSERT INTO bbl_organization_assertion_rels (assertion_id, rel_organization_id, kind, start_date, end_date) VALUES ")
	args := make([]any, 0, len(assertionIDs)*5)
	for i, r := range items {
		if i > 0 {
			b.WriteString(", ")
		}
		p := i * 5
		fmt.Fprintf(&b, "($%d, $%d, $%d, $%d, $%d)", p+1, p+2, p+3, p+4, p+5)
		args = append(args, assertionIDs[i], r.RelOrganizationID, r.Kind, r.StartDate, r.EndDate)
	}
	return b.String(), args
}
