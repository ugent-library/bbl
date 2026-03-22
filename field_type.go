package bbl

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ugent-library/vo"
)

// fieldType defines the runtime behavior for a value type used in assertions.
// Each fieldType is concrete — it knows its Go type, which entities can use it,
// and how to handle relation data (if any).
type fieldType struct {
	name       string   // "string", "title", "workContributor", etc.
	entities   []string // which entity types can use this
	collection bool     // true for types that marshal/unmarshal as arrays (titles, identifiers, etc.)

	// validate performs domain validation on a field value, receiving the
	// profile FieldDef for context (schemes, etc.). nil means no domain
	// rules beyond presence — the engine skips nil validate.
	validate func(val any, def *FieldDef) []*vo.Error

	equal     func(a, b any) bool
	marshal   func(val any) ([]json.RawMessage, error)
	unmarshal func(raw json.RawMessage) (any, error)

	// relation describes the extension table for FK-bearing types.
	// nil for pure-value types.
	relation *relation

	// csv is non-nil for scalar types that support CSV batch editing.
	csv *csvCodec
}

// relation describes the extension table for a FK-bearing collection type.
// Handles both writing (buildInsert) and reading (enrichVal) FK data.
type relation struct {
	// buildInsert returns SQL + args for batch-inserting FK rows.
	// Receives the original Go value to extract FK fields.
	buildInsert func(assertionIDs []int64, val any) (string, []any)
	// joinSQL is the LEFT JOIN clause for reading, e.g.
	// "LEFT JOIN bbl_work_assertion_contributors _wac ON _wac.assertion_id = a.id"
	joinSQL string
	// cols lists the extra SELECT column expressions, e.g. ["_wac.person_id"].
	cols []string
	// scanDests returns fresh scan destinations for one row (parallel to cols).
	scanDests func() []any
	// enrichVal merges scanned FK columns into the assertion val JSON.
	// Called per assertion row before collection aggregation.
	enrichVal func(val json.RawMessage, scanned []any) json.RawMessage
}

// csvCodec defines how a scalar fieldType maps to/from CSV columns.
type csvCodec struct {
	// columns returns the CSV column names for a field (e.g. "pages" → ["pages.start", "pages.end"]).
	columns func(field string) []string
	// flatten converts a Go value to flat CSV column→value pairs.
	flatten func(field string, val any) map[string]string
	// unflatten reconstructs a Go value from CSV columns. Returns (nil, false) when all columns are empty.
	unflatten func(field string, cols map[string]string) (any, bool)
}

// allEntities is a convenience for fieldTypes valid on all entity types.
var allEntities = []string{"work", "person", "project", "organization"}

// --- scalar value types ---

var ftString = fieldType{
	name:     "string",
	entities: allEntities,
	equal: func(a, b any) bool {
		return a.(string) == b.(string)
	},
	marshal: func(val any) ([]json.RawMessage, error) {
		b, err := json.Marshal(val.(string))
		return []json.RawMessage{b}, err
	},
	unmarshal: func(raw json.RawMessage) (any, error) {
		var s string
		err := json.Unmarshal(raw, &s)
		return s, err
	},
	csv: &csvCodec{
		columns: func(field string) []string { return []string{field} },
		flatten: func(field string, val any) map[string]string {
			return map[string]string{field: val.(string)}
		},
		unflatten: func(field string, cols map[string]string) (any, bool) {
			v := cols[field]
			return v, v != ""
		},
	},
}

var ftConference = fieldType{
	name:     "conference",
	entities: []string{"work"},
	equal: func(a, b any) bool {
		return a.(Conference) == b.(Conference)
	},
	marshal: func(val any) ([]json.RawMessage, error) {
		b, err := json.Marshal(val.(Conference))
		return []json.RawMessage{b}, err
	},
	unmarshal: func(raw json.RawMessage) (any, error) {
		var v Conference
		err := json.Unmarshal(raw, &v)
		return v, err
	},
	csv: &csvCodec{
		columns: func(field string) []string {
			return []string{field + ".name", field + ".organizer", field + ".location"}
		},
		flatten: func(field string, val any) map[string]string {
			c := val.(Conference)
			return map[string]string{
				field + ".name":      c.Name,
				field + ".organizer": c.Organizer,
				field + ".location":  c.Location,
			}
		},
		unflatten: func(field string, cols map[string]string) (any, bool) {
			name := cols[field+".name"]
			org := cols[field+".organizer"]
			loc := cols[field+".location"]
			if name == "" && org == "" && loc == "" {
				return nil, false
			}
			return Conference{Name: name, Organizer: org, Location: loc}, true
		},
	},
}

var ftExtent = fieldType{
	name:     "extent",
	entities: []string{"work"},
	equal: func(a, b any) bool {
		return a.(Extent) == b.(Extent)
	},
	marshal: func(val any) ([]json.RawMessage, error) {
		b, err := json.Marshal(val.(Extent))
		return []json.RawMessage{b}, err
	},
	unmarshal: func(raw json.RawMessage) (any, error) {
		var v Extent
		err := json.Unmarshal(raw, &v)
		return v, err
	},
	csv: &csvCodec{
		columns: func(field string) []string {
			return []string{field + ".start", field + ".end"}
		},
		flatten: func(field string, val any) map[string]string {
			e := val.(Extent)
			return map[string]string{
				field + ".start": e.Start,
				field + ".end":   e.End,
			}
		},
		unflatten: func(field string, cols map[string]string) (any, bool) {
			start := cols[field+".start"]
			end := cols[field+".end"]
			if start == "" && end == "" {
				return nil, false
			}
			return Extent{Start: start, End: end}, true
		},
	},
}

// --- collection value types (pure-value, no extension table) ---

var ftText = fieldType{
	name:       "text",
	entities:   allEntities,
	collection: true,
	equal: func(a, b any) bool {
		return slices.Equal(a.([]Text), b.([]Text))
	},
	marshal: func(val any) ([]json.RawMessage, error) {
		items := val.([]Text)
		out := make([]json.RawMessage, len(items))
		for i, v := range items {
			b, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			out[i] = b
		}
		return out, nil
	},
	unmarshal: func(raw json.RawMessage) (any, error) {
		var v []Text
		err := json.Unmarshal(raw, &v)
		return v, err
	},
}

var ftTitle = fieldType{
	name:       "title",
	entities:   allEntities,
	collection: true,
	validate: func(val any, def *FieldDef) []*vo.Error {
		titles := val.([]Title)
		var errs []*vo.Error
		for i, t := range titles {
			errs = append(errs, vo.NotBlank(fmt.Sprintf("%s[%d].val", def.Name, i), t.Val))
			if t.Lang != "" {
				errs = append(errs, vo.ISO639_2(fmt.Sprintf("%s[%d].lang", def.Name, i), t.Lang))
			}
		}
		return errs
	},
	equal: func(a, b any) bool {
		return slices.Equal(a.([]Title), b.([]Title))
	},
	marshal: func(val any) ([]json.RawMessage, error) {
		items := val.([]Title)
		out := make([]json.RawMessage, len(items))
		for i, v := range items {
			b, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			out[i] = b
		}
		return out, nil
	},
	unmarshal: func(raw json.RawMessage) (any, error) {
		var v []Title
		err := json.Unmarshal(raw, &v)
		return v, err
	},
}

var ftNote = fieldType{
	name:       "note",
	entities:   []string{"work"},
	collection: true,
	equal: func(a, b any) bool {
		return slices.Equal(a.([]Note), b.([]Note))
	},
	marshal: func(val any) ([]json.RawMessage, error) {
		items := val.([]Note)
		out := make([]json.RawMessage, len(items))
		for i, v := range items {
			b, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			out[i] = b
		}
		return out, nil
	},
	unmarshal: func(raw json.RawMessage) (any, error) {
		var v []Note
		err := json.Unmarshal(raw, &v)
		return v, err
	},
}

var ftKeyword = fieldType{
	name:       "keyword",
	entities:   allEntities,
	collection: true,
	equal: func(a, b any) bool {
		return slices.Equal(a.([]Keyword), b.([]Keyword))
	},
	marshal: func(val any) ([]json.RawMessage, error) {
		items := val.([]Keyword)
		out := make([]json.RawMessage, len(items))
		for i, v := range items {
			b, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			out[i] = b
		}
		return out, nil
	},
	unmarshal: func(raw json.RawMessage) (any, error) {
		var v []Keyword
		err := json.Unmarshal(raw, &v)
		return v, err
	},
}

var ftIdentifier = fieldType{
	name:       "identifier",
	entities:   allEntities,
	collection: true,
	validate: func(val any, def *FieldDef) []*vo.Error {
		ids := val.([]Identifier)
		var errs []*vo.Error
		for i, id := range ids {
			errs = append(errs, vo.NotBlank(fmt.Sprintf("%s[%d].scheme", def.Name, i), id.Scheme))
			errs = append(errs, vo.NotBlank(fmt.Sprintf("%s[%d].val", def.Name, i), id.Val))
			if len(def.Schemes) > 0 && id.Scheme != "" && !slices.Contains(def.Schemes, id.Scheme) {
				errs = append(errs, vo.NewError(
					fmt.Sprintf("%s[%d].scheme", def.Name, i),
					vo.RuleOneOf, def.Schemes,
				).WithMessage(fmt.Sprintf(vo.MessageOneOf, vo.FormatSlice(def.Schemes))))
			}
		}
		return errs
	},
	equal: func(a, b any) bool {
		return slices.Equal(a.([]Identifier), b.([]Identifier))
	},
	marshal: func(val any) ([]json.RawMessage, error) {
		items := val.([]Identifier)
		out := make([]json.RawMessage, len(items))
		for i, v := range items {
			b, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			out[i] = b
		}
		return out, nil
	},
	unmarshal: func(raw json.RawMessage) (any, error) {
		var v []Identifier
		err := json.Unmarshal(raw, &v)
		return v, err
	},
}

var ftClassification = fieldType{
	name:       "classification",
	entities:   []string{"work"},
	collection: true,
	equal: func(a, b any) bool {
		return slices.Equal(a.([]Identifier), b.([]Identifier))
	},
	marshal: func(val any) ([]json.RawMessage, error) {
		items := val.([]Identifier)
		out := make([]json.RawMessage, len(items))
		for i, v := range items {
			b, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			out[i] = b
		}
		return out, nil
	},
	unmarshal: func(raw json.RawMessage) (any, error) {
		var v []Identifier
		err := json.Unmarshal(raw, &v)
		return v, err
	},
}

// --- FK-bearing collection types ---
//
// Marshal produces val-only JSON (no FK fields). FKs live exclusively
// in extension tables, managed by the relation field on fieldType.
// fetchState reconstructs full Go values by LEFT JOINing extension
// tables via relation.enrichVal.

var ftWorkContributor = fieldType{
	name:       "workContributor",
	entities:   []string{"work"},
	collection: true,
	validate: func(val any, def *FieldDef) []*vo.Error {
		contributors := val.([]WorkContributor)
		var errs []*vo.Error
		for i, c := range contributors {
			errs = append(errs, vo.NotBlank(fmt.Sprintf("%s[%d].name", def.Name, i), c.Name))
		}
		return errs
	},
	equal: func(a, b any) bool {
		aa, bb := a.([]WorkContributor), b.([]WorkContributor)
		if len(aa) != len(bb) {
			return false
		}
		for i := range aa {
			if aa[i].Kind != bb[i].Kind || aa[i].Name != bb[i].Name ||
				aa[i].GivenName != bb[i].GivenName || aa[i].FamilyName != bb[i].FamilyName ||
				!idPtrEqual(aa[i].PersonID, bb[i].PersonID) ||
				!slices.Equal(aa[i].Roles, bb[i].Roles) {
				return false
			}
		}
		return true
	},
	marshal: func(val any) ([]json.RawMessage, error) {
		items := val.([]WorkContributor)
		out := make([]json.RawMessage, len(items))
		for i, v := range items {
			b, err := json.Marshal(struct {
				Kind       string   `json:"kind,omitempty"`
				Name       string   `json:"name,omitempty"`
				GivenName  string   `json:"given_name,omitempty"`
				FamilyName string   `json:"family_name,omitempty"`
				Roles      []string `json:"roles,omitempty"`
			}{v.Kind, v.Name, v.GivenName, v.FamilyName, v.Roles})
			if err != nil {
				return nil, err
			}
			out[i] = b
		}
		return out, nil
	},
	unmarshal: func(raw json.RawMessage) (any, error) {
		var v []WorkContributor
		err := json.Unmarshal(raw, &v)
		return v, err
	},
	relation: &relation{
		buildInsert: func(assertionIDs []int64, val any) (string, []any) {
			return buildWorkContributorRelation(assertionIDs, val)
		},
		joinSQL:   "LEFT JOIN bbl_work_assertion_contributors _wac ON _wac.assertion_id = a.id",
		cols:      []string{"_wac.person_id"},
		scanDests: func() []any { return []any{new(pgtype.UUID)} },
		enrichVal: func(val json.RawMessage, scanned []any) json.RawMessage {
			uid := scanned[0].(*pgtype.UUID)
			if !uid.Valid {
				return val
			}
			id := ID(uid.Bytes)
			b, _ := json.Marshal(id)
			return jsonSet(val, "person_id", b)
		},
	},
}

var ftWorkProject = fieldType{
	name:       "workProject",
	entities:   []string{"work"},
	collection: true,
	equal: func(a, b any) bool {
		return slices.Equal(a.([]ID), b.([]ID))
	},
	marshal: func(val any) ([]json.RawMessage, error) {
		ids := val.([]ID)
		out := make([]json.RawMessage, len(ids))
		for i := range ids {
			out[i] = []byte(`null`)
		}
		return out, nil
	},
	unmarshal: func(raw json.RawMessage) (any, error) {
		var v []ID
		err := json.Unmarshal(raw, &v)
		return v, err
	},
	relation: &relation{
		buildInsert: func(assertionIDs []int64, val any) (string, []any) {
			return buildWorkProjectRelation(assertionIDs, val)
		},
		joinSQL:   "LEFT JOIN bbl_work_assertion_projects _wap ON _wap.assertion_id = a.id",
		cols:      []string{"_wap.project_id"},
		scanDests: func() []any { return []any{new(pgtype.UUID)} },
		enrichVal: func(_ json.RawMessage, scanned []any) json.RawMessage {
			uid := scanned[0].(*pgtype.UUID)
			id := ID(uid.Bytes)
			b, _ := json.Marshal(id)
			return b
		},
	},
}

var ftWorkOrganization = fieldType{
	name:       "workOrganization",
	entities:   []string{"work"},
	collection: true,
	equal: func(a, b any) bool {
		return slices.Equal(a.([]ID), b.([]ID))
	},
	marshal: func(val any) ([]json.RawMessage, error) {
		ids := val.([]ID)
		out := make([]json.RawMessage, len(ids))
		for i := range ids {
			out[i] = []byte(`null`)
		}
		return out, nil
	},
	unmarshal: func(raw json.RawMessage) (any, error) {
		var v []ID
		err := json.Unmarshal(raw, &v)
		return v, err
	},
	relation: &relation{
		buildInsert: func(assertionIDs []int64, val any) (string, []any) {
			return buildWorkOrganizationRelation(assertionIDs, val)
		},
		joinSQL:   "LEFT JOIN bbl_work_assertion_organizations _wao ON _wao.assertion_id = a.id",
		cols:      []string{"_wao.organization_id"},
		scanDests: func() []any { return []any{new(pgtype.UUID)} },
		enrichVal: func(_ json.RawMessage, scanned []any) json.RawMessage {
			uid := scanned[0].(*pgtype.UUID)
			id := ID(uid.Bytes)
			b, _ := json.Marshal(id)
			return b
		},
	},
}

var ftWorkRel = fieldType{
	name:       "workRel",
	entities:   []string{"work"},
	collection: true,
	equal: func(a, b any) bool {
		return slices.Equal(a.([]WorkRel), b.([]WorkRel))
	},
	marshal: func(val any) ([]json.RawMessage, error) {
		items := val.([]WorkRel)
		out := make([]json.RawMessage, len(items))
		for i := range items {
			out[i] = []byte(`null`)
		}
		return out, nil
	},
	unmarshal: func(raw json.RawMessage) (any, error) {
		var v []WorkRel
		err := json.Unmarshal(raw, &v)
		return v, err
	},
	relation: &relation{
		buildInsert: func(assertionIDs []int64, val any) (string, []any) {
			return buildWorkRelRelation(assertionIDs, val)
		},
		joinSQL:   "LEFT JOIN bbl_work_assertion_rels _war ON _war.assertion_id = a.id",
		cols:      []string{"_war.related_work_id", "_war.kind"},
		scanDests: func() []any { return []any{new(pgtype.UUID), new(pgtype.Text)} },
		enrichVal: func(_ json.RawMessage, scanned []any) json.RawMessage {
			uid := scanned[0].(*pgtype.UUID)
			kind := scanned[1].(*pgtype.Text)
			id := ID(uid.Bytes)
			idJSON, _ := json.Marshal(id)
			kindJSON, _ := json.Marshal(kind.String)
			return jsonBuild("related_work_id", idJSON, "kind", kindJSON)
		},
	},
}

var ftPersonAffiliation = fieldType{
	name:       "personAffiliation",
	entities:   []string{"person"},
	collection: true,
	equal: func(a, b any) bool {
		aa, bb := a.([]PersonAffiliation), b.([]PersonAffiliation)
		if len(aa) != len(bb) {
			return false
		}
		for i := range aa {
			if aa[i].OrganizationID != bb[i].OrganizationID {
				return false
			}
		}
		return true
	},
	marshal: func(val any) ([]json.RawMessage, error) {
		items := val.([]PersonAffiliation)
		out := make([]json.RawMessage, len(items))
		for i := range items {
			out[i] = []byte(`null`)
		}
		return out, nil
	},
	unmarshal: func(raw json.RawMessage) (any, error) {
		var v []PersonAffiliation
		err := json.Unmarshal(raw, &v)
		return v, err
	},
	relation: &relation{
		buildInsert: func(assertionIDs []int64, val any) (string, []any) {
			return buildPersonAffiliationRelation(assertionIDs, val)
		},
		joinSQL:   "LEFT JOIN bbl_person_assertion_affiliations _paa ON _paa.assertion_id = a.id",
		cols:      []string{"_paa.organization_id"},
		scanDests: func() []any { return []any{new(pgtype.UUID)} },
		enrichVal: func(_ json.RawMessage, scanned []any) json.RawMessage {
			uid := scanned[0].(*pgtype.UUID)
			id := ID(uid.Bytes)
			b, _ := json.Marshal(id)
			return jsonBuild("organization_id", b)
		},
	},
}

var ftProjectParticipant = fieldType{
	name:       "projectParticipant",
	entities:   []string{"project"},
	collection: true,
	equal: func(a, b any) bool {
		aa, bb := a.([]ProjectParticipant), b.([]ProjectParticipant)
		if len(aa) != len(bb) {
			return false
		}
		for i := range aa {
			if aa[i].PersonID != bb[i].PersonID || aa[i].Role != bb[i].Role {
				return false
			}
		}
		return true
	},
	marshal: func(val any) ([]json.RawMessage, error) {
		items := val.([]ProjectParticipant)
		out := make([]json.RawMessage, len(items))
		for i := range items {
			out[i] = []byte(`null`)
		}
		return out, nil
	},
	unmarshal: func(raw json.RawMessage) (any, error) {
		var v []ProjectParticipant
		err := json.Unmarshal(raw, &v)
		return v, err
	},
	relation: &relation{
		buildInsert: func(assertionIDs []int64, val any) (string, []any) {
			return buildProjectParticipantRelation(assertionIDs, val)
		},
		joinSQL:   "LEFT JOIN bbl_project_assertion_participants _pap ON _pap.assertion_id = a.id",
		cols:      []string{"_pap.person_id", "_pap.role"},
		scanDests: func() []any { return []any{new(pgtype.UUID), new(pgtype.Text)} },
		enrichVal: func(_ json.RawMessage, scanned []any) json.RawMessage {
			uid := scanned[0].(*pgtype.UUID)
			role := scanned[1].(*pgtype.Text)
			id := ID(uid.Bytes)
			idJSON, _ := json.Marshal(id)
			var roleJSON json.RawMessage
			if role.Valid {
				roleJSON, _ = json.Marshal(role.String)
			}
			return jsonBuild("person_id", idJSON, "role", roleJSON)
		},
	},
}

var ftOrganizationRel = fieldType{
	name:       "organizationRel",
	entities:   []string{"organization"},
	collection: true,
	equal: func(a, b any) bool {
		aa, bb := a.([]OrganizationRel), b.([]OrganizationRel)
		if len(aa) != len(bb) {
			return false
		}
		for i := range aa {
			if aa[i].RelOrganizationID != bb[i].RelOrganizationID || aa[i].Kind != bb[i].Kind {
				return false
			}
		}
		return true
	},
	marshal: func(val any) ([]json.RawMessage, error) {
		items := val.([]OrganizationRel)
		out := make([]json.RawMessage, len(items))
		for i := range items {
			out[i] = []byte(`null`)
		}
		return out, nil
	},
	unmarshal: func(raw json.RawMessage) (any, error) {
		var v []OrganizationRel
		err := json.Unmarshal(raw, &v)
		return v, err
	},
	relation: &relation{
		buildInsert: func(assertionIDs []int64, val any) (string, []any) {
			return buildOrganizationRelRelation(assertionIDs, val)
		},
		joinSQL:   "LEFT JOIN bbl_organization_assertion_rels _oar ON _oar.assertion_id = a.id",
		cols:      []string{"_oar.rel_organization_id", "_oar.kind"},
		scanDests: func() []any { return []any{new(pgtype.UUID), new(pgtype.Text)} },
		enrichVal: func(_ json.RawMessage, scanned []any) json.RawMessage {
			uid := scanned[0].(*pgtype.UUID)
			kind := scanned[1].(*pgtype.Text)
			id := ID(uid.Bytes)
			idJSON, _ := json.Marshal(id)
			kindJSON, _ := json.Marshal(kind.String)
			return jsonBuild("rel_organization_id", idJSON, "kind", kindJSON)
		},
	},
}

// --- fieldType registry ---

var fieldTypeRegistry = map[string]*fieldType{
	"string":             &ftString,
	"conference":         &ftConference,
	"extent":             &ftExtent,
	"text":               &ftText,
	"title":              &ftTitle,
	"note":               &ftNote,
	"keyword":            &ftKeyword,
	"identifier":         &ftIdentifier,
	"classification":     &ftClassification,
	"workContributor":    &ftWorkContributor,
	"workProject":        &ftWorkProject,
	"workOrganization":   &ftWorkOrganization,
	"workRel":            &ftWorkRel,
	"personAffiliation":  &ftPersonAffiliation,
	"projectParticipant": &ftProjectParticipant,
	"organizationRel":    &ftOrganizationRel,
}
