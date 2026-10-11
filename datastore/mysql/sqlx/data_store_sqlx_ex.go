//go:build ignore

// Your part of the data store for sqlx and MySQL: the connection. See data_store_sqlx.go.

package dbal

import (
	"context"
	"os"

	_ "github.com/go-sql-driver/mysql" // MySQL
	"github.com/jmoiron/sqlx"
)

func (ds *_DS) initDb() (err error) {
	dsn := os.Getenv("DATABASE_URL") // user:pwd@tcp(host:3306)/db?parseTime=true
	if dsn == "" {
		dsn = "root:root@tcp(localhost:3306)/mysql?parseTime=true" // parseTime: DATETIME -> time.Time
	}
	ds.db, err = sqlx.Open("mysql", dsn)
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
