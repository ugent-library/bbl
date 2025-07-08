package muxurl

import (
	"errors"
	"fmt"
	"net/url"
	"slices"

	"github.com/gorilla/mux"
)

func New(router *mux.Router, name string, params ...any) *url.URL {
	u, err := TryNew(router, name, params...)
	if err != nil {
		panic(err)
	}
	return u
}

func TryNew(router *mux.Router, name string, params ...any) (*url.URL, error) {
	route := router.Get(name)
	if route == nil {
		return nil, errors.New("muxurl: unknown route " + name)
	}

	if len(params)%2 != 0 {
		return nil, errors.New("muxurl: params length should be even")
	}

	varsNames, err := route.GetVarNames()
	if err != nil {
		return nil, fmt.Errorf("muxurl: %w", err)
	}

	var vars []string
	var queryParams []string

	for i := 0; i+1 < len(params); i += 2 {
		key, ok := params[i].(string)
		if !ok {
			return nil, errors.New("muxurl: param keys should be strings")
		}
		val := fmt.Sprint(params[i+1])

		if slices.Contains(varsNames, key) {
			vars = append(vars, key, val)
		} else {
			queryParams = append(queryParams, key, val)
		}
	}

	u, err := route.URL(vars...)
	if err != nil {
		return nil, fmt.Errorf("muxurl: %w", err)
	}

	if len(queryParams) > 0 {
		q := u.Query()
		for i := 0; i < len(queryParams); i += 2 {
			q.Add(queryParams[i], queryParams[i+1])
		}
		u.RawQuery = q.Encode()
	}

	return u, nil
}
