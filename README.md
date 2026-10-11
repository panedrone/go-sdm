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
go-sdm/sdm.xml                         classes to generate
go-sdm/settings.xml                    folders, database URL, type map
go-sdm/*.xsd                           completion and validation in the IDE
internal/dbal/data_store_no_orm.go     the data store used by the generated code (database/sql + pgx)
internal/dbal/data_store_no_orm_ex.go  your part: connection
```

The generated code goes under `internal/`: the DAO and the models are the data layer of your
application, not an API for other modules, and the Go compiler keeps them so. Deeper folders work
the same way, for example `internal/noorm/dbal` (`sdm init -dao internal/noorm/dbal` or
`scope` in `settings.xml`).

Describe classes in `go-sdm/sdm.xml`, set `<jdbc url>` in `go-sdm/settings.xml` and run `sdm`
again: from now on it generates models and DAO. It also keeps the XSD in step with the
version of `sdm`.

| command | |
|---|---|
| `sdm` | no configuration yet: `init`, otherwise `generate` |
| `sdm init` | creates the files above; if any of them exists, it is an error and nothing is written |
| `sdm generate` | generates into the folders of `settings.xml`, writes only what changed |
| `sdm check` | for CI: exit 1 if somebody changed the XML or the database and forgot to regenerate |

Options: `-dir <folder>` instead of `go-sdm`; for `init`, `-dao <folder>` instead of `internal/dbal`
and `-orm sqlx` or `-orm gorm` instead of database/sql.
`SDM_PWD`, if set, overrides `pwd` of `settings.xml`.

## database/sql, sqlx or gorm

`sdm init` prepares the type map for all three. A class is generated for one of them by its
name in `sdm.xml`:

```xml
<dto-class name="Task" ref="tasks"/>       <!-- database/sql -->
<dto-class name="sqlx-Task" ref="tasks"/>  <!-- sqlx: db:"..." tags -->
<dto-class name="gorm-Task" ref="tasks"/>  <!-- gorm: gorm:"column:...;size:...;primaryKey;..." tags -->
```

The generated DAO need a data store in their folder: two files, the code and your part
(`_ex`, the connection). `sdm init -orm no_orm|sqlx|gorm` writes the pair you choose
(`no_orm` by default; for `sqlx` also `field-assign-mode="2"`). By hand, take them from
[`datastore/`](datastore):

| | |
|---|---|
| database/sql | [data_store_no_orm.go](datastore/no_orm/data_store_no_orm.go), [data_store_no_orm_ex.go](datastore/no_orm/data_store_no_orm_ex.go) |
| sqlx | [data_store_sqlx.go](datastore/sqlx/data_store_sqlx.go), [data_store_sqlx_ex.go](datastore/sqlx/data_store_sqlx_ex.go) |
| gorm | [data_store_gorm.go](datastore/gorm/data_store_gorm.go), [data_store_gorm_ex.go](datastore/gorm/data_store_gorm_ex.go) |

remove the first line `//go:build ignore` and set the package name of your DAO folder.

## Type map

`<type detected="...">` in `settings.xml` is the type of a column as the driver names it:

| Database   | `detected`                                                                       |
|------------|----------------------------------------------------------------------------------|
| PostgreSQL | `int4`, `int8`, `numeric`, `varchar`, `text`, `timestamptz`, `uuid`, `_text` (arrays), `refcursor`, ... |
| MySQL      | `INT`, `UNSIGNED BIGINT`, `DECIMAL`, `VARCHAR`, `TEXT`, `DATETIME`, `JSON`, ...  |
| SQLite     | what the column is declared with: `INTEGER`, `VARCHAR`, `REAL`, ...              |

with a prefix for sqlx and gorm classes (`sqlx-int4`, `gorm-timestamptz`) and `+auto` for
the column of `auto="..."` (`gorm-uuid+auto`). `target` is the Go type of the field, its tags
and the function that reads it from a row:

```xml
<type detected="int4" target="int32${json} -> SetInt32"/>
```

`sdm init` writes the keys of PostgreSQL. To see which names your database gives, empty the
type map (`<type-map default=""/>`) and generate: the code then has the raw names of the
driver instead of Go types. Where the driver says nothing (SQLite
expressions), the key is `java.lang.Object`. A type written in `sdm.xml` (`<field type="int4">`,
`return-type="int8"`) goes through the same type map.

The `settings.xml` of the IDE plug-ins (Java names: `java.lang.Integer`, ...) works as is.

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
datastore/        the data stores for database/sql, sqlx and gorm (also for download)
cli/              design-time tools, not a part of the build
```
