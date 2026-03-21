package bbl

import "encoding/json"

// dedup returns ids with duplicates removed, preserving order.
func dedupIDs(ids []ID) []ID {
	seen := make(map[ID]struct{}, len(ids))
	out := make([]ID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

// dedupStrings deduplicates a string slice, preserving order.
func dedupStrings(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// idPtrEqual reports whether two *ID pointers are equal (both nil or same value).
func idPtrEqual(a, b *ID) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// jsonSet injects a key-value pair into a JSON object.
// If val is null or empty, returns the original unchanged.
func jsonSet(obj json.RawMessage, key string, val json.RawMessage) json.RawMessage {
	if len(val) == 0 {
		return obj
	}
	m := make(map[string]json.RawMessage)
	json.Unmarshal(obj, &m)
	m[key] = val
	out, _ := json.Marshal(m)
	return out
}

// jsonBuild constructs a JSON object from key-value pairs.
// Keys with nil values are omitted.
func jsonBuild(kvs ...any) json.RawMessage {
	m := make(map[string]json.RawMessage, len(kvs)/2)
	for i := 0; i < len(kvs); i += 2 {
		k := kvs[i].(string)
		v := kvs[i+1].(json.RawMessage)
		if v != nil {
			m[k] = v
		}
	}
	out, _ := json.Marshal(m)
	return out
}
