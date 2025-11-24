package pgxrepo

import (
	"github.com/jackc/pgx/v5"
)

func collectable[T any](scan func(pgx.Row) (T, error)) func(pgx.CollectableRow) (T, error) {
	return func(row pgx.CollectableRow) (T, error) {
		return scan(row)
	}
}
