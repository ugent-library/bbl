package vo

import (
	"fmt"
	"slices"
)

var (
	RuleNotEmpty    = "not_empty"
	RuleSize        = "size"
	RuleSizeBetween = "size_between"
	RuleOneOf       = "one_of"

	MessageNotEmpty    = "cannot be empty"
	MessageSize        = "size must be %d"
	MessageSizeBetween = "size must be between %d and %d"
	MessageOneOf       = "must be one of: %s"
)

// NotEmpty checks if the given slice is not empty.
//
//	err := vo.NotEmpty("keywords", []string{"childcare"})
func NotEmpty[T any](key string, val []T) *Error {
	if len(val) == 0 {
		return NewError(key, RuleNotEmpty).WithMessage(MessageNotEmpty)
	}
	return nil
}

// Size checks if the given slice has a given length.
//
//	err := vo.Size("authors", []string{"A. Nonymous"}, 1)
func Size[T any](key string, val []T, n int) *Error {
	if len(val) != n {
		return NewError(key, RuleSize, n).WithMessage(fmt.Sprintf(MessageSize, n))
	}
	return nil
}

// SizeBetween checks if the given slice has a length that is
// greater than or equal to min and less than or equal to max.
//
//	err := vo.SizeBetween("authors", []string{"A. Nonymous"}, 1, 10)
func SizeBetween[T any](key string, val []T, min, max int) *Error {
	if len(val) < min || len(val) > max {
		return NewError(key, RuleSizeBetween, min, max).WithMessage(fmt.Sprintf(MessageSizeBetween, min, max))
	}
	return nil
}

func OneOf[TT ~[]T, T comparable](key string, val T, vals TT) *Error {
	if !slices.Contains(vals, val) {
		return NewError(key, RuleOneOf, vals).WithMessage(fmt.Sprintf(MessageOneOf, FormatSlice(vals)))
	}
	return nil
}
