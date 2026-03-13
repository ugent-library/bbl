package bbl

import (
	"github.com/ugent-library/vo"
)

// ValidateWork validates a work against the profiles: shape is always checked;
// completeness (required fields) is checked when status is public.
func ValidateWork(w *Work, profiles *WorkProfiles) vo.Errors {
	v := vo.New()

	// Shape: always enforced.
	kindNames := make([]string, len(profiles.Kinds))
	for i := range profiles.Kinds {
		kindNames[i] = profiles.Kinds[i].Name
	}
	v.Add(vo.OneOf("kind", w.Kind, kindNames))

	profile := profiles.Profile(w.Kind)
	if profile == nil {
		// Unknown kind — can't validate fields further.
		return v.Validate()
	}

	// Validate value shapes for populated fields.
	validateTitleSlice(v, "titles", w.Titles)
	validateTextSlice(v, "abstracts", w.Abstracts)
	validateTextSlice(v, "lay_summaries", w.LaySummaries)

	for i, n := range w.Notes {
		v.In("notes").Index(i).Add(
			vo.NotBlank("val", n.Val),
		)
	}

	// Build lookup of active fields and their definitions.
	activeFields := make(map[string]*WorkFieldDef, len(profile.Fields))
	for i := range profile.Fields {
		activeFields[profile.Fields[i].Name] = &profile.Fields[i]
	}

	// Validate relation shapes and schemes.
	if def, ok := activeFields["identifiers"]; ok {
		for i, id := range w.Identifiers {
			b := v.In("identifiers").Index(i)
			b.Add(vo.NotBlank("scheme", id.Scheme))
			b.Add(vo.NotBlank("val", id.Val))
			if len(def.Schemes) > 0 && id.Scheme != "" {
				b.Add(vo.OneOf("scheme", id.Scheme, def.Schemes))
			}
		}
	}

	if def, ok := activeFields["classifications"]; ok {
		for i, c := range w.Classifications {
			b := v.In("classifications").Index(i)
			b.Add(vo.NotBlank("scheme", c.Scheme))
			b.Add(vo.NotBlank("val", c.Val))
			if len(def.Schemes) > 0 && c.Scheme != "" {
				b.Add(vo.OneOf("scheme", c.Scheme, def.Schemes))
			}
		}
	}

	// Completeness: only when public.
	if w.Status == WorkStatusPublic {
		for _, f := range profile.Fields {
			if !f.Required {
				continue
			}
			switch f.Type {
			case "text_list":
				switch f.Name {
				case "titles":
					if len(w.Titles) == 0 {
						v.Add(vo.NewError(f.Name, vo.RuleNotEmpty).WithMessage(vo.MessageNotEmpty))
					}
				default:
					if len(getTextSliceFromWork(w, f.Name)) == 0 {
						v.Add(vo.NewError(f.Name, vo.RuleNotEmpty).WithMessage(vo.MessageNotEmpty))
					}
				}
			case "string_list":
				if f.Name == "keywords" && len(w.Keywords) == 0 {
					v.Add(vo.NewError(f.Name, vo.RuleNotEmpty).WithMessage(vo.MessageNotEmpty))
				}
			case "identifier_list":
				if len(w.Identifiers) == 0 {
					v.Add(vo.NewError(f.Name, vo.RuleNotEmpty).WithMessage(vo.MessageNotEmpty))
				}
			case "classification_list":
				if len(w.Classifications) == 0 {
					v.Add(vo.NewError(f.Name, vo.RuleNotEmpty).WithMessage(vo.MessageNotEmpty))
				}
			case "contributor_list":
				if len(w.Contributors) == 0 {
					v.Add(vo.NewError(f.Name, vo.RuleNotEmpty).WithMessage(vo.MessageNotEmpty))
				}
			}
			// TODO: scalar field completeness checks will use str_fields once
			// the assertion read path is wired up.
		}
	}

	return v.Validate()
}

// validateTitleSlice validates shape of a []Title field (lang codes, non-blank values).
func validateTitleSlice(v *vo.Validator, key string, titles []Title) {
	for i, t := range titles {
		b := v.In(key).Index(i)
		b.Add(vo.NotBlank("val", t.Val))
		if t.Lang != "" {
			b.Add(vo.ISO639_2("lang", t.Lang))
		}
	}
}

// validateTextSlice validates shape of a []Text field (lang codes, non-blank values).
func validateTextSlice(v *vo.Validator, key string, texts []Text) {
	for i, t := range texts {
		b := v.In(key).Index(i)
		b.Add(vo.NotBlank("val", t.Val))
		if t.Lang != "" {
			b.Add(vo.ISO639_2("lang", t.Lang))
		}
	}
}

func getTextSliceFromWork(w *Work, name string) []Text {
	switch name {
	case "abstracts":
		return w.Abstracts
	case "lay_summaries":
		return w.LaySummaries
	}
	return nil
}

