//go:build ignore

// Your part of the data store for sqlx and SQLite: the connection. See data_store_sqlx.go.

package dbal

import (
	"context"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // SQLite3, cgo; without cgo: modernc.org/sqlite, driver "sqlite"
)

func (ds *_DS) initDb() (err error) {
	file := os.Getenv("DATABASE_URL") // path to the database file
	if file == "" {
		file = "./app.sqlite3"
	}
	ds.db, err = sqlx.Open("sqlite3", file)
	return
}

var _ds = &_DS{}

func Ds() DataStore {
	return _ds
}

func init() {
	// generated DAO functions get DataStore via DS(ctx)
	DS = func(ctx context.Context) DataStore {
		return _ds
	}
}
