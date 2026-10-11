# go-sdm

SQL DAL Maker for Go: generates DAO functions and models from a short XML description
and the metadata of a **live database**. One static binary, no runtime dependencies.

SQLite is built in (pure Go, no cgo), so `go test` runs the whole cycle on an in-memory
database: no PostgreSQL, no external tools.

```xml
<dto-class name="Test" ref="tests"/>

<dao-class name="TestsDao">
    <crud dto="Test" table="tests"/>
    <query-dto-list method="GetTestsByRating(t_rating)" dto="Test" ref="select * from get_tests_by_rating(?)"/>
    <exec-dml method="SetTestRating(t_id, t_rating)" ref="call set_test_rating(?, ?)"/>
</dao-class>
```

Why not just parse the DDL: the database knows what DDL doesn't show — views, functions,
procedures with `INOUT` parameters, ref-cursors, the real types of query columns.

## Status

Early development. PostgreSQL, MySQL 8 and SQLite.

| | |
|---|---|
| `sdm` (init or generate) | ready |
| database metadata: PostgreSQL, MySQL, SQLite | ready, verified line by line against the SQL DAL Maker IDE plug-ins |
| generation | ready, output byte-identical to the IDE plug-ins |
| `sdm check` (for CI) | ready: exit 1 if the generated code is out of date |

Verified against the SQL DAL Maker IDE plug-ins on real projects: metadata line by line,
generated code byte by byte (PostgreSQL, MySQL; SQLite in `go test`).

`sdm.xml` and `settings.xml` are the same as in the
[SQL DAL Maker IDE plug-ins](https://sqldalmaker.sourceforge.net/). go-sdm keeps them in
its own folder `go-sdm/`, so it can live in a project next to the `sdm/` folder of a plug-in.

## Quick start

```bash
go install github.com/panedrone/go-sdm/cmd/sdm@latest
```

In the folder of `go.mod`, just

```bash
sdm
```

The first run finds no configuration and creates it:

```
go-sdm/sdm.xml                classes to generate
go-sdm/settings.xml           folders, database URL, type map
go-sdm/*.xsd                  completion and validation in the IDE
dbal/data_store_no_orm.go     the data store used by the generated code (database/sql + pgx)
dbal/data_store_no_orm_ex.go  your part: connection
```

Describe classes in `go-sdm/sdm.xml`, set `<jdbc url>` in `go-sdm/settings.xml` and run `sdm`
again: from now on it generates models and DAO. It also keeps the XSD in step with the
version of `sdm`.

| command | |
|---|---|
| `sdm` | no configuration yet: `init`, otherwise `generate` |
| `sdm init` | creates the files above; if any of them exists, it is an error and nothing is written |
| `sdm generate` | generates into the folders of `settings.xml`, writes only what changed |
| `sdm check` | for CI: exit 1 if somebody changed the XML or the database and forgot to regenerate |

Options: `-dir <folder>` instead of `go-sdm`, `-dao <folder>` (for `init`) instead of `dbal`.
`SDM_PWD`, if set, overrides `pwd` of `settings.xml`.

The XML files are meant to be edited in an IDE with XML Schema support:
GoLand out of the box, VS Code with the "XML" extension by Red Hat.

## Layout

```
cmd/sdm/          the CLI
internal/meta/    database metadata: one implementation per dialect (pg, mysql, sqlite)
internal/sdmxml/  sdm.xml, DAO XML files, settings.xml
internal/cg/      the core: fields, parameters, type map, macros, ref and method grammar
internal/gogen/   Go output: models, DAO, go.tmpl
internal/gen/     one generation run; go test runs it on in-memory SQLite against golden files
internal/skeleton the files of 'sdm init'
cli/              design-time tools, not a part of the build
```
