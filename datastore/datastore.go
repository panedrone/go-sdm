// Package datastore — исходники data store, на которые опираются сгенерированные DAO:
// по варианту на базу (pg, mysql, sqlite) и фреймворк (no_orm, sqlx, gorm), datastore/<db>/<orm>/.
// Лежат в репозитории как есть, чтобы их можно было скачать руками; sdm init берёт их отсюда же.
// Под "//go:build ignore": модулю go-sdm драйверы, sqlx и gorm не нужны.
//
// Варианты собираются из общих шаблонов templates/*.go.txt (Generate, cli/dsgen):
// строки между "//sdm:if pg mysql" и "//sdm:end" остаются только для перечисленных баз,
// "//sdm:else" — для остальных. Правится шаблон, варианты — перегенерируются; тест следит,
// чтобы они не разошлись.
//
// templates/data_store_no_orm.go.txt — от src/resources/data_store_no_orm.go из sdm-plugins
// (без Oracle и SQL Server).
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

// Variants — значения sdm init -orm, первое — по умолчанию.
var Variants = []string{"no_orm", "sqlx", "gorm"}

// DBs — базы, первая — по умолчанию.
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

// Files — имя файла -> текст для папки DAO с пакетом pkg: без "//go:build ignore" и шапки.
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
		_, body, ok := strings.Cut(string(data), "\npackage dbal\n")
		if !ok {
			return nil, fmt.Errorf("%s/%s/%s: 'package dbal' not found", db, variant, name)
		}
		res[name] = "package " + pkg + "\n" + body
	}
	return res, nil
}

// Generate — файлы варианта db/variant из шаблонов: имя файла -> текст.
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

// specialize — оставляет строки блоков "//sdm:if ..." для db, маркеры убирает. Блоки вкладываются.
func specialize(tmpl, db string) (string, error) {
	var out strings.Builder
	var stack []bool // для каждого открытого блока: его строки — для db
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
