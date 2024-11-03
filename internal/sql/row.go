package sql

import (
	"database/sql"
)

type Row struct {
	*sql.Row
	query string
	args  []any
}

func (row *Row) Err() error {
	if err := row.Row.Err(); err != nil {
		return newErrFailedQuery(err, "query", row.query, row.args...)
	}
	return nil
}

func (row *Row) Scan(dest ...any) error {
	if err := row.Row.Scan(dest...); err != nil {
		return newErrFailedQuery(err, "scan", row.query, row.args...)
	}
	return nil
}
