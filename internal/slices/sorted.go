package slices

import "golang.org/x/exp/slices"

type Option[E any] func(*options[E])

type options[E any] struct {
	searchOffset int
	delete       bool
	replace      bool
	replaceFunc  func(existing, target *E)
}

func newOptions[E any](opts ...Option[E]) *options[E] {
	var o options[E]
	WithOptions(opts...)(&o)
	return &o
}
func WithOptions[E any](opts ...Option[E]) Option[E] {
	return func(o *options[E]) {
		for _, opt := range opts {
			opt(o)
		}
	}
}
func WithSearchOffset[E any](offset int) Option[E] {
	return func(o *options[E]) {
		o.searchOffset += offset
	}
}
func WithDelete[E any]() Option[E] {
	return func(o *options[E]) {
		o.delete = true
	}
}
func WithInsert[E any]() Option[E] {
	return func(o *options[E]) {
		o.delete = false
	}
}
func WithReplace[E any]() Option[E] {
	return func(o *options[E]) {
		o.replace = true
	}
}
func WithReplaceFunc[E any](replace func(existing, target *E)) Option[E] {
	return func(o *options[E]) {
		o.replaceFunc = replace
	}
}
func WithKeepOriginal[E any]() Option[E] {
	return WithReplaceFunc(func(_, _ *E) {})
}

// UpdateSorted uses binary search to either insert or delete v from the sorted
// slice s, according to the given opts. Later options override earlier ones.
//
// If the slice is not sorted according to cmp, the behavior is undefined.
//
// The default behavior with no options is to insert v into s at its sorted
// position, ahead of any exact matches in s.
//
// The WithReplace or WithReplaceFunc options cause the first encountered exact
// match in s to be replaced, if it exists. Otherwise v is inserted normally.
//
// The WithKeepOriginal option only inserts v if no exact match is found,
// keeping existing exact matches in s.
//
// The WithDelete option will cause v to instead be deleted from s.
//
// The WithSearchOffset option causes the binary search to start at the given
// offset. This is useful when the caller already knows that v must occur after
// a certain point in s. The returned pos is relative to the start of the
// slice, regardless of the offset.
func UpdateSorted[S ~[]E, E any](s S, v E, cmp func(a, b E) int, opts ...Option[E]) (r S, pos int, found bool) {
	o := newOptions(opts...)
	pos, found = slices.BinarySearchFunc(s[o.searchOffset:], v, cmp)
	pos += o.searchOffset

	r = s
	if o.delete {
		if found {
			r = slices.Delete(s, pos, pos+1)
		}
		return
	}

	if !found || !o.replace {
		r = slices.Insert(s, pos, v)
		return
	}

	if o.replaceFunc != nil {
		o.replaceFunc(&r[pos], &v)
		return
	}

	r[pos] = v
	return
}

func DeleteSorted[S ~[]E, E any](s S, v E, cmp func(a, b E) int, opts ...Option[E]) S {
	s, _, _ = UpdateSorted(s, v, cmp, append(opts, WithDelete[E]())...)
	return s
}

func InsertSorted[S ~[]E, E any](s S, v E, cmp func(a, b E) int, opts ...Option[E]) (r S, pos int, found bool) {
	return UpdateSorted(s, v, cmp, append(opts, WithInsert[E]())...)
}

func MergeSorted[S ~[]E, E any](s1, s2 S, cmp func(a, b E) int, opts ...Option[E]) S {
	short, long := shortLong(s1, s2)
	var pos int
	for _, v := range short {
		long, pos, _ = UpdateSorted(long, v, cmp, append(opts,
			WithSearchOffset[E](pos))...)
	}
	return long
}
func shortLong[S ~[]E, E any](s1, s2 S) (short, long S) {
	short, long = s1, s2
	if len(short) > len(long) {
		short, long = long, short
	}
	return
}

func DiffSorted[S ~[]E, E any](a, b S, cmp func(a, b E) int) (added, removed S) {
	var i, j int
	for i < len(a) && j < len(b) {
		c := cmp(a[i], b[j])
		if c < 0 {
			removed = append(removed, a[i])
			i++
			continue
		}
		if c > 0 {
			added = append(added, b[j])
			j++
			continue
		}
		i++
		j++
	}
	removed = append(removed, a[i:]...)
	added = append(added, b[j:]...)
	return
}
