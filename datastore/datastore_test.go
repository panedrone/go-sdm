package datastore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The variants in datastore/<db>/<orm>/ are exactly what is built from the templates (go run ./cli/dsgen).
func TestVariantsUpToDate(t *testing.T) {
	for _, db := range DBs {
		for _, orm := range Variants {
			files, err := Generate(orm, db)
			if err != nil {
				t.Fatal(err)
			}
			for name, want := range files {
				got, err := os.ReadFile(filepath.Join(db, orm, name))
				if err != nil {
					t.Fatal(err)
				}
				if strings.ReplaceAll(string(got), "\r\n", "\n") != string(want) {
					t.Errorf("%s/%s/%s is out of date: go run ./cli/dsgen", db, orm, name)
				}
			}
		}
	}
}

// The sources in a checkout with core.autocrlf=true have CRLF.
func TestWithPackageCRLF(t *testing.T) {
	src := "//go:build ignore\r\n\r\n// header\r\npackage dbal\r\n\r\nfunc f() {}\r\n"
	got, ok := withPackage(src, "store")
	if !ok {
		t.Fatal("'package dbal' not found")
	}
	if want := "package store\n\nfunc f() {}\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// SQLite and MySQL have no PostgreSQL cursor code, SQLite has no procedure code either.
func TestSpecific(t *testing.T) {
	cases := []struct {
		db, orm string
		has     []string
		hasNot  []string
	}{
		{"pg", "no_orm", []string{"RETURNING", "FetchPg", "exec2", "queryImplRc", `sql.Open("pgx"`}, nil},
		{"mysql", "no_orm", []string{"LastInsertId", "queryImplRc", `sql.Open("mysql"`, "type Out", "type InOut", "validateDest"}, []string{"RETURNING", "FetchPg", "exec2"}},
		{"sqlite", "no_orm", []string{"LastInsertId", `sql.Open("sqlite3"`}, []string{"RETURNING", "queryImplRc", "processExecParams", "type Out"}},
		{"sqlite", "gorm", []string{"sqlite.Open"}, []string{"FetchSql", "queryImplRc", "processExecParams"}},
		// the same DAO compile with any of the three: Out/InOut wherever there are procedures
		{"mysql", "gorm", []string{"queryImplRc", "type Out", "type InOut", "validateDest"}, []string{"FetchSql", "exec2"}},
		{"pg", "gorm", []string{"FetchSql", "exec2", "type Out", "type InOut", "validateDest"}, nil},
		{"pg", "sqlx", []string{"Rebind", "RETURNING"}, nil},
		{"mysql", "sqlx", []string{"LastInsertId", "queryImplRc", "type Out", "type InOut", "validateDest"}, []string{"RETURNING", "FetchPg", "exec2"}},
	}
	for _, c := range cases {
		files, err := Files(c.orm, c.db, "dbal")
		if err != nil {
			t.Fatal(err)
		}
		all := ""
		for _, f := range files {
			all += f
		}
		for _, s := range c.has {
			if !strings.Contains(all, s) {
				t.Errorf("%s/%s: %q expected", c.db, c.orm, s)
			}
		}
		for _, s := range c.hasNot {
			if strings.Contains(all, s) {
				t.Errorf("%s/%s: %q not expected", c.db, c.orm, s)
			}
		}
	}
}
