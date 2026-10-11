//go:build ignore

// Your part of the data store for database/sql (no ORM): the connection. See data_store_no_orm.go.

package dbal

import (
	"context"
	"database/sql"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL
)

func (ds *_DS) initDb() (err error) {
	ds.paramPrefix = "$"
	ds.db, err = sql.Open("pgx", os.Getenv("DATABASE_URL")) // postgres://user:pwd@host:5432/db
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
