//go:build ignore

// Your part of the data store for database/sql (no ORM) and PostgreSQL: the connection. See data_store_no_orm.go.

package dbal

import (
	"context"
	"database/sql"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL
)

func (ds *_DS) initDb() (err error) {
	url := os.Getenv("DATABASE_URL") // postgres://user:pwd@host:5432/db
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}
	ds.db, err = sql.Open("pgx", url)
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
