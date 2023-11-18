// Package sql is a wrapper around database/sql that provides some convenience
// features for error handling and transactions. Most exported symbols are
// passthrough to the original package except for DB, Conn, Stmt, Tx, and Row.
package sql

import "database/sql"

const (
	LevelDefault         = sql.LevelDefault
	LevelReadUncommitted = sql.LevelReadUncommitted
	LevelReadCommitted   = sql.LevelReadCommitted
	LevelWriteCommitted  = sql.LevelWriteCommitted
	LevelRepeatableRead  = sql.LevelRepeatableRead
	LevelSnapshot        = sql.LevelSnapshot
	LevelSerializable    = sql.LevelSerializable
	LevelLinearizable    = sql.LevelLinearizable
)

var (
	ErrConnDone = sql.ErrConnDone
	ErrNoRows   = sql.ErrNoRows
	ErrTxDone   = sql.ErrTxDone

	Drivers  = sql.Drivers
	Register = sql.Register

	OpenDB = sql.OpenDB

	Named = sql.Named
)

type (
	ColumnType     = sql.ColumnType
	DBStats        = sql.DBStats
	IsolationLevel = sql.IsolationLevel
	NamedArg       = sql.NamedArg
	NullBool       = sql.NullBool
	NullByte       = sql.NullByte
	NullFloat64    = sql.NullFloat64
	NullInt16      = sql.NullInt16
	NullInt32      = sql.NullInt32
	NullInt64      = sql.NullInt64
	NullString     = sql.NullString
	NullTime       = sql.NullTime
	Out            = sql.Out
	RawBytes       = sql.RawBytes
	Result         = sql.Result
	Rows           = sql.Rows
	Scanner        = sql.Scanner
	TxOptions      = sql.TxOptions
)
