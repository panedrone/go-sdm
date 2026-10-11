// Package skeleton — то, что `sdm init` выкладывает в проект.
//
// Init ничего не перезаписывает: если хоть один из его файлов уже есть, это ошибка,
// и не создаётся ничего (запуск по ошибке в готовом проекте безвреден). XSD принадлежат
// версии sdm, их обновляет RefreshXSD при каждой генерации.
package skeleton

import (
	"bufio"
	"bytes"
	"embed"
	"errors"
	"fmt"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/panedrone/go-sdm/datastore"
)

//go:embed files
var files embed.FS

type Options struct {
	ProjectDir string // папка go.mod
	SdmDir     string // относительно ProjectDir, с '/'
	DaoDir     string // относительно ProjectDir, с '/'; модели — в DaoDir/models
	ORM        string // data store: datastore.Variants, "" — первый
	DB         string // type-map и url: pg, mysql, sqlite; "" — pg
}

var xsds = []string{"sdm.xsd", "dao.xsd", "settings.xsd"}

// DBs — значения sdm init -db, первое — по умолчанию.
var DBs = []string{"pg", "mysql", "sqlite"}

var dbConf = map[string]struct{ name, url, user string }{
	"pg":     {"PostgreSQL", "postgres://localhost:5432/postgres", "postgres"},
	"mysql":  {"MySQL", "mysql://localhost:3306/mysql", "root"},
	"sqlite": {"SQLite", "sqlite:app.sqlite3", ""},
}

// Exists — в папке уже есть конфигурация (sdm.xml или settings.xml).
func Exists(projectDir, sdmDir string) bool {
	dir := filepath.Join(projectDir, filepath.FromSlash(sdmDir))
	for _, n := range []string{"sdm.xml", "settings.xml"} {
		if _, err := os.Stat(filepath.Join(dir, n)); err == nil {
			return true
		}
	}
	return false
}

// Init создаёт конфигурацию и data store. Возвращает созданные файлы.
func Init(o Options) ([]string, error) {
	module, err := readModule(filepath.Join(o.ProjectDir, "go.mod"))
	if err != nil {
		return nil, err
	}
	pkg := path.Base(o.DaoDir)
	if !token.IsIdentifier(pkg) {
		return nil, fmt.Errorf("the last element of the DAO folder must be a Go package name: %q", o.DaoDir)
	}
	orm := o.ORM
	if orm == "" {
		orm = datastore.Variants[0]
	}
	ds, err := datastore.Files(orm, pkg)
	if err != nil {
		return nil, err
	}
	db := o.DB
	if db == "" {
		db = DBs[0]
	}
	dc, ok := dbConf[db]
	if !ok {
		return nil, fmt.Errorf("unknown database %q, expected one of: %s", db, strings.Join(DBs, ", "))
	}
	tm, err := files.ReadFile("files/typemap_" + db + ".xml")
	if err != nil {
		return nil, err
	}
	assign := "0"
	if orm == "sqlx" {
		assign = "2" // ptr to DTO or ptr to slice of DTO-s
	}
	vars := strings.NewReplacer(
		"{{module}}", module,
		"{{sql}}", path.Join(o.SdmDir, "sql"),
		"{{dto}}", path.Join(o.DaoDir, "models"),
		"{{dao}}", o.DaoDir,
		"{{assign}}", assign,
		"{{db}}", dc.name,
		"{{url}}", dc.url,
		"{{user}}", dc.user,
		"{{type-map}}", strings.TrimRight(string(tm), "\r\n"),
	)
	sdmDir := filepath.Join(o.ProjectDir, filepath.FromSlash(o.SdmDir))
	daoDir := filepath.Join(o.ProjectDir, filepath.FromSlash(o.DaoDir))
	// src — файл скелета, text — готовый текст (data store)
	type step struct{ src, text, dst string }
	steps := []step{
		{src: "sdm.xml", dst: filepath.Join(sdmDir, "sdm.xml")},
		{src: "settings.xml", dst: filepath.Join(sdmDir, "settings.xml")},
	}
	for _, name := range []string{"data_store_" + orm + ".go", "data_store_" + orm + "_ex.go"} {
		steps = append(steps, step{text: ds[name], dst: filepath.Join(daoDir, name)})
	}
	for _, x := range xsds {
		steps = append(steps, step{src: x, dst: filepath.Join(sdmDir, x)})
	}
	// сначала проверка всего: либо создаётся всё, либо ничего
	var existing []string
	for _, s := range steps {
		if _, err := os.Stat(s.dst); err == nil {
			existing = append(existing, s.dst)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	if len(existing) > 0 {
		return nil, fmt.Errorf("already exists, nothing is written: %s", strings.Join(existing, ", "))
	}
	for _, dir := range []string{filepath.Join(sdmDir, "sql"), daoDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	var written []string
	for _, s := range steps {
		text := s.text
		if s.src != "" {
			data, err := files.ReadFile("files/" + s.src)
			if err != nil {
				return written, err
			}
			text = vars.Replace(string(data))
		}
		if err := os.WriteFile(s.dst, []byte(text), 0o644); err != nil {
			return written, err
		}
		written = append(written, s.dst)
	}
	return written, nil
}

// RefreshXSD — XSD этой версии sdm: подсказки в IDE должны соответствовать генератору.
// Возвращает обновлённые (созданные) файлы; совпадающие не трогает.
func RefreshXSD(projectDir, sdmDir string) ([]string, error) {
	dir := filepath.Join(projectDir, filepath.FromSlash(sdmDir))
	var res []string
	for _, x := range xsds {
		want, err := files.ReadFile("files/" + x)
		if err != nil {
			return res, err
		}
		dst := filepath.Join(dir, x)
		if got, err := os.ReadFile(dst); err == nil && bytes.Equal(got, want) {
			continue
		}
		if err := os.WriteFile(dst, want, 0o644); err != nil {
			return res, err
		}
		res = append(res, dst)
	}
	return res, nil
}

// readModule — значение "module" из go.mod.
func readModule(goMod string) (string, error) {
	data, err := os.ReadFile(goMod)
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("%s not found: run sdm in the folder of go.mod (or 'go mod init' first)", goMod)
	}
	if err != nil {
		return "", err
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) >= 2 && f[0] == "module" {
			return strings.Trim(f[1], `"`), nil
		}
	}
	return "", fmt.Errorf("%s: 'module' not found", goMod)
}
