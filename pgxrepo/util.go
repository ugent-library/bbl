package pgxrepo

import (
	"context"
	"iter"

	"github.com/jackc/pgx/v5"
)

func rowsIter[T any](ctx context.Context, conn Conn, errPtr *error, q string, args []any, scan func(pgx.Row) (T, error)) iter.Seq[T] {
	return func(yield func(T) bool) {
		rows, err := conn.Query(ctx, q, args)
		if err != nil {
			*errPtr = err
			return
		}
		defer rows.Close()

		for rows.Next() {
			rec, err := scan(rows)
			if err != nil {
				*errPtr = err
				return
			}
			if !yield(rec) {
				return
			}
		}
	}
}

func collectable[T any](scan func(pgx.Row) (T, error)) func(pgx.CollectableRow) (T, error) {
	return func(row pgx.CollectableRow) (T, error) {
		return scan(row)
	}
}
