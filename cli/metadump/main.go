// metadump — design-time сверка слоя метаданных с Java-плагином на живой базе.
//
// Печатает meta.Metadata в формате MetaDump.java (sdm-plugins/src_harness), чтобы два
// вывода сравнивались обычным diff. Пароль — из SDM_PWD.
//
//	go run ./cli/metadump -url jdbc:postgresql://host:5432/db -user postgres cli/metadump/my_tests_pg.txt
//	go run ./cli/metadump -url sqlite::memory: internal/meta/testdata/sqlite.txt
//
// SQLite сверяется и в go test (internal/meta/sqlite) — эта утилита нужна для PostgreSQL.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/panedrone/go-sdm/internal/meta"
	"github.com/panedrone/go-sdm/internal/meta/metadump"
	"github.com/panedrone/go-sdm/internal/meta/pg"
	"github.com/panedrone/go-sdm/internal/meta/sqlite"
)

func main() {
	url := flag.String("url", "", "jdbc:postgresql://..., postgres://... or sqlite:<file>")
	user := flag.String("user", "", "user")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: metadump -url <url> [-user <user>] <inputs-file>")
		os.Exit(2)
	}
	if err := run(*url, *user, os.Getenv("SDM_PWD"), flag.Arg(0)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(url, user, pwd, inputs string) error {
	md, err := open(url, user, pwd)
	if err != nil {
		return err
	}
	defer md.Close()
	f, err := os.Open(inputs)
	if err != nil {
		return err
	}
	defer f.Close()
	return metadump.Run(md, f, os.Stdout)
}

func open(url, user, pwd string) (meta.Metadata, error) {
	if file, ok := strings.CutPrefix(url, "sqlite:"); ok {
		return sqlite.Open(file)
	}
	cs, err := pg.ConnString(url, user, pwd)
	if err != nil {
		return nil, err
	}
	return pg.Open(context.Background(), cs)
}
