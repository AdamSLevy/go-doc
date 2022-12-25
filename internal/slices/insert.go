// Package slices provides some generic slice utility functions not offered by
// golang.org/x/exp/slices.
package slices

import "golang.org/x/exp/slices"

// InsertFunc inserts the values v... into the slice s at the first index i for
// which f(s[i]) returns true.
//
// If no such index exists, the values are appended. The return value is the
// resulting slice.
//
// If f panics, the panic is silently recovered and the original slice is
// returned, allowing for f to abort the insert.
func InsertFunc[S ~[]E, E any](s S, f func(i E) (insert bool), v ...E) (r S) {
	return InsertOrReplaceFunc(s, func(i E) (bool, bool) { return f(i), false }, v...)
}

// InsertOrReplaceFunc inserts or replaces the values v... into the slice s at
// the first index i for which f(s[i]) returns (true, _).
//
// If no such index exists, the values are appended.
//
// If f(s[i]) returns (true, true), the values v... will replace the current
// value at s[i].
//
// The return value is the resulting slice. If f panics, the panic is silently
// recovered and the original slice is returned, allowing f to abort the
// insert.
func InsertOrReplaceFunc[S ~[]E, E any](s S, f func(e E) (insert, replace bool), v ...E) (r S) {
	defer func() {
		if err := recover(); err != nil {
			r = s
		}
	}()
	var insert, replace bool
	idx := slices.IndexFunc(s, func(e E) bool {
		insert, replace = f(e)
		return insert
	})
	if idx < 0 {
		return append(s, v...)
	}
	if replace {
		return slices.Replace(s, idx, idx+1, v...)
	}
	return slices.Insert(s, idx, v...)
}
