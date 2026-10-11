// Package datastore — исходники data store, на которые опираются сгенерированные DAO:
// no_orm (database/sql), sqlx, gorm. Лежат в репозитории как есть, чтобы их можно было
// скачать руками; sdm init берёт их отсюда же. Под "//go:build ignore": модулю go-sdm
// зависимости sqlx и gorm не нужны.
//
// no_orm/data_store_no_orm.go — копия src/resources/data_store_no_orm.go из sdm-plugins;
// при правке шаблона там — обновить и здесь.
package datastore

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed no_orm/*.go sqlx/*.go gorm/*.go
var fs embed.FS

// Variants — значения sdm init -orm, первое — по умолчанию.
var Variants = []string{"no_orm", "sqlx", "gorm"}

// Files — имя файла -> текст для папки DAO с пакетом pkg: без "//go:build ignore" и шапки.
func Files(variant, pkg string) (map[string]string, error) {
	known := false
	for _, v := range Variants {
		known = known || v == variant
	}
	if !known {
		return nil, fmt.Errorf("unknown data store %q, expected one of: %s", variant, strings.Join(Variants, ", "))
	}
	res := map[string]string{}
	for _, name := range []string{"data_store_" + variant + ".go", "data_store_" + variant + "_ex.go"} {
		data, err := fs.ReadFile(variant + "/" + name)
		if err != nil {
			return nil, err
		}
		_, body, ok := strings.Cut(string(data), "\npackage dbal\n")
		if !ok {
			return nil, fmt.Errorf("%s/%s: 'package dbal' not found", variant, name)
		}
		res[name] = "package " + pkg + "\n" + body
	}
	return res, nil
}
