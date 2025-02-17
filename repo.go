package bbl

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"

	"go.breu.io/ulid"
)

type recSpec struct {
	Attrs map[string]*attrSpec
}

type attrSpec struct{}

type Repo struct {
	db       DbAdapter
	recSpecs map[string]*recSpec
}

func NewRepo(db DbAdapter) *Repo {
	return &Repo{
		db: db,
		recSpecs: map[string]*recSpec{
			organizationKind: organizationSpec,
		},
	}
}

func (r *Repo) MigrateUp(ctx context.Context) error {
	return r.db.MigrateUp(ctx)
}

func (r *Repo) MigrateDown(ctx context.Context) error {
	return r.db.MigrateDown(ctx)
}

func (r *Repo) AddRev(ctx context.Context, changes []*Change) error {
	return r.db.Do(ctx, func(tx DbTx) error {
		var changesApplied []*Change
		var rec *DbRec
		var err error
		for _, c := range changes {
			if c.Op == OpAddRec {
				c.ID = newID()
				rec = &DbRec{
					ID:   c.ID,
					Kind: c.AddRecArgs().Kind,
				}
				if _, ok := r.recSpecs[rec.Kind]; !ok {
					return errors.New("invalid rec kind")
				}
			} else {
				if rec != nil && c.ID == "" {
					c.ID = rec.ID
				}
				if rec == nil || rec.ID != c.ID {
					rec, err = tx.GetRec(ctx, c.ID)
					if err != nil {
						return err
					}
					if rec == nil {
						return errors.New("rec doesn't exist")
					}
				}

				switch c.Op {
				case OpDelRec:
					rec = nil
				case OpAddAttr:
					recSpec := r.recSpecs[rec.Kind]
					args := c.AddAttrArgs()
					_, ok := recSpec.Attrs[args.Kind]
					if !ok {
						return errors.New("invalid attr kind")
					}
					if args.ID == "" {
						args.ID = newID()
					}
					rec.Attrs = append(rec.Attrs, &DbAttr{
						ID:   args.ID,
						Kind: args.Kind,
						Val:  args.Val,
					})
				case OpSetAttr:
					args := c.SetAttrArgs()
					var attr *DbAttr
					for _, a := range rec.Attrs {
						if a.ID == args.ID {
							attr = a
							break
						}
					}
					if attr == nil {
						return errors.New("attr doesn't exist")
					}
					var oldVal, newVal any
					if err = json.Unmarshal(attr.Val, &oldVal); err != nil {
						return err
					}
					if err = json.Unmarshal(args.Val, &newVal); err != nil {
						return err
					}
					if reflect.DeepEqual(oldVal, newVal) {
						continue
					}
					attr.Val = args.Val
				case OpDelAttr:
					args := c.DelAttrArgs()
					var exists bool
					for i, a := range rec.Attrs {
						if a.ID == args.ID {
							exists = true
							rec.Attrs = slices.Delete(rec.Attrs, i, i+1)
							break
						}
					}
					if !exists {
						return errors.New("attr doesn't exist")
					}
				}
			}

			changesApplied = append(changesApplied, c)
		}

		return tx.AddRev(ctx, &Rev{
			ID:      newID(),
			Changes: changesApplied,
		})
	})
}

func (r *Repo) GetOrganization(ctx context.Context, id string) (*Organization, error) {
	rec, err := r.db.GetRecWithKind(ctx, organizationKind, id)
	if err != nil {
		return nil, err
	}
	return loadOrganization(rec)
}

func newID() string {
	return ulid.Make().UUIDString()
}
