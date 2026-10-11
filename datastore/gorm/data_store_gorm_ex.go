//go:build ignore

// Your part of the data store for gorm: the connection. See data_store_gorm.go.

package dbal

import (
	"context"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func (ds *_DS) initDb() (err error) {
	ds.paramPrefix = "$"
	dsn := os.Getenv("DATABASE_URL") // postgres://user:pwd@host:5432/db
	ds.rootDb, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
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
