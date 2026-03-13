package bbl

import "github.com/ugent-library/vo"

// ValidateOrganization validates an organization: shape is always checked;
// completeness (required fields) is checked when status is public.
func ValidateOrganization(o *Organization) vo.Errors {
	v := vo.New()

	// Shape: always enforced.
	v.Add(vo.OneOf("status", o.Status, []string{
		OrganizationStatusPublic,
		OrganizationStatusDeleted,
	}))
	v.Add(vo.NotBlank("kind", o.Kind))
	for i, n := range o.Names {
		b := v.In("names").Index(i)
		b.Add(vo.NotBlank("val", n.Val))
		if n.Lang != "" {
			b.Add(vo.ISO639_2("lang", n.Lang))
		}
	}

	// Completeness: only when public.
	if o.Status == OrganizationStatusPublic {
		if len(o.Names) == 0 {
			v.Add(vo.NewError("names", vo.RuleNotEmpty).WithMessage(vo.MessageNotEmpty))
		}
	}

	return v.Validate()
}
