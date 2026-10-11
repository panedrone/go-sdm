//go:build ignore

// Your part of the data store for sqlx: the connection. See data_store_sqlx.go.

package dbal

import (
	"context"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL
	"github.com/jmoiron/sqlx"
)

func (ds *_DS) initDb() (err error) {
	ds.paramPrefix = "$"
	ds.db, err = sqlx.Open("pgx", os.Getenv("DATABASE_URL")) // postgres://user:pwd@host:5432/db
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
