package datastore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Варианты в datastore/<db>/<orm>/ — ровно то, что собирается из шаблонов (go run ./cli/dsgen).
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

// Под SQLite и MySQL нет кода курсоров PostgreSQL, под SQLite — и процедур.
func TestSpecific(t *testing.T) {
	cases := []struct {
		db, orm string
		has     []string
		hasNot  []string
	}{
		{"pg", "no_orm", []string{"RETURNING", "FetchPg", "exec2", "queryImplRc", `sql.Open("pgx"`}, nil},
		{"mysql", "no_orm", []string{"LastInsertId", "queryImplRc", `sql.Open("mysql"`}, []string{"RETURNING", "FetchPg", "exec2", "type Out"}},
		{"sqlite", "no_orm", []string{"LastInsertId", `sql.Open("sqlite3"`}, []string{"RETURNING", "queryImplRc", "processExecParams", "type Out"}},
		{"sqlite", "gorm", []string{"sqlite.Open"}, []string{"FetchSql", "queryImplRc", "processExecParams"}},
		{"pg", "sqlx", []string{"Rebind", "RETURNING"}, nil},
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
