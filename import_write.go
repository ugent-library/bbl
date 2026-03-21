package bbl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Import-path write helpers. Used by ImportWorks, ImportPeople, etc.
// These write assertion rows directly for source imports — separate
// from the generic Set/Hide/Unset updaters which are for human edits.

// --- Work ---

func writeCreateWorkField(ctx context.Context, tx pgx.Tx, revID int64, workID ID, field string, val any, workSourceID *ID, userID *ID, role *string) error {
	valJSON, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("writeCreateWorkField(%s): %w", field, err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO bbl_work_assertions (rev_id, work_id, field, val, work_source_id, user_id, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		revID, workID, field, valJSON, workSourceID, userID, role)
	if err != nil {
		return fmt.Errorf("writeCreateWorkField(%s): %w", field, err)
	}
	return nil
}

func writeWorkAssertion(ctx context.Context, tx pgx.Tx, revID int64, workID ID, field string, val any, hidden bool, workSourceID *ID, userID *ID, role *string) (int64, error) {
	var valJSON []byte
	if val != nil {
		var err error
		valJSON, err = json.Marshal(val)
		if err != nil {
			return 0, fmt.Errorf("writeWorkAssertion(%s): %w", field, err)
		}
	}
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO bbl_work_assertions (rev_id, work_id, field, val, hidden, work_source_id, user_id, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		revID, workID, field, valJSON, hidden, workSourceID, userID, role).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("writeWorkAssertion(%s): %w", field, err)
	}
	return id, nil
}

func writeWorkContributor(ctx context.Context, tx pgx.Tx, assertionID int64, personID *ID, organizationID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_assertion_contributors (assertion_id, person_id, organization_id)
		VALUES ($1, $2, $3)`,
		assertionID, personID, organizationID)
	if err != nil {
		return fmt.Errorf("writeWorkContributor: %w", err)
	}
	return nil
}

func writeWorkProject(ctx context.Context, tx pgx.Tx, assertionID int64, projectID ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_assertion_projects (assertion_id, project_id)
		VALUES ($1, $2)`,
		assertionID, projectID)
	if err != nil {
		return fmt.Errorf("writeWorkProject: %w", err)
	}
	return nil
}

func writeWorkOrganization(ctx context.Context, tx pgx.Tx, assertionID int64, orgID ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_assertion_organizations (assertion_id, organization_id)
		VALUES ($1, $2)`,
		assertionID, orgID)
	if err != nil {
		return fmt.Errorf("writeWorkOrganization: %w", err)
	}
	return nil
}

func writeWorkRel(ctx context.Context, tx pgx.Tx, assertionID int64, relatedWorkID ID, kind string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_assertion_rels (assertion_id, related_work_id, kind)
		VALUES ($1, $2, $3)`,
		assertionID, relatedWorkID, kind)
	if err != nil {
		return fmt.Errorf("writeWorkRel: %w", err)
	}
	return nil
}

// --- Person ---

func writeCreatePersonField(ctx context.Context, tx pgx.Tx, revID int64, personID ID, field string, val any, personSourceID *ID, userID *ID, role *string) error {
	valJSON, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("writeCreatePersonField(%s): %w", field, err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO bbl_person_assertions (rev_id, person_id, field, val, person_source_id, user_id, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		revID, personID, field, valJSON, personSourceID, userID, role)
	if err != nil {
		return fmt.Errorf("writeCreatePersonField(%s): %w", field, err)
	}
	return nil
}

func writePersonAssertion(ctx context.Context, tx pgx.Tx, revID int64, personID ID, field string, val any, hidden bool, personSourceID *ID, userID *ID, role *string) (int64, error) {
	var valJSON []byte
	if val != nil {
		var err error
		valJSON, err = json.Marshal(val)
		if err != nil {
			return 0, fmt.Errorf("writePersonAssertion(%s): %w", field, err)
		}
	}
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO bbl_person_assertions (rev_id, person_id, field, val, hidden, person_source_id, user_id, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		revID, personID, field, valJSON, hidden, personSourceID, userID, role).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("writePersonAssertion(%s): %w", field, err)
	}
	return id, nil
}

func writePersonAffiliation(ctx context.Context, tx pgx.Tx, assertionID int64, orgID ID, validFrom, validTo *interface{}) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_person_assertion_affiliations (assertion_id, organization_id)
		VALUES ($1, $2)`,
		assertionID, orgID)
	if err != nil {
		return fmt.Errorf("writePersonAffiliation: %w", err)
	}
	return nil
}

// --- Project ---

func writeProjectAssertion(ctx context.Context, tx pgx.Tx, revID int64, projectID ID, field string, val any, hidden bool, projectSourceID *ID, userID *ID, role *string) (int64, error) {
	var valJSON []byte
	if val != nil {
		var err error
		valJSON, err = json.Marshal(val)
		if err != nil {
			return 0, fmt.Errorf("writeProjectAssertion(%s): %w", field, err)
		}
	}
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO bbl_project_assertions (rev_id, project_id, field, val, hidden, project_source_id, user_id, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		revID, projectID, field, valJSON, hidden, projectSourceID, userID, role).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("writeProjectAssertion(%s): %w", field, err)
	}
	return id, nil
}

func writeProjectParticipant(ctx context.Context, tx pgx.Tx, assertionID int64, personID ID, role string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_project_assertion_participants (assertion_id, person_id, role)
		VALUES ($1, $2, $3)`,
		assertionID, personID, nilIfEmpty(role))
	if err != nil {
		return fmt.Errorf("writeProjectParticipant: %w", err)
	}
	return nil
}

// --- Organization ---

func writeOrganizationAssertion(ctx context.Context, tx pgx.Tx, revID int64, orgID ID, field string, val any, hidden bool, orgSourceID *ID, userID *ID, role *string) (int64, error) {
	var valJSON []byte
	if val != nil {
		var err error
		valJSON, err = json.Marshal(val)
		if err != nil {
			return 0, fmt.Errorf("writeOrganizationAssertion(%s): %w", field, err)
		}
	}
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO bbl_organization_assertions (rev_id, organization_id, field, val, hidden, organization_source_id, user_id, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		revID, orgID, field, valJSON, hidden, orgSourceID, userID, role).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("writeOrganizationAssertion(%s): %w", field, err)
	}
	return id, nil
}

func writeOrganizationRel(ctx context.Context, tx pgx.Tx, assertionID int64, relOrgID ID, kind string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_organization_assertion_rels (assertion_id, rel_organization_id, kind)
		VALUES ($1, $2, $3)`,
		assertionID, relOrgID, kind)
	if err != nil {
		return fmt.Errorf("writeOrganizationRel: %w", err)
	}
	return nil
}

