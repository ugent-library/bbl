package bbl

import "github.com/ugent-library/vo"

// validateRecord validates field completeness and domain rules from the profile.
// Required checks are status-aware. Domain validation runs on all present values.
func validateRecord(status string, fields map[string]any, defs []FieldDef) vo.Errors {
	v := vo.New()
	for _, def := range defs {
		val, ok := fields[def.Name]
		hasVal := ok && !fieldEmpty(val)

		// Required check.
		isRequired := def.Required == "always" ||
			(def.Required == "public" && status == "public")
		if isRequired && !hasVal {
			v.Add(vo.NewError(def.Name, vo.RuleNotEmpty).WithMessage(vo.MessageNotEmpty))
			continue
		}

		// Domain validation — runs on all present values, not just required fields.
		if hasVal && def.ft.validate != nil {
			v.Add(def.ft.validate(val, &def)...)
		}
	}
	return v.Validate()
}

// fieldEmpty reports whether a field value is empty (nil, empty string, or empty slice).
func fieldEmpty(val any) bool {
	if val == nil {
		return true
	}
	switch v := val.(type) {
	case string:
		return v == ""
	case []Title:
		return len(v) == 0
	case []Text:
		return len(v) == 0
	case []Note:
		return len(v) == 0
	case []Keyword:
		return len(v) == 0
	case []Identifier:
		return len(v) == 0
	case []WorkContributor:
		return len(v) == 0
	case []ID:
		return len(v) == 0
	case []WorkRel:
		return len(v) == 0
	case []PersonAffiliation:
		return len(v) == 0
	case []ProjectParticipant:
		return len(v) == 0
	case []OrganizationRel:
		return len(v) == 0
	}
	return false
}
