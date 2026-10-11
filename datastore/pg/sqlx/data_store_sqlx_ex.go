//go:build ignore

// Your part of the data store for sqlx and PostgreSQL: the connection. See data_store_sqlx.go.

package dbal

import (
	"context"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL
	"github.com/jmoiron/sqlx"
)

func (ds *_DS) initDb() (err error) {
	url := os.Getenv("DATABASE_URL") // postgres://user:pwd@host:5432/db
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}
	ds.db, err = sqlx.Open("pgx", url)
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
