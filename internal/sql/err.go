package sql

import (
	"errors"
	"fmt"
	"strings"
)

func newErrFailedQuery(err error, verb, query string, args ...any) error {
	if errors.Is(err, &ErrFailedQuery{}) {
		// Don't wrap errors that are already ErrFailedQuery.
		return err
	}
	return ErrFailedQuery{
		Err:   err,
		Verb:  verb,
		Query: query,
		Args:  args,
	}
}

type ErrFailedQuery struct {
	Err   error
	Verb  string
	Query string
	Args  []any
}

func (err ErrFailedQuery) Unwrap() error { return err.Err }
func (err ErrFailedQuery) Error() string {
	format := `
failed to %s: %v

query:
%s

args: %v
`[1:] // skip leading newline
	return fmt.Sprintf(format,
		err.Verb, err.Err,
		strings.TrimSpace(err.Query),
		err.Args,
	)
}
