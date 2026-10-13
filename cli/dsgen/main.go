// dsgen is design-time: it builds the data store for each database from the templates datastore/templates
// into datastore/<db>/<orm>/. Run it from the go-sdm root after editing the templates:
//
//	go run ./cli/dsgen
//
// The datastore package test fails if the variants diverge from the templates.
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
