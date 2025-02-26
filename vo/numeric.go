package vo

import (
	"fmt"

	"golang.org/x/exp/constraints"
)

var (
	RuleMin = "min"
	RuleMax = "max"

	MessageMin = "must be %d or more"
	MessageMax = "must be %d or less"
)

// Min checks if the given number is greater than or equal to min.
//
//	err := vo.Min("count", 2, 1)
func Min[T constraints.Integer | constraints.Float](key string, val T, min T) *Error {
	if val < min {
		return NewError(key, RuleMin, min).WithMessage(fmt.Sprintf(MessageMin, min))
	}
	return nil
}

// Max checks if the given number is less than or equal to max.
//
//	err := vo.Max("count", 1, 2)
func Max[T constraints.Integer | constraints.Float](key string, val T, max T) *Error {
	if val > max {
		return NewError(key, RuleMax, max).WithMessage(fmt.Sprintf(MessageMax, max))
	}
	return nil
}
