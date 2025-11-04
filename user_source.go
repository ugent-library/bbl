package bbl

import (
	"context"
	"iter"
	"maps"
	"time"
)

type UserSource interface {
	Interval() time.Duration
	MatchIdentifierScheme() string
	Iter(context.Context, *error) iter.Seq[*User]
}

var userSources = map[string]UserSource{}

func RegisterUserSource(name string, source UserSource) {
	userSources[name] = source
}

func UserSources() iter.Seq[string] {
	return maps.Keys(userSources)
}

func GetUserSource(name string) UserSource {
	return userSources[name]
}
