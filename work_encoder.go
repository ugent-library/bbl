package bbl

import (
	"encoding/json"
	"fmt"
	"iter"
	"maps"
)

type WorkEncoder = func(*Work) ([]byte, error)

var workEncoders = map[string]WorkEncoder{
	"json": func(rec *Work) ([]byte, error) {
		return json.Marshal(rec)
	},
}

func RegisterWorkEncoder(format string, enc WorkEncoder) {
	workEncoders[format] = enc
}

func WorkEncoders() iter.Seq[string] {
	return maps.Keys(workEncoders)
}

func EncodeWork(rec *Work, format string) ([]byte, error) {
	enc, ok := workEncoders[format]
	if !ok {
		return nil, fmt.Errorf("EncodeWork: unknown encoder %q", format)
	}
	return enc(rec)
}
