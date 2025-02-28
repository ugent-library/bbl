package bbl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"

	"go.breu.io/ulid"
)

type Repo struct {
	db          DbAdapter
	recordSpecs map[string]*RecordSpec
}

func NewRepo(db DbAdapter) *Repo {
	return &Repo{
		db: db,
		recordSpecs: map[string]*RecordSpec{
			organizationSpec.BaseKind: organizationSpec,
			personSpec.BaseKind:       personSpec,
			projectSpec.BaseKind:      projectSpec,
			workSpec.BaseKind:         workSpec,
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
		var rawRecs []*RawRecord
		var changesApplied []*Change
		var rec *RawRecord
		var err error
		for _, c := range changes {
			if c.Op == OpAddRec {
				c.ID = newID()
				rec = &RawRecord{
					ID:   c.ID,
					Kind: c.AddRecArgs().Kind,
				}
				if _, ok := r.recordSpecs[rec.BaseKind()]; !ok {
					return fmt.Errorf("invalid rec kind %s", rec.Kind)
				}
				rawRecs = append(rawRecs, rec)
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
						return fmt.Errorf("rec %s doesn't exist", c.ID)
					}
					rawRecs = append(rawRecs, rec)
				}

				switch c.Op {
				case OpDelRec:
					rec = nil
				case OpAddAttr: // TODO check RelID
					recSpec := r.recordSpecs[rec.BaseKind()]
					args := c.AddAttrArgs()
					_, ok := recSpec.Attrs[args.Kind]
					if !ok {
						return errors.New("invalid attr kind")
					}
					if args.ID == "" {
						args.ID = newID()
					}
					rec.Attrs = append(rec.Attrs, &DbAttr{
						ID:    args.ID,
						Kind:  args.Kind,
						Val:   args.Val,
						RelID: args.RelID,
					})
				case OpSetAttr: // TODO check RelID
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
					// skip if nothing changed
					if args.RelID == attr.RelID {
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
					}
					attr.Val = args.Val
					attr.RelID = args.RelID
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

		for _, rawRec := range rawRecs {
			recSpec, ok := r.recordSpecs[rawRec.BaseKind()]
			if !ok {
				return fmt.Errorf("unknown record base kind %s", rawRec.BaseKind())
			}
			rec := recSpec.New()
			if err := rec.Load(rawRec); err != nil {
				return err
			}
			if err := rec.Validate(); err != nil {
				return err
			}
		}

		return tx.AddRev(ctx, &Rev{
			ID:      newID(),
			Changes: changesApplied,
		})
	})
}

func (r *Repo) GetOrganization(ctx context.Context, id string) (*Organization, error) {
	rec, err := r.db.GetRecWithKind(ctx, organizationSpec.BaseKind, id)
	if err != nil {
		return nil, err
	}
	return loadOrganization(rec)
}

func (r *Repo) GetPerson(ctx context.Context, id string) (*Person, error) {
	rec, err := r.db.GetRecWithKind(ctx, personSpec.BaseKind, id)
	if err != nil {
		return nil, err
	}
	return loadPerson(rec)
}

func (r *Repo) GetProject(ctx context.Context, id string) (*Project, error) {
	rec, err := r.db.GetRecWithKind(ctx, projectSpec.BaseKind, id)
	if err != nil {
		return nil, err
	}
	return loadProject(rec)
}

func (r *Repo) GetWork(ctx context.Context, id string) (*Work, error) {
	rec, err := r.db.GetRecWithKind(ctx, workSpec.BaseKind, id)
	if err != nil {
		return nil, err
	}
	return loadWork(rec)
}

func newID() string {
	return ulid.Make().UUIDString()
}
