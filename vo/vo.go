package vo

import (
	"fmt"
)

type Errors []*Error

// Error returns an error or nil.
func (errs Errors) ToError() error {
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// Error returns a string representation of an Errors.
func (errs Errors) Error() string {
	msg := ""
	for i, err := range errs {
		msg += err.Error()
		if i < len(errs)-1 {
			msg += "; "
		}
	}
	return msg
}

type Validator struct {
	errors []*Error
}

// Add is a convenience function to create an Validator and return it's validation errors.
// Returns nil if there are no errors.
func Validate(errs ...*Error) Errors {
	return New().Add(errs...).Validate()
}

// New constructs a new Validator with the given validation errors.
func New(errs ...*Error) *Validator {
	return new(Validator).Add(errs...)
}

func (v *Validator) Add(errs ...*Error) *Validator {
	for _, err := range errs {
		if !err.Valid() {
			v.errors = append(v.errors, err)
		}
	}
	return v
}

func (v *Validator) In(key string) *Builder {
	return &Builder{validator: v, path: key, namespace: key}
}

func (v *Validator) Index(i int) *Builder {
	return &Builder{validator: v, path: fmt.Sprintf("[%d]", i)}
}

// Get fetches an Error by path or return nil if the key is not found.
func (v *Validator) Get(path string) *Error {
	for _, e := range v.errors {
		if e.Path == path {
			return e
		}
	}
	return nil
}

// Valid returns true if there are no errors.
func (v *Validator) Valid() bool {
	return len(v.errors) == 0
}

// Validate returns the errors.
func (v *Validator) Validate() Errors {
	return v.errors
}

type Error struct {
	Namespace string `json:"namespace,omitempty"`
	Path      string `json:"path"`
	Rule      string `json:"rule"`
	Params    []any  `json:"params,omitempty"`
	Message   string `json:"message"`
}

// NewError constructs a new validation error. key represents the field or value
// that failed validation. There are no assumptions about the nature of this
// key, it could be a JSON pointer or the name of a (nested) form field.
func NewError(key, rule string, params ...any) *Error {
	return &Error{
		Path:   key,
		Rule:   rule,
		Params: params,
	}
}

// WithMessage sets a custom error message if the validation error is not nil.
func (e *Error) WithMessage(msg string) *Error {
	if e != nil {
		e.Message = msg
	}
	return e
}

func (e *Error) Valid() bool {
	return e == nil
}

// Error returns a string representation of the validation error.
func (e *Error) Error() string {
	msg := e.Path + " "
	if e.Message != "" {
		msg += e.Message
	} else if e.Rule != "" {
		msg += e.Rule
		if len(e.Params) > 0 {
			msg += "["
			for i, p := range e.Params {
				msg += fmt.Sprintf("%v", p)
				if i < len(e.Params)-1 {
					msg += ", "
				}
			}
			msg += "]"
		}
	}
	return msg
}

type Builder struct {
	validator *Validator
	namespace string
	path      string
}

func (b Builder) In(key string) Builder {
	return Builder{
		validator: b.validator,
		namespace: join(b.namespace, key),
		path:      join(b.path, key),
	}
}

func (b Builder) Index(i int) Builder {
	return Builder{
		validator: b.validator,
		namespace: b.namespace,
		path:      b.path + fmt.Sprintf("[%d]", i),
	}
}

func (b Builder) Add(errs ...*Error) Builder {
	if b.path != "" {
		for _, err := range errs {
			if !err.Valid() {
				err.Namespace = join(b.namespace, err.Namespace)
				err.Path = join(b.path, err.Path)
			}
		}
	}
	b.validator.Add(errs...)
	return b
}

func join(p1, p2 string) string {
	if p1 != "" && p2 != "" {
		return p1 + "." + p2
	}
	if p1 != "" {
		return p1
	}
	return p2
}
