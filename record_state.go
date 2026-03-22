package bbl

// recordState holds the write-model state for a single entity during an update.
// One struct for all 4 record types — the only difference is kind (empty for person, project).
type recordState struct {
	recordType string
	id         ID
	version    int
	status     string
	kind       string                   // empty for person, project
	fields     map[string]any           // decoded pinned field values
	assertions map[string][]assertion   // all assertion rows per field
}
