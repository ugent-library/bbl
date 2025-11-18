package pgxrepo

import (
	"context"
	"iter"

	"github.com/jackc/pgx/v5"
	"github.com/ugent-library/bbl"
)

func getRow[T any](ctx context.Context, conn Conn, q string, args []any, scan func(pgx.Row) (T, error)) (T, error) {
	row := conn.QueryRow(ctx, q, args...)

	rec, err := scan(row)
	if err == pgx.ErrNoRows {
		err = bbl.ErrNotFound
	}
	if err != nil {
		var t T
		return t, err
	}

	return rec, nil
}

func getRows[T any](ctx context.Context, conn Conn, q string, args []any, scan func(pgx.Row) (T, error)) ([]T, error) {
	rows, err := conn.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}

	recs, err := pgx.CollectRows(rows, collectable(scan))
	if err != nil {
		return nil, err
	}

	return recs, nil
}

func rowsIter[T any](ctx context.Context, conn Conn, errPtr *error, q string, args []any, scan func(pgx.Row) (T, error)) iter.Seq[T] {
	return func(yield func(T) bool) {
		rows, err := conn.Query(ctx, q, args...)
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
