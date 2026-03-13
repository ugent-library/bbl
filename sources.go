package bbl

import (
	"context"
	"fmt"
)

// builtinSources are system-level sources that every deployment needs.
// They are seeded at boot with default priorities. If they already exist,
// priority is not overwritten (so deployments can adjust via SQL).
var builtinSources = []struct {
	ID       string
	Priority int
}{
	{"curator", 100},
	{"self_deposit", 50},
}

// SeedBuiltinSources ensures the system-level sources (curator, self_deposit)
// exist in bbl_sources with their default priorities. Existing rows are not
// modified, so deployment-specific priority overrides are preserved.
func (r *Repo) SeedBuiltinSources(ctx context.Context) error {
	for _, s := range builtinSources {
		_, err := r.db.Exec(ctx, `
			INSERT INTO bbl_sources (id, priority)
			VALUES ($1, $2)
			ON CONFLICT (id) DO NOTHING`,
			s.ID, s.Priority)
		if err != nil {
			return fmt.Errorf("SeedBuiltinSources: %w", err)
		}
	}
	return nil
}

// UpsertSource registers a source in bbl_sources if it does not already exist.
// All tables that reference bbl_sources require the source to be present first.
func (r *Repo) UpsertSource(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO bbl_sources (id)
		VALUES ($1)
		ON CONFLICT (id) DO NOTHING`,
		id)
	if err != nil {
		return fmt.Errorf("UpsertSource: %w", err)
	}
	return nil
}
