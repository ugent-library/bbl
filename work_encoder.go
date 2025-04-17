package bbl

import (
	"context"
)

type WorkEncoder = func(context.Context, *Work) ([]byte, error)
