package bbl

import "github.com/ugent-library/vo"

// ValidateProject validates a project: shape is always checked;
// completeness (required fields) is checked when status is public.
func ValidateProject(p *Project) vo.Errors {
	v := vo.New()

	// Shape: always enforced.
	v.Add(vo.OneOf("status", p.Status, []string{
		ProjectStatusPublic,
		ProjectStatusDeleted,
	}))
	for i, t := range p.Titles {
		b := v.In("titles").Index(i)
		b.Add(vo.NotBlank("val", t.Val))
		if t.Lang != "" {
			b.Add(vo.ISO639_2("lang", t.Lang))
		}
	}
	for i, d := range p.Descriptions {
		b := v.In("descriptions").Index(i)
		b.Add(vo.NotBlank("val", d.Val))
		if d.Lang != "" {
			b.Add(vo.ISO639_2("lang", d.Lang))
		}
	}

	// Completeness: titles required always.
	if len(p.Titles) == 0 {
		v.Add(vo.NewError("titles", vo.RuleNotEmpty).WithMessage(vo.MessageNotEmpty))
	}

	return v.Validate()
}
