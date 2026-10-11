// dsgen — design-time: собирает data store под каждую базу из шаблонов datastore/templates
// в datastore/<db>/<orm>/. Запускается из корня go-sdm после правки шаблонов:
//
//	go run ./cli/dsgen
//
// Тест пакета datastore падает, если варианты разошлись с шаблонами.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/panedrone/go-sdm/datastore"
)

func main() {
	for _, db := range datastore.DBs {
		for _, orm := range datastore.Variants {
			files, err := datastore.Generate(orm, db)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			dir := filepath.Join("datastore", db, orm)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			for name, text := range files {
				if err := os.WriteFile(filepath.Join(dir, name), text, 0o644); err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
			}
		}
	}
}
