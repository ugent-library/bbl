package bbl

// Record type discriminators for entity types.
const (
	RecordTypeOrganization = "organization"
	RecordTypePerson       = "person"
	RecordTypeProject      = "project"
	RecordTypeWork         = "work"
)

// updater is the interface implemented by all update types.
// Unexported method names ensure only the bbl package can define updates.
type updater interface {
	name() string
	needs() updateNeeds
	apply(state updateState, user *User) (*updateEffect, error)
	// write returns SQL + args to be queued into a pgx.Batch.
	// Returns ("", nil) if nothing to queue (field ops use executeFieldWrites instead).
	write(revID int64, user *User) (string, []any)
}

// updateEffect is what apply returns for non-noop updates.
type updateEffect struct {
	recordType   string
	recordID     ID
	autoPinField string // non-empty for field ops that need auto-pin
}

// updateNeeds declares what existing state must be pre-fetched.
type updateNeeds struct {
	organizationIDs []ID
	personIDs       []ID
	projectIDs      []ID
	workIDs         []ID
}

// updateState holds pre-fetched state for a batch of updates.
type updateState struct {
	records    map[ID]*recordState
	priorities map[string]int // source name → priority
}
