// Package skeleton is what `sdm init` puts into a project.
//
// Init overwrites nothing: if any of its files already exists, it is an error,
// and nothing is created (running it by mistake in a ready project is harmless). The XSD belong
// to the sdm version, RefreshXSD updates them on every generation. The data store too, but only on
// request (sdm datastore): RefreshDataStore.
package skeleton

import (
	"bufio"
	"bytes"
	"embed"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/panedrone/go-sdm/datastore"
	"github.com/panedrone/go-sdm/internal/sdmxml"
)

//go:embed files
var files embed.FS

type Options struct {
	ProjectDir string // the folder of go.mod
	SdmDir     string // relative to ProjectDir, with '/'
	DaoDir     string // relative to ProjectDir, with '/'; models go to DaoDir/models
	ORM        string // data store: datastore.Variants, "" means the first
	DB         string // type-map and url: pg, mysql, sqlite; "" means pg
}

var xsds = []string{"sdm.xsd", "dao.xsd", "settings.xsd"}

// DBs are the values of sdm init -db, the first is the default.
var DBs = datastore.DBs

// dbConf holds <go-db> and the plug-in's <jdbc>: one settings.xml for both sdm and the IDE.
var dbConf = map[string]struct{ name, url, user, jar, class, jdbcURL string }{
	"pg": {"PostgreSQL", "postgres://localhost:5432/postgres", "postgres",
		"postgresql-42.7.3.jar", "org.postgresql.Driver", "jdbc:postgresql://localhost:5432/postgres"},
	"mysql": {"MySQL", "mysql://localhost:3306/mysql", "root",
		"mysql-connector-j-26.7.0.jar", "com.mysql.cj.jdbc.Driver", "jdbc:mysql://localhost:3306/mysql"},
	"sqlite": {"SQLite", "sqlite:app.sqlite3", "",
		"sqlite-jdbc-3.51.1.0.jar", "org.sqlite.JDBC", "jdbc:sqlite:$PROJECT_DIR$/app.sqlite3"},
}

// Init creates the configuration and the data store. It returns the created files.
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
	db := o.DB
	if db == "" {
		db = DBs[0]
	}
	dc, ok := dbConf[db]
	if !ok {
		return nil, fmt.Errorf("unknown database %q, expected one of: %s", db, strings.Join(DBs, ", "))
	}
	ds, err := datastore.Files(orm, db, pkg) // the data store for the database
	if err != nil {
		return nil, err
	}
	tm, err := files.ReadFile("files/typemap_" + db + ".xml")
	if err != nil {
		return nil, err
	}
	assign := "3"
	if orm == "no_orm" {
		assign = "0" // by column name (database/sql, gorm)
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
		"{{jdbc-jar}}", dc.jar,
		"{{jdbc-class}}", dc.class,
		"{{jdbc-url}}", dc.jdbcURL,
		"{{type-map}}", strings.TrimRight(string(tm), "\r\n"),
	)
	sdmDir := filepath.Join(o.ProjectDir, filepath.FromSlash(o.SdmDir))
	daoDir := filepath.Join(o.ProjectDir, filepath.FromSlash(o.DaoDir))
	// src is a skeleton file, text is ready text (data store)
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
	// check everything first: either everything is created, or nothing
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

// RefreshXSD writes the XSD of this sdm version: IDE hints must match the generator.
// It returns the updated (created) files; it does not touch the matching ones.
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

// DataStoreFile is a data store found by RefreshDataStore: ORM by its name data_store_<orm>.go, DB by
// <go-db url> (or -db).
type DataStoreFile struct {
	Path    string
	ORM, DB string
	Updated bool // false: it matched this sdm version
}

// RefreshDataStore rewrites data_store_<orm>.go in the DAO folder of settings.xml with the one of this
// sdm version, for each data store found there. data_store_<orm>_ex.go (connection) is yours, it is not
// touched. db "" means the database of <go-db url>. It returns the data stores found; the matching ones
// are not touched.
func RefreshDataStore(projectDir, sdmDir, db string) ([]DataStoreFile, error) {
	dir := filepath.Join(projectDir, filepath.FromSlash(sdmDir))
	s, err := sdmxml.LoadSettings(dir)
	if err != nil {
		return nil, err
	}
	if db == "" {
		if s.GoDB == nil {
			return nil, fmt.Errorf("%s/settings.xml: no <go-db url>, the database is unknown (-db %s)", sdmDir, strings.Join(DBs, "|"))
		}
		if db, err = dbOf(s.GoDB.URL); err != nil {
			return nil, err
		}
	}
	daoDir := filepath.Join(projectDir, filepath.FromSlash(s.Dao.Scope))
	var res []DataStoreFile
	found := false
	for _, orm := range datastore.Variants {
		name := "data_store_" + orm + ".go"
		dst := filepath.Join(daoDir, name)
		got, err := os.ReadFile(dst)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return res, err
		}
		found = true
		pkg, err := packageName(dst, got)
		if err != nil {
			return res, err
		}
		ds, err := datastore.Files(orm, db, pkg)
		if err != nil {
			return res, err
		}
		f := DataStoreFile{Path: dst, ORM: orm, DB: db}
		if string(got) != ds[name] {
			if err := os.WriteFile(dst, []byte(ds[name]), 0o644); err != nil {
				return res, err
			}
			f.Updated = true
		}
		res = append(res, f)
	}
	if !found {
		return nil, fmt.Errorf("no data_store_{%s}.go in %s (sdm init creates it)", strings.Join(datastore.Variants, ","), daoDir)
	}
	return res, nil
}

// dbOf maps <go-db url> to pg, mysql or sqlite.
func dbOf(url string) (string, error) {
	u := strings.TrimPrefix(strings.TrimSpace(url), "jdbc:")
	switch {
	case strings.HasPrefix(u, "sqlite:"):
		return "sqlite", nil
	case strings.HasPrefix(u, "mysql:"):
		return "mysql", nil
	case strings.HasPrefix(u, "postgres:"), strings.HasPrefix(u, "postgresql:"):
		return "pg", nil
	}
	return "", fmt.Errorf("the database of <go-db url=%q> is unknown (-db %s)", url, strings.Join(DBs, "|"))
}

// packageName is the package clause of a Go file: the data store keeps the package of the DAO folder.
func packageName(file string, src []byte) (string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), file, src, parser.PackageClauseOnly)
	if err != nil {
		return "", err
	}
	return f.Name.Name, nil
}

// readModule returns the "module" value of go.mod.
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
