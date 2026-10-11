//go:build ignore

// Your part of the data store for gorm and MySQL: the connection. See data_store_gorm.go.

package dbal

import (
	"context"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func (ds *_DS) initDb() (err error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" { // user:pwd@tcp(host:3306)/db?parseTime=true
		dsn = "root:root@tcp(localhost:3306)/mysql?parseTime=true" // parseTime: DATETIME -> time.Time
	}
	dialector := mysql.Open(dsn)
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
