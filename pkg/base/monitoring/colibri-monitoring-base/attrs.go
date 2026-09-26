package colibri_monitoring_base

import (
	"sort"
	"sync"
)

// Attr is a single key/value metric attribute.
type Attr struct {
	Key   string
	Value string
}

// Attrs is an immutable, reusable set of metric attributes.
//
// Building the provider representation of an attribute set is the expensive part of
// recording a measurement, so an Attrs caches it on first use and every later record
// reuses it. Hold on to the Attrs you record with — creating a new one per call throws
// that cache away.
//
// The zero value is valid and carries no attributes. Attrs is safe to copy: it holds a
// single pointer to the shared, immutable set.
type Attrs struct {
	set *attrSet
}

type attrSet struct {
	pairs []Attr

	once  sync.Once
	cache any
}

// NewAttrs builds an attribute set from alternating key/value arguments:
//
//	attrs := NewAttrs("route", "/api/users", "status", "200")
//
// A trailing key without a value is ignored. Duplicate keys keep the last value.
func NewAttrs(kv ...string) Attrs {
	pairs := make([]Attr, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		pairs = append(pairs, Attr{Key: kv[i], Value: kv[i+1]})
	}

	return newAttrs(pairs)
}

// AttrsFromMap builds an attribute set from a string map. It is the migration path for
// callers still holding the map[string]string attributes of the deprecated APIs; prefer
// NewAttrs, which keeps the allocation out of the recording path.
func AttrsFromMap(m map[string]string) Attrs {
	pairs := make([]Attr, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, Attr{Key: k, Value: v})
	}

	return newAttrs(pairs)
}

// newAttrs normalizes the pairs and wraps them in a shared set. Sorting gives equal
// attribute sets the same representation regardless of construction order, which matters
// because map iteration order is random.
func newAttrs(pairs []Attr) Attrs {
	sort.SliceStable(pairs, func(i, j int) bool { return pairs[i].Key < pairs[j].Key })

	deduped := pairs[:0]
	for i, pair := range pairs {
		if i > 0 && deduped[len(deduped)-1].Key == pair.Key {
			deduped[len(deduped)-1] = pair
			continue
		}
		deduped = append(deduped, pair)
	}

	return Attrs{set: &attrSet{pairs: deduped}}
}

// Pairs returns the attributes, sorted by key. The slice must not be modified.
func (a Attrs) Pairs() []Attr {
	if a.set == nil {
		return nil
	}

	return a.set.pairs
}

// Len returns the number of attributes.
func (a Attrs) Len() int {
	return len(a.Pairs())
}

// Cached returns the provider representation of this attribute set, calling build once
// and reusing the result on every later call. Monitoring implementations use it to hoist
// their per-call conversion out of the hot path; callers of the Monitoring API do not
// need it.
//
// build is only ever called for one provider, since a process runs a single Monitoring
// implementation. On the zero value, build is called with nil attributes every time,
// because there is no shared set to cache into.
func (a Attrs) Cached(build func([]Attr) any) any {
	if a.set == nil {
		return build(nil)
	}

	a.set.once.Do(func() { a.set.cache = build(a.set.pairs) })

	return a.set.cache
}
