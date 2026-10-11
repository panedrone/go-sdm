// sdm — генератор DAO для Go по sdm.xml и метаданным живой базы (PostgreSQL, SQLite).
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/panedrone/go-sdm/internal/gen"
	"github.com/panedrone/go-sdm/internal/sdmxml"
	"github.com/panedrone/go-sdm/internal/skeleton"
)

const version = "0.1.0-dev"

const usage = `sdm %s — SQL DAL Maker for Go

usage:
  sdm [-dir go-sdm] [-dao dbal]   no configuration yet: init; otherwise: generate
  sdm init [-dir go-sdm] [-dao dbal]
                                  create sdm.xml, settings.xml, XSD and the data store;
                                  an error if any of them exists, nothing is overwritten
  sdm generate [-dir go-sdm]      generate models and DAO into the folders of settings.xml
  sdm check [-dir go-sdm]         exit 1 if the generated code is out of date (for CI)
  sdm version

Run it in the folder of go.mod. The database is <jdbc url user pwd> of settings.xml;
SDM_PWD, if set, overrides "pwd".
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	cmd := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "":
		return runAuto(args, stdout, stderr)
	case "init":
		o, ok := parseInit("init", args, stderr)
		if !ok {
			return 2
		}
		return runInit(o, stdout, stderr)
	case "generate":
		return runGenerate(args, true, stdout, stderr)
	case "check":
		return runGenerate(args, false, stdout, stderr)
	case "version":
		fmt.Fprintln(stdout, version)
		return 0
	case "help":
		fmt.Fprintf(stdout, usage, version)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n"+usage, cmd, version)
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
	dao := fs.String("dao", "dbal", "folder of generated DAO, relative to go.mod; models go to <dao>/models")
	if err := fs.Parse(args); err != nil {
		return skeleton.Options{}, false
	}
	return skeleton.Options{ProjectDir: ".", SdmDir: cleanRel(*dir), DaoDir: cleanRel(*dao)}, true
}

// runAuto — без команды: init, если конфигурации ещё нет, иначе generate.
func runAuto(args []string, stdout, stderr io.Writer) int {
	o, ok := parseInit("sdm", args, stderr)
	if !ok {
		return 2
	}
	if !skeleton.Exists(o.ProjectDir, o.SdmDir) {
		return runInit(o, stdout, stderr)
	}
	return generate(o.SdmDir, true, stdout, stderr)
}

func runInit(o skeleton.Options, stdout, stderr io.Writer) int {
	written, err := skeleton.Init(o)
	for _, f := range written {
		fmt.Fprintln(stdout, "created: "+f)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "next: describe classes in %s/sdm.xml, set <jdbc url> in %s/settings.xml, run sdm again\n",
		o.SdmDir, o.SdmDir)
	return 0
}

// runGenerate — generate (write == true) или check.
func runGenerate(args []string, write bool, stdout, stderr io.Writer) int {
	fs, dir := newFlags(map[bool]string{true: "generate", false: "check"}[write], stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	return generate(cleanRel(*dir), write, stdout, stderr)
}

func generate(dir string, write bool, stdout, stderr io.Writer) int {
	s, err := sdmxml.LoadSettings(dir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if pwd, ok := os.LookupEnv("SDM_PWD"); ok {
		s.Jdbc.Pwd = pwd
	}
	md, err := gen.OpenMetadata(".", s.Jdbc)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer md.Close()
	res, err := gen.Run(".", dir, md)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if write {
		xsd, err := skeleton.RefreshXSD(".", dir)
		for _, f := range xsd {
			fmt.Fprintln(stdout, "updated: "+f)
		}
		if err != nil {
			fmt.Fprintln(stderr, err)
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
			fmt.Fprintln(stdout, "missing: "+c.Path)
		default:
			fmt.Fprintln(stdout, "out of date: "+c.Path)
		}
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if !write && len(changes) > 0 {
		return 1
	}
	return 0
}

func cleanRel(p string) string {
	return path.Clean(strings.ReplaceAll(p, `\`, "/"))
}
