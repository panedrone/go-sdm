// sdm is a DAO generator for Go from sdm.xml and the metadata of a live database (PostgreSQL, MySQL, SQLite).
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/panedrone/go-sdm/datastore"
	"github.com/panedrone/go-sdm/internal/assist"
	"github.com/panedrone/go-sdm/internal/cg"
	"github.com/panedrone/go-sdm/internal/gen"
	"github.com/panedrone/go-sdm/internal/meta"
	"github.com/panedrone/go-sdm/internal/sdmxml"
	"github.com/panedrone/go-sdm/internal/skeleton"
)

const version = "1.0.7"

const usage = `sdm %s — SQL DAL Maker for Go

usage:
  sdm init [-dir go-sdm] [-dao internal/dbal] [-orm no_orm|sqlx|gorm] [-db pg|mysql|sqlite]
                                  create sdm.xml, settings.xml, XSD and the data store
                                  (database/sql, sqlx or gorm; no_orm by default),
                                  type map and url for pg (default), mysql or sqlite;
                                  an error if any of them exists, nothing is overwritten
  sdm generate|gen [-dir go-sdm|-c N|all] [-m] [-d] [class ...]
                                  generate models and DAO into the folders of settings.xml
  sdm validate|val [-dir go-sdm|-c N|all] [-m] [-d] [class ...]
                                  exit 1 if the generated code is out of date (for CI)
  sdm list|ls [-dir go-sdm|-c N|all] [-m] [-d] [-l] [class ...]
                                  numbered classes of sdm.xml in the order of declaration,
                                  -l: a table with files and dates
  sdm xml -orm no_orm|sqlx|gorm [-dir go-sdm|-c N] [-m] [-d] [-force]
                                  starting XML from the database schema, to the console:
                                  -m <dto-class> for each table and view, -d <dao-class>
                                  for each, named by the table, with <crud/> and FK access;
                                  -orm (required): DTO classes Task, sqlx-Task or gorm-Task;
                                  classes already in sdm.xml are skipped, -force: not;
                                  to a file: sdm xml -orm gorm -m > models.xml
  sdm datastore|ds [-dir go-sdm|-c N|all] [-db pg|mysql|sqlite]
                                  update data_store_<orm>.go in the DAO folder to this sdm
                                  version (the database of <go-db url>, or -db);
                                  data_store_<orm>_ex.go, your connection, is not touched
  sdm version|-v|--version
  sdm help

-m: models only, -d: DAO only.

-c N: the configuration N of .sdm, the root file of the IDE plug-ins next to go.mod (its lines
<dir>/go.dal, numbered from 1); -c all: each of them, one after another (not for xml). Without -dir and -c: the only one of .sdm; several are printed as
a table with the command for each, and the console asks for the number; no go.dal in .sdm
(or no .sdm): -dir go-sdm.

class: a number of sdm ls or a name; several, by spaces or commas: sdm gen 1,3 5 TasksDao.
None: all classes.

Run it in the folder of go.mod. The database is <go-db url user pwd> of settings.xml;
SDM_PWD, if set, overrides "pwd".
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	cmd := ""
	if len(args) > 0 && (args[0] == "-v" || args[0] == "--version" || args[0] == "-version") {
		cmd, args = "version", args[1:]
	} else if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "":
		fmt.Fprintf(stderr, usage, version)
		return 2
	case "init":
		o, ok := parseInit("init", args, stderr)
		if !ok {
			return 2
		}
		return runInit(o, stdout, stderr)
	case "generate", "gen":
		return runGenerate(cmd, args, true, stdout, stderr)
	case "validate", "val":
		return runGenerate(cmd, args, false, stdout, stderr)
	case "list", "ls":
		return runList(cmd, args, stdout, stderr)
	case "xml":
		return runXML(cmd, args, stdout, stderr)
	case "datastore", "ds":
		return runDataStore(cmd, args, stdout, stderr)
	case "version":
		fmt.Fprintln(stdout, version)
		return 0
	case "help":
		fmt.Fprintf(stdout, usage, version)
		return 0
	default:
		printErr(stderr, fmt.Errorf("unknown command %q", cmd))
		fmt.Fprintf(stderr, "\n"+usage, version)
		return 2
	}
}

func newFlags(name string, stderr io.Writer) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprintf(stderr, usage, version) }
	dir := fs.String("dir", "go-sdm", "folder of sdm.xml and settings.xml, relative to go.mod")
	return fs, dir
}

func parseInit(name string, args []string, stderr io.Writer) (skeleton.Options, bool) {
	fs, dir := newFlags(name, stderr)
	dao := fs.String("dao", "internal/dbal", "folder of generated DAO, relative to go.mod; models go to <dao>/models")
	orm := fs.String("orm", datastore.Variants[0], "data store: "+strings.Join(datastore.Variants, ", "))
	db := fs.String("db", skeleton.DBs[0], "database of the type map and url: "+strings.Join(skeleton.DBs, ", "))
	if err := fs.Parse(args); err != nil {
		return skeleton.Options{}, false
	}
	return skeleton.Options{ProjectDir: ".", SdmDir: cleanRel(*dir), DaoDir: cleanRel(*dao), ORM: *orm, DB: *db}, true
}

// parse allows flags mixed with classes: sdm gen 1,3 -dir x 5 (the flag package stops
// at the first non-flag).
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	var rest []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return rest, nil
		}
		rest = append(rest, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

func runInit(o skeleton.Options, stdout, stderr io.Writer) int {
	written, err := skeleton.Init(o)
	for _, f := range written {
		fmt.Fprintln(stdout, "created: "+f)
	}
	if err != nil {
		printErr(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "next: describe classes in %s/sdm.xml, set <go-db url> in %s/settings.xml, then sdm gen\n",
		o.SdmDir, o.SdmDir)
	return 0
}

// runDataStore updates the data store of the DAO folder to this sdm version.
func runDataStore(cmd string, args []string, stdout, stderr io.Writer) int {
	fs, dir := newFlags("datastore", stderr)
	db := fs.String("db", "", "database of the data store, by <go-db url> if not given: "+strings.Join(skeleton.DBs, ", "))
	c := configFlag(fs)
	rest, err := parse(fs, args)
	if err != nil {
		return 2
	}
	if len(rest) > 0 {
		printErr(stderr, fmt.Errorf("datastore takes no classes: %s", strings.Join(rest, " ")))
		return 2
	}
	if *db != "" && !slices.Contains(skeleton.DBs, *db) {
		printErr(stderr, fmt.Errorf("datastore: -db %s expected", strings.Join(skeleton.DBs, "|")))
		return 2
	}
	return forEachConfig(fs, *dir, *c, cmd, args, stdout, stderr, func(sdmDir string) int {
		found, err := skeleton.RefreshDataStore(".", sdmDir, *db)
		for _, f := range found {
			state := "up to date"
			if f.Updated {
				state = "updated"
			}
			// which one and why: ORM by the file name, DB by <go-db url> or -db
			fmt.Fprintf(stdout, "%s: %s (%s for %s)\n", state, f.Path, f.ORM, f.DB)
		}
		if err != nil {
			printErr(stderr, err)
			return 1
		}
		return 0
	})
}

// runGenerate is generate (write == true) or validate.
func runGenerate(cmd string, args []string, write bool, stdout, stderr io.Writer) int {
	fs, dir := newFlags(map[bool]string{true: "generate", false: "validate"}[write], stderr)
	models, daos := kindFlags(fs)
	c := configFlag(fs)
	classes, err := parse(fs, args)
	if err != nil {
		return 2
	}
	return forEachConfig(fs, *dir, *c, cmd, args, stdout, stderr, func(sdmDir string) int {
		return generate(sdmDir, classes, *models, *daos, write, stdout, stderr)
	})
}

func kindFlags(fs *flag.FlagSet) (models, daos *bool) {
	return fs.Bool("m", false, "models only"), fs.Bool("d", false, "DAO only")
}

func runList(cmd string, args []string, stdout, stderr io.Writer) int {
	fs, dir := newFlags("list", stderr)
	models, daos := kindFlags(fs)
	long := fs.Bool("l", false, "a table with files and dates")
	c := configFlag(fs)
	classes, err := parse(fs, args)
	if err != nil {
		return 2
	}
	return forEachConfig(fs, *dir, *c, cmd, args, stdout, stderr, func(sdmDir string) int {
		return list(sdmDir, classes, *models, *daos, *long, stdout, stderr)
	})
}

// list prints the classes of one configuration.
func list(sdmDir string, classes []string, models, daos, long bool, stdout, stderr io.Writer) int {
	entries, err := gen.List(".", sdmDir)
	if err != nil {
		printErr(stderr, err)
		return 1
	}
	all := make([]sdmxml.Class, len(entries))
	for i, e := range entries {
		all[i] = e.Class
	}
	only, err := gen.Select(all, classes, models, daos)
	if err != nil {
		printErr(stderr, err)
		return 2
	}
	var shown []gen.Entry
	for _, e := range entries {
		if only == nil || only[e.Name] {
			shown = append(shown, e)
		}
	}
	if !long {
		items := make([]string, len(shown))
		for i, e := range shown {
			items[i] = fmt.Sprintf("%d:%s", e.N, e.Name)
		}
		if len(items) > 0 {
			fmt.Fprintln(stdout, strings.Join(items, "  "))
		}
		return 0
	}
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "#\tkind\tclass\tfile\tmodified")
	for _, e := range shown {
		kind, mod := "model", "-"
		if e.Dao {
			kind = "dao"
		}
		if !e.Modified.IsZero() {
			mod = e.Modified.Format("2006-01-02 15:04:05")
		}
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\n", e.N, kind, e.Name, e.Path, mod)
	}
	_ = tw.Flush()
	return 0
}

func generate(dir string, classes []string, models, daos, write bool, stdout, stderr io.Writer) int {
	s, err := loadSettings(dir)
	if err != nil {
		printErr(stderr, err)
		return 1
	}
	sdm, err := sdmxml.LoadSdm(dir)
	if err != nil {
		printErr(stderr, err)
		return 1
	}
	only, err := gen.Select(sdm.Classes, classes, models, daos)
	if err != nil {
		printErr(stderr, err)
		return 2
	}
	st := newStatus(stderr)
	st.title = map[bool]string{true: "generating", false: "validating"}[write]
	md, err := connect(s, st)
	if err != nil {
		printErr(stderr, err)
		return 1
	}
	defer md.Close()
	res, err := gen.Run(".", dir, md, only, st.progress)
	if err != nil {
		st.clear()
		printErr(stderr, err)
		return 1
	}
	st.finish() // stays: how long it took
	if write {
		xsd, err := skeleton.RefreshXSD(".", dir)
		for _, f := range xsd {
			fmt.Fprintln(stdout, "updated: "+f)
		}
		if err != nil {
			printErr(stderr, err)
			return 1
		}
	}
	changes, err := gen.Sync(".", res, write)
	for _, c := range changes {
		switch {
		case write && c.Missing:
			fmt.Fprintln(stdout, "created: "+c.Path)
		case write:
			fmt.Fprintln(stdout, "updated: "+c.Path)
		case c.Missing:
			fmt.Fprintln(stdout, red(stdout, "missing: "+c.Path))
		default:
			fmt.Fprintln(stdout, red(stdout, "out of date: "+c.Path))
		}
	}
	if err != nil {
		printErr(stderr, err)
		return 1
	}
	if !write && len(changes) > 0 {
		return 1
	}
	if len(changes) == 0 {
		fmt.Fprintf(stdout, "up to date: %d classes\n", len(res.Dto)+len(res.Dao))
	}
	return 0
}

// loadSettings: settings.xml with <go-db>.
func loadSettings(dir string) (*sdmxml.Settings, error) {
	s, err := sdmxml.LoadSettings(dir)
	if err != nil {
		return nil, err
	}
	if s.GoDB == nil {
		return nil, fmt.Errorf("%s/settings.xml has no <go-db>, the database to generate from. Add one of:\n\n"+
			"    <go-db url=\"postgres://localhost:5432/db\" user=\"postgres\" pwd=\"\"/>\n"+
			"    <go-db url=\"mysql://localhost:3306/db\" user=\"root\" pwd=\"\"/>\n"+
			"    <go-db url=\"sqlite:app.sqlite3\" user=\"\" pwd=\"\"/>   (the file is relative to go.mod)", dir)
	}
	return s, nil
}

// connect opens the database of <go-db>; SDM_PWD overrides pwd.
func connect(s *sdmxml.Settings, st *status) (meta.Metadata, error) {
	db := *s.GoDB
	if pwd, ok := os.LookupEnv("SDM_PWD"); ok {
		db.Pwd = pwd
	}
	label := dbLabel(db.URL)
	st.show("connecting to %s ...", label)
	start := time.Now()
	md, err := gen.OpenMetadata(".", db)
	if err == nil {
		st.show("connected to %s in %.2fs", label, time.Since(start).Seconds())
	}
	st.keep() // stays, the error (if any) goes below it
	return md, err
}

// runXML prints the starting XML from the database schema: <dto-class> (-m) and <dao-class>
// with <crud> and FK access (-d), both if neither or both are given.
func runXML(cmd string, args []string, stdout, stderr io.Writer) int {
	fs, dir := newFlags("xml", stderr)
	models, daos := kindFlags(fs)
	orm := fs.String("orm", "", "the model of DTO classes, required: "+strings.Join(datastore.Variants, ", "))
	force := fs.Bool("force", false, "all tables and views, sdm.xml is not read")
	c := configFlag(fs)
	rest, err := parse(fs, args)
	if err != nil {
		return 2
	}
	if len(rest) > 0 {
		printErr(stderr, fmt.Errorf("xml takes no classes: %s", strings.Join(rest, " ")))
		return 2
	}
	if !slices.Contains(datastore.Variants, *orm) {
		printErr(stderr, fmt.Errorf("xml: -orm %s expected", strings.Join(datastore.Variants, "|")))
		return 2
	}
	model := *orm // the prefix of DTO class names: sqlx-Task, gorm-Task
	if model == datastore.Variants[0] {
		model = "" // database/sql: Task
	}
	if !*models && !*daos {
		*models, *daos = true, true
	}
	if strings.EqualFold(*c, allConfigs) {
		printErr(stderr, errors.New("xml: -c all is not for xml, it prints one configuration"))
		return 2
	}
	sdmDir := ""
	if code := forEachConfig(fs, *dir, *c, cmd, args, stdout, stderr, func(d string) int { sdmDir = d; return 0 }); code != 0 {
		return code
	}
	s, err := loadSettings(sdmDir)
	if err != nil {
		printErr(stderr, err)
		return 1
	}
	sdm := &sdmxml.Sdm{} // -force: nothing is in use
	if !*force {
		if sdm, err = sdmxml.LoadSdm(sdmDir); err != nil {
			printErr(stderr, err)
			return 1
		}
	}
	st := newStatus(stderr)
	md, err := connect(s, st)
	if err != nil {
		printErr(stderr, err)
		return 1
	}
	defer md.Close()
	cat, ok := md.(meta.Catalog)
	if !ok {
		st.clear()
		printErr(stderr, errors.New("this database cannot list its tables"))
		return 1
	}
	st.show("reading the schema ...")
	var out []string
	if *models {
		x, err := assist.DtoClasses(cat, model, dtoClasses(sdm))
		if err != nil {
			st.clear()
			printErr(stderr, err)
			return 1
		}
		if x != "" {
			out = append(out, x)
		}
	}
	if *daos {
		x, err := assist.DaoClasses(cat, model, crudDtos(sdm), daoClasses(sdm))
		if err != nil {
			st.clear()
			printErr(stderr, err)
			return 1
		}
		if x != "" {
			out = append(out, x)
		}
	}
	st.clear()
	if len(out) == 0 {
		// stderr: the output stays empty for > and |, the console still sees why
		tables, err := cat.Tables(true)
		if err != nil {
			printErr(stderr, err)
			return 1
		}
		if len(tables) == 0 {
			fmt.Fprintln(stderr, "nothing to print: no tables and views in the database")
		} else {
			fmt.Fprintf(stderr, "nothing to print: all %d tables and views are in %s/sdm.xml (-force: all of them anyway)\n",
				len(tables), sdmDir)
		}
		return 0
	}
	_, _ = fmt.Fprint(stdout, strings.Join(out, "\n"))
	return 0
}

// dtoClasses: the names of <dto-class> in sdm.xml, without the model (sqlx-Task -> Task): one
// table, one model file, whatever the model.
func dtoClasses(sdm *sdmxml.Sdm) map[string]bool {
	res := map[string]bool{}
	for _, c := range sdm.DtoClasses {
		res[cg.ClassNameWithoutModel(c.Name)] = true
	}
	return res
}

// daoClasses: the names of <dao-class> in sdm.xml.
func daoClasses(sdm *sdmxml.Sdm) map[string]bool {
	res := map[string]bool{}
	for _, c := range sdm.DaoClasses {
		res[c.Name] = true
	}
	return res
}

// crudDtos: the DTO classes of <crud> in all DAO classes, without the model.
func crudDtos(sdm *sdmxml.Sdm) map[string]bool {
	res := map[string]bool{}
	for _, c := range sdm.DaoClasses {
		for _, it := range c.Items {
			if it.Kind() == "crud" {
				res[cg.ClassNameWithoutModel(it.Dto)] = true
			}
		}
	}
	return res
}

func cleanRel(p string) string {
	return path.Clean(strings.ReplaceAll(p, `\`, "/"))
}
