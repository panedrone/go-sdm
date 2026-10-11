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

## database/sql, sqlx or gorm

`sdm init` prepares all three. A class is generated for one of them by its name in `sdm.xml`:

```xml
<dto-class name="Task" ref="tasks"/>       <!-- database/sql -->
<dto-class name="sqlx-Task" ref="tasks"/>  <!-- sqlx: db:"..." tags -->
<dto-class name="gorm-Task" ref="tasks"/>  <!-- gorm: gorm:"column:...;size:...;primaryKey;..." tags -->
```

The DAO folder gets a data store for each of them; sqlx and gorm are switched off by the
first line `//go:build ignore`. To use sqlx or gorm, move this line from
`data_store_sqlx.go` + `data_store_sqlx_ex.go` (or the gorm pair) to
`data_store_no_orm.go` + `data_store_no_orm_ex.go`. For sqlx, set `field-assign-mode="2"`
in `settings.xml`. Remove what you don't use.

## Type map

`<type detected="...">` in `settings.xml` says what the database reports about a column,
in Go words:

```
int16 int32 int64 uint64 float32 float64 decimal string []byte bool time.Time
uuid array cursor json xml interval money any (not detected)
```

with a prefix for sqlx and gorm classes (`sqlx-int32`, `gorm-time.Time`) and `+auto` for
the column of `auto="..."` (`gorm-uuid+auto`). `target` is the Go type of the field, its tags
and the function that reads it from a row:

```xml
<type detected="int32" target="int32${json} -> SetInt32"/>
```

The Java names of the IDE plug-ins (`java.lang.Integer`, ...) are understood as well, so their
`settings.xml` works as is; `go test` checks that both spellings produce the same code.

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
