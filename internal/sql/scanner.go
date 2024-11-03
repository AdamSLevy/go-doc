package sql

var (
	_ RowScanner = (*Row)(nil)
	_ RowScanner = (*Rows)(nil)
)

type RowScanner interface {
	Scan(dest ...any) error
	Err() error
}
