// Package datastore holds the data store sources the generated DAO rely on:
// one variant per database (pg, mysql, sqlite) and framework (no_orm, sqlx, gorm), datastore/<db>/<orm>/.
// They are in the repository as is, so that they can be downloaded by hand; sdm init takes them from here too.
// Under "//go:build ignore": the go-sdm module needs no drivers, sqlx or gorm.
//
// The variants are built from the shared templates templates/*.go.txt (Generate, cli/dsgen):
// lines between "//sdm:if pg mysql" and "//sdm:end" stay only for the listed databases,
// "//sdm:else" is for the rest. The template is edited, the variants are regenerated; a test makes sure
// they do not diverge.
//
// templates/data_store_no_orm.go.txt comes from src/resources/data_store_no_orm.go of sdm-plugins
// (without Oracle and SQL Server).
package datastore

import (
	"bufio"
	"embed"
	"fmt"
	"go/format"
	"strings"
)

//go:embed templates/*.go.txt pg/*/*.go mysql/*/*.go sqlite/*/*.go
var fs embed.FS

// Variants are the values of sdm init -orm, the first is the default.
var Variants = []string{"no_orm", "sqlx", "gorm"}

// DBs are the databases, the first is the default.
var DBs = []string{"pg", "mysql", "sqlite"}

var dbNames = map[string]string{"pg": "PostgreSQL", "mysql": "MySQL", "sqlite": "SQLite"}

var ormNames = map[string]string{"no_orm": "database/sql (no ORM)", "sqlx": "sqlx", "gorm": "gorm"}

func check(variant, db string) error {
	if _, ok := ormNames[variant]; !ok {
		return fmt.Errorf("unknown data store %q, expected one of: %s", variant, strings.Join(Variants, ", "))
	}
	if _, ok := dbNames[db]; !ok {
		return fmt.Errorf("unknown database %q, expected one of: %s", db, strings.Join(DBs, ", "))
	}
	return nil
}

func fileNames(variant string) []string {
	return []string{"data_store_" + variant + ".go", "data_store_" + variant + "_ex.go"}
}

// Files maps a file name to its text for the DAO folder with package pkg: without "//go:build ignore" and the header.
func Files(variant, db, pkg string) (map[string]string, error) {
	if err := check(variant, db); err != nil {
		return nil, err
	}
	res := map[string]string{}
	for _, name := range fileNames(variant) {
		data, err := fs.ReadFile(db + "/" + variant + "/" + name)
		if err != nil {
			return nil, err
		}
		text, ok := withPackage(string(data), pkg)
		if !ok {
			return nil, fmt.Errorf("%s/%s/%s: 'package dbal' not found", db, variant, name)
		}
		res[name] = text
	}
	return res, nil
}

// withPackage drops everything before "package dbal" and renames the package to pkg.
// A checkout with core.autocrlf=true embeds CRLF: the result is LF.
func withPackage(src, pkg string) (string, bool) {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	_, body, ok := strings.Cut(src, "\npackage dbal\n")
	if !ok {
		return "", false
	}
	return "package " + pkg + "\n" + body, true
}

// Generate returns the files of variant db/variant from the templates: file name -> text.
func Generate(variant, db string) (map[string][]byte, error) {
	if err := check(variant, db); err != nil {
		return nil, err
	}
	vars := strings.NewReplacer("{{db}}", db, "{{DB}}", dbNames[db], "{{orm}}", variant, "{{ORM}}", ormNames[variant])
	res := map[string][]byte{}
	for _, name := range fileNames(variant) {
		tmpl, err := fs.ReadFile("templates/" + name + ".txt")
		if err != nil {
			return nil, err
		}
		text, err := specialize(string(tmpl), db)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		src, err := format.Source([]byte(vars.Replace(text)))
		if err != nil {
			return nil, fmt.Errorf("%s/%s/%s: %w", db, variant, name, err)
		}
		res[name] = src
	}
	return res, nil
}

// specialize keeps the lines of "//sdm:if ..." blocks for db and removes the markers. Blocks nest.
func specialize(tmpl, db string) (string, error) {
	var out strings.Builder
	var stack []bool // for each open block: whether its lines are for db
	keep := func() bool {
		for _, k := range stack {
			if !k {
				return false
			}
		}
		return true
	}
	sc := bufio.NewScanner(strings.NewReader(tmpl))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSuffix(sc.Text(), string(rune(13))) // CRLF
		t := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(t, "//sdm:if "):
			match := false
			for _, d := range strings.Fields(strings.TrimPrefix(t, "//sdm:if ")) {
				if _, ok := dbNames[d]; !ok {
					return "", fmt.Errorf("line %d: unknown database %q", n, d)
				}
				match = match || d == db
			}
			stack = append(stack, match)
			continue
		case t == "//sdm:else":
			if len(stack) == 0 {
				return "", fmt.Errorf("line %d: //sdm:else without //sdm:if", n)
			}
			stack[len(stack)-1] = !stack[len(stack)-1]
			continue
		case t == "//sdm:end":
			if len(stack) == 0 {
				return "", fmt.Errorf("line %d: //sdm:end without //sdm:if", n)
			}
			stack = stack[:len(stack)-1]
			continue
		}
		if keep() {
			out.WriteString(line)
			out.WriteByte('\n')
		}
	}
	if len(stack) > 0 {
		return "", fmt.Errorf("//sdm:if without //sdm:end")
	}
	return out.String(), sc.Err()
}
