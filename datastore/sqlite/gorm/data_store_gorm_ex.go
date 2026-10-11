//go:build ignore

// Your part of the data store for gorm and SQLite: the connection. See data_store_gorm.go.

package dbal

import (
	"context"
	"os"

	"gorm.io/driver/sqlite" // cgo; without cgo: github.com/glebarez/sqlite
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func (ds *_DS) initDb() (err error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" { // path to the database file
		dsn = "./app.sqlite3"
	}
	dialector := sqlite.Open(dsn)
	ds.rootDb, err = gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
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
