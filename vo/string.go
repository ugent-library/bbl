package vo

import (
	"fmt"
	"regexp"
)

var (
	RuleNotBlank      = "not_blank"
	RuleLength        = "length"
	RuleLengthBetween = "length_between"
	RuleMatch         = "match"
	RuleAlphanumeric  = "alphanumeric"

	MessageNotBlank      = "cannot be blank"
	MessageLength        = "length must be %d"
	MessageLengthBetween = "length must be between %d and %d"
	MessageMatch         = "must match %s"
	MessageAlphanumeric  = "must only contain letters a to z and digits"

	reAlphanumeric = regexp.MustCompile("^[a-zA-Z0-9]+$")
)

// NotBlank checks if the given string is not empty.
//
//	err := vo.NotBlank("title", "A title"})
func NotBlank(key string, val string) *Error {
	if len(val) == 0 {
		return NewError(key, RuleNotBlank).WithMessage(MessageNotBlank)
	}
	return nil
}

// Length checks if the given string has a given length.
//
//	err := vo.Length("postal_code", "ABAB", 4)
func Length(key string, val string, n int) *Error {
	if len(val) != n {
		return NewError(key, RuleLength, n).WithMessage(fmt.Sprintf(MessageLength, n))
	}
	return nil
}

// LengthBetween checks if the given string has a length that is
// greater than or equal to min and less than or equal to max.
//
//	err := vo.LengthBetween("tag", "childcare", 1, 25)
func LengthBetween(key string, val string, min, max int) *Error {
	if len(val) < min || len(val) > max {
		return NewError(key, RuleLengthBetween, min, max).WithMessage(fmt.Sprintf(MessageLengthBetween, min, max))
	}
	return nil
}

// Match checks if the given string matches a regular expression.
//
//	err := vo.Match("issn", "1940-5758", regexp.MustCompile(`^[0-9]{4}-[0-9]{3}[0-9X]$`))
func Match(key, val string, r *regexp.Regexp) *Error {
	if !r.MatchString(val) {
		return NewError(key, RuleMatch, r).WithMessage(fmt.Sprintf(MessageMatch, r))
	}
	return nil
}

// Alphanumeric checks if a given string only contains letters a to z, letters A to Z or digits.
//
//	err := vo.Match("issn", "1940-5758", regexp.MustCompile(`^[0-9]{4}-[0-9]{3}[0-9X]$`))
func Alphanumeric(key, val string) *Error {
	if !reAlphanumeric.MatchString(val) {
		return NewError(key, RuleAlphanumeric).WithMessage(MessageAlphanumeric)
	}
	return nil
}
