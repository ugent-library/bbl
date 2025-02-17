package pgadapter

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/ugentlib/bbl"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

var _ bbl.DbAdapter = (*Adapter)(nil)

type dbConn interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...interface{}) pgx.Row
}

type Adapter struct {
	conn *pgxpool.Pool
}

func New(conn *pgxpool.Pool) *Adapter {
	return &Adapter{
		conn: conn,
	}
}

func (a *Adapter) MigrateUp(ctx context.Context) error {
	goose.SetTableName("bbl_goose_version")
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	db := stdlib.OpenDBFromPool(a.conn)
	defer db.Close()

	return goose.UpContext(ctx, db, "migrations")
}

func (a *Adapter) MigrateDown(ctx context.Context) error {
	goose.SetTableName("bbl_goose_version")
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	db := stdlib.OpenDBFromPool(a.conn)
	defer db.Close()

	return goose.ResetContext(ctx, db, "migrations")
}

func (a *Adapter) GetRecWithKind(ctx context.Context, kind, id string) (*bbl.DbRec, error) {
	return getRecWithKind(ctx, a.conn, kind, id)
}

func (a *Adapter) Do(ctx context.Context, fn func(bbl.DbTx) error) error {
	tx, err := a.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := fn(&Tx{conn: tx}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

type Tx struct {
	conn pgx.Tx
}

func (tx *Tx) GetRec(ctx context.Context, id string) (*bbl.DbRec, error) {
	return getRec(ctx, tx.conn, id)
}

func (tx *Tx) AddRev(ctx context.Context, rev *bbl.Rev) error {
	batch := &pgx.Batch{}

	batch.Queue(`
		insert into bbl_revs (id)
	    values ($1);`,
		rev.ID,
	)

	for _, c := range rev.Changes {
		args, err := json.Marshal(c.Args)
		if err != nil {
			return err
		}
		batch.Queue(`
			insert into bbl_changes (rev_id, rec_id, op, seq, args)
	        values (
	            $1,
	            $2,
	            $3,
	            (select count(*) from bbl_changes where rec_id = $2) + 1,
	            $4
	        );`,
			rev.ID,
			c.ID,
			c.Op,
			args,
		)

		switch c.Op {
		case bbl.OpAddRec:
			batch.Queue(`
				insert into bbl_recs (id, kind)
		        values ($1, $2);`,
				c.ID,
				c.AddRecArgs().Kind,
			)
		case bbl.OpDelRec:
			batch.Queue(`
				delete from bbl_recs
		        where id = $1;`,
				c.ID,
			)
		case bbl.OpAddAttr:
			args := c.AddAttrArgs()
			batch.Queue(`
				insert into bbl_attrs (rec_id, id, kind, seq, val)
		        values (
		        	$1,
		        	$2,
		        	$3,
		        	(select count(*) from bbl_attrs where rec_id = $1 and kind = $3) + 1,
		        	$4
		        );`,
				c.ID,
				args.ID,
				args.Kind,
				args.Val,
			)
		case bbl.OpSetAttr:
			args := c.SetAttrArgs()
			batch.Queue(`
				update bbl_attrs
		        set val = $3
		        where rec_id = $1 and id = $2;`,
				c.ID,
				args.ID,
				args.Val,
			)
		case bbl.OpDelAttr:
			batch.Queue(`
				with attr as (
		            delete from bbl_attrs
		            where rec_id = $1 and id = $2
		            returning kind, seq
		    	)
		        update bbl_attrs a
		        set seq = a.seq - 1
		        from attr
		        where a.rec_id = $1 and a.kind = attr.kind and a.seq > attr.seq;`,
				c.ID,
				c.DelAttrArgs().ID,
			)
		}
	}

	res := tx.conn.SendBatch(ctx, batch)
	defer res.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := res.Exec(); err != nil {
			return fmt.Errorf("AddRev: %w: %s", err, batch.QueuedQueries[i].SQL)
		}
	}

	if err := res.Close(); err != nil {
		return fmt.Errorf("AddRev: %w", err)
	}

	return nil
}

func getRec(ctx context.Context, conn dbConn, id string) (*bbl.DbRec, error) {
	q := `
		select r.kind,
       	       json_agg(distinct jsonb_build_object('id', a.id, 'kind', a.kind, 'val', a.val)) filter (where a.rec_id is not null) as attrs
		from bbl_recs r
		left join bbl_attrs a on r.id = a.rec_id
		where r.id = $1
		group by r.id;`

	var kind string
	var attrs json.RawMessage

	if err := conn.QueryRow(ctx, q, id).Scan(&kind, &attrs); err == pgx.ErrNoRows {
		return nil, bbl.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	rec := bbl.DbRec{ID: id, Kind: kind}

	if err := json.Unmarshal(attrs, &rec.Attrs); err != nil {
		return nil, err
	}

	return &rec, nil
}

func getRecWithKind(ctx context.Context, conn dbConn, k, id string) (*bbl.DbRec, error) {
	q := `
		select r.kind,
       	       json_agg(distinct jsonb_build_object('id', a.id, 'kind', a.kind, 'val', a.val)) filter (where a.rec_id is not null) as attrs
		from bbl_recs r
		left join bbl_attrs a on r.id = a.rec_id
		where r.kind <@ $1 and r.id = $2 
		group by r.id;`

	var kind string
	var attrs json.RawMessage

	if err := conn.QueryRow(ctx, q, k, id).Scan(&kind, &attrs); err == pgx.ErrNoRows {
		return nil, bbl.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	rec := bbl.DbRec{ID: id, Kind: kind}

	if err := json.Unmarshal(attrs, &rec.Attrs); err != nil {
		return nil, err
	}

	return &rec, nil
}
