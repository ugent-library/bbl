package bbl

import "github.com/ugent-library/vo"

// ValidatePerson validates a person: shape is always checked;
// completeness (required fields) is checked when status is public.
func ValidatePerson(p *Person) vo.Errors {
	v := vo.New()

	// Shape: always enforced.
	v.Add(vo.OneOf("status", p.Status, []string{
		PersonStatusPublic,
		PersonStatusDeleted,
	}))
	v.Add(vo.NotBlank("name", p.Name))

	return v.Validate()
}
