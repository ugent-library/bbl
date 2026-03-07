package bbl

import (
	"context"
	"iter"
	"maps"
	"time"
)

type WorkSource interface {
	Interval() time.Duration
	MatchIdentifierScheme() string
	Iter(context.Context) (iter.Seq[*Work], func() error)
}

var workSources = map[string]WorkSource{}

func RegisterWorkSource(name string, source WorkSource) {
	workSources[name] = source
}

func WorkSources() iter.Seq[string] {
	return maps.Keys(workSources)
}

func GetWorkSource(name string) WorkSource {
	return workSources[name]
}
