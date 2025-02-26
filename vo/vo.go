package vo

import (
	"fmt"
)

type Errors []*Error

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

func (v *Validator) In(namespace string) *Builder {
	return &Builder{validator: v, path: namespace, namespace: namespace}
}

func (v *Validator) Index(i int) *Builder {
	return &Builder{validator: v, path: fmt.Sprintf("[%d]", i)}
}

// Get fetches an Error by key or return nil if the key is not found.
func (v *Validator) Get(key string) *Error {
	for _, e := range v.errors {
		if e.Key == key {
			return e
		}
	}
	return nil
}

// Valid returns true if there are no errors.
func (v *Validator) Valid() bool {
	return len(v.errors) == 0
}

// Validate returns the errors or nil.
func (v *Validator) Validate() Errors {
	if len(v.errors) > 0 {
		return v.errors
	}
	return nil
}

type Error struct {
	Key       string
	Namespace string
	Path      string
	Rule      string
	Params    []any
	Message   string
}

// NewError constructs a new validation error. key represents the field or value
// that failed validation. There are no assumptions about the nature of this
// key, it could be a JSON pointer or the name of a (nested) form field.
func NewError(key, rule string, params ...any) *Error {
	return &Error{
		Key:    key,
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

func (e *Error) NamespacedKey() string {
	if e.Namespace != "" {
		return e.Namespace + "." + e.Key
	}
	return e.Key
}

type Builder struct {
	validator *Validator
	namespace string
	path      string
}

func (w *Builder) In(namespace string) *Builder {
	return &Builder{
		validator: w.validator,
		namespace: w.namespace + "." + namespace,
		path:      w.path + "." + namespace,
	}
}

func (w *Builder) Index(i int) *Builder {
	return &Builder{
		validator: w.validator,
		namespace: w.namespace,
		path:      w.path + fmt.Sprintf("[%d]", i),
	}
}

func (w *Builder) Add(errs ...*Error) *Builder {
	if w.path != "" {
		for _, err := range errs {
			if !err.Valid() {
				if err.Namespace != "" {
					err.Namespace = w.namespace + "." + err.Namespace
				} else {
					err.Namespace = w.namespace
				}
				err.Path = w.path + "." + err.Path
			}
		}
	}
	w.validator.Add(errs...)
	return w
}
