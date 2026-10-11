// Package sqlite — meta.Metadata поверх SQLite (ncruces/go-sqlite3: чистый Go, без cgo).
//
// Повторяет то, что видит Java-плагин через xerial sqlite-jdbc (эталон — 3.53):
//   - колонки запроса до выполнения не имеют типа: java.lang.Object; precision/scale —
//     из скобок declared type, display — всегда 2147483647;
//   - настоящие типы приходят из getColumns: классификация declared type по подстрокам;
//   - getIndexInfo игнорирует флаг unique: unique == indexed;
//   - автоинкремент — только у колонок с AUTOINCREMENT.
//
// Сверено с MetaDump.java на internal/meta/testdata/sqlite.txt.
package sqlite

import (
	"errors"
	"strconv"
	"strings"

	"github.com/ncruces/go-sqlite3"

	"github.com/panedrone/go-sdm/internal/meta"
)

const (
	unknownDisplay = 2147483647
	unknownSize    = 2000000000 // COLUMN_SIZE без скобок у xerial
	typeInteger    = 4          // java.sql.Types
	typeFloat      = 6
	typeVarchar    = 12
	javaString     = "java.lang.String" // ParameterMetaData xerial: всегда String
)

type DB struct {
	conn *sqlite3.Conn
}

var _ meta.Metadata = (*DB)(nil)

// Open — путь к файлу или ":memory:".
func Open(filename string) (*DB, error) {
	conn, err := sqlite3.Open(filename)
	if err != nil {
		return nil, err
	}
	return newDB(conn)
}

// OpenReadOnly — файл базы проекта: генерация ничего не пишет и не создаёт пустую базу по опечатке в пути.
func OpenReadOnly(filename string) (*DB, error) {
	conn, err := sqlite3.OpenFlags(filename, sqlite3.OPEN_READONLY|sqlite3.OPEN_URI)
	if err != nil {
		return nil, err
	}
	return newDB(conn)
}

// newDB — строки в двойных кавычках как литералы (DQS), как у xerial: старые схемы и SQL
// на них рассчитывают, а современная сборка SQLite их по умолчанию запрещает.
func newDB(conn *sqlite3.Conn) (*DB, error) {
	for _, op := range []sqlite3.DBConfig{sqlite3.DBCONFIG_DQS_DML, sqlite3.DBCONFIG_DQS_DDL} {
		if _, err := conn.Config(op, true); err != nil {
			_ = conn.Close()
			return nil, err
		}
	}
	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// Exec — для подготовки схемы в тестах и дампах.
func (db *DB) Exec(sql string) error {
	return xerialError(db.conn.Exec(sql))
}

func (db *DB) prepare(sql string) (*sqlite3.Stmt, error) {
	stmt, _, err := db.conn.Prepare(sql)
	if err != nil {
		return nil, xerialError(err)
	}
	if stmt == nil {
		return nil, errors.New("empty SQL statement")
	}
	return stmt, nil
}

// SQL statement ---------------------------------------------------

func (db *DB) SQLColumns(jdbcSQL string) ([]meta.Column, error) {
	stmt, err := db.prepare(jdbcSQL)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	n := stmt.ColumnCount()
	if n == 0 {
		// дословно как в Java: xerial бросает это из getColumnCount()
		return nil, errors.New("Exception in getColumnCount(): column 1 out of bounds [1,0]")
	}
	res := make([]meta.Column, 0, n)
	for i := 0; i < n; i++ {
		name := stmt.ColumnName(i)
		if name == "" {
			name = "col_" + strconv.Itoa(i+1)
		}
		decl := stmt.ColumnDeclType(i)
		p, s, _ := sizeInParens(decl)
		ai := false
		if table := stmt.ColumnTableName(i); table != "" {
			_, _, _, _, ai, err = db.conn.TableColumnMetadata(stmt.ColumnDatabaseName(i), table, stmt.ColumnOriginName(i))
			if err != nil {
				return nil, xerialError(err)
			}
		}
		res = append(res, meta.Column{
			Name:        name,
			JavaType:    meta.JavaObject,
			RawType:     rawName(decl),
			IsAI:        ai,
			Scale:       s,
			Precision:   p,
			DisplaySize: unknownDisplay,
		})
	}
	return res, nil
}

func (db *DB) SQLParams(jdbcSQL string, typeCount int) (*meta.Params, error) {
	stmt, err := db.prepare(jdbcSQL)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	res := &meta.Params{Count: stmt.BindCount(), JavaTypes: make([]string, typeCount), RawTypes: make([]string, typeCount)}
	for i := range res.JavaTypes {
		res.JavaTypes[i] = javaString // и за пределами Count: xerial не бросает
	}
	return res, nil
}

// Table -----------------------------------------------------------

func (db *DB) Table(tableRef string) (*meta.Table, error) {
	schema, table, err := splitSchema(tableRef)
	if err != nil {
		return nil, err
	}
	found, err := db.exists(schema, table)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.New("Not found: '" + tableRef + "'. The search is case-sensitive.")
	}
	columns, err := db.SQLColumns("select * from " + tableRef)
	if err != nil {
		return nil, err
	}
	tableColumns, err := db.tableColumns(schema, table)
	if err != nil {
		return nil, err
	}
	return &meta.Table{Columns: columns, TableColumns: tableColumns}, nil
}

func splitSchema(tableRef string) (schema, table string, err error) {
	if !strings.Contains(tableRef, ".") {
		return "", tableRef, nil
	}
	parts := strings.Split(tableRef, ".")
	if len(parts) != 2 {
		return "", "", errors.New("Invalid table name: '" + tableRef + "'")
	}
	return parts[0], parts[1], nil
}

func master(schema string) string {
	if schema == "" {
		return "sqlite_master"
	}
	return `"` + strings.ReplaceAll(schema, `"`, `""`) + `".sqlite_master`
}

// exists — getTables(TABLE, VIEW): LIKE, как у xerial.
func (db *DB) exists(schema, table string) (bool, error) {
	found := false
	err := db.query(`select 1 from `+master(schema)+` where type in ('table', 'view') and name like ?`,
		[]string{table}, func(*sqlite3.Stmt) { found = true })
	return found, err
}

func (db *DB) tableColumns(schema, table string) ([]meta.TableColumn, error) {
	indexed, err := db.indexedColumns(schema, table)
	if err != nil {
		return nil, err
	}
	fk, err := db.importedKeys(schema, table)
	if err != nil {
		return nil, err
	}
	var res []meta.TableColumn
	err = db.query(`select name, type, "notnull" from pragma_table_info(?, nullif(?, '')) order by cid`,
		[]string{table, schema}, func(s *sqlite3.Stmt) {
			name, declType, notNull := s.ColumnText(0), s.ColumnText(1), s.ColumnInt(2) != 0
			// у колонок вьюхи метаданных нет (no such table column), и автоинкремента тоже
			_, _, _, _, ai, err := db.conn.TableColumnMetadata(schema, table, name)
			if err != nil {
				ai = false
			}
			dataType, size, digits := columnType(declType)
			nullable := !notNull
			lo := strings.ToLower(name)
			_, isIndexed := indexed[lo]
			res = append(res, meta.TableColumn{
				Name:          name,
				DataType:      &dataType,
				RawType:       rawName(declType),
				ColumnSize:    &size,
				DecimalDigits: &digits,
				IsAI:          &ai,
				IsNullable:    &nullable,
				FK:            fk[lo],
				IsIndexed:     isIndexed,
				IsUnique:      isIndexed, // xerial getIndexInfo не фильтрует по unique
			})
		})
	return res, err
}

func (db *DB) indexedColumns(schema, table string) (map[string]struct{}, error) {
	res := map[string]struct{}{}
	err := db.query(`select ii.name from pragma_index_list(?, nullif(?, '')) il
		join pragma_index_info(il.name, nullif(?, '')) ii where ii.name is not null`,
		[]string{table, schema, schema}, func(s *sqlite3.Stmt) {
			res[strings.ToLower(s.ColumnText(0))] = struct{}{}
		})
	return res, err
}

func (db *DB) importedKeys(schema, table string) (map[string]string, error) {
	res := map[string]string{}
	err := db.query(`select "from", "table", coalesce("to", '') from pragma_foreign_key_list(?, nullif(?, ''))`,
		[]string{table, schema}, func(s *sqlite3.Stmt) {
			res[strings.ToLower(s.ColumnText(0))] = s.ColumnText(1) + "." + s.ColumnText(2)
		})
	return res, err
}

func (db *DB) PKColNames(tableRef string) ([]string, error) {
	schema, table, err := splitSchema(tableRef)
	if err != nil {
		return nil, err
	}
	res := []string{}
	err = db.query(`select name from pragma_table_info(?, nullif(?, '')) where pk > 0 order by pk`,
		[]string{table, schema}, func(s *sqlite3.Stmt) { res = append(res, s.ColumnText(0)) })
	return res, err
}

func (db *DB) query(sql string, args []string, onRow func(*sqlite3.Stmt)) error {
	stmt, err := db.prepare(sql)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i, a := range args {
		if err := stmt.BindText(i+1, a); err != nil {
			return xerialError(err)
		}
	}
	for stmt.Step() {
		onRow(stmt)
	}
	return xerialError(stmt.Err())
}

// Типы ------------------------------------------------------------

// columnType — DATA_TYPE, COLUMN_SIZE, DECIMAL_DIGITS из declared type, как getColumns у xerial:
// подстроки проверяются в этом порядке, размер — из скобок (COLUMN_SIZE = p + s).
func columnType(declType string) (dataType, size, digits int) {
	t := strings.ToUpper(declType)
	switch {
	case strings.Contains(t, "INT") || strings.Contains(t, "BOOL"):
		dataType, digits = typeInteger, 0
	case strings.Contains(t, "CHAR") || strings.Contains(t, "CLOB") || strings.Contains(t, "TEXT") || strings.Contains(t, "BLOB"):
		dataType, digits = typeVarchar, 0
	case strings.Contains(t, "REAL") || strings.Contains(t, "FLOA") || strings.Contains(t, "DOUB") ||
		strings.Contains(t, "DEC") || strings.Contains(t, "NUM"):
		dataType, digits = typeFloat, 10
	default:
		dataType, digits = typeVarchar, 10
	}
	size = unknownSize
	if p, s, ok := sizeInParens(declType); ok {
		size, digits = p+s, s
	}
	return
}

// sizeInParens — "(p)" или "(p, s)" в declared type; пробелы допускаются.
func sizeInParens(declType string) (p, s int, ok bool) {
	open := strings.IndexByte(declType, '(')
	if open < 0 {
		return 0, 0, false
	}
	end := strings.IndexByte(declType[open:], ')')
	if end < 0 {
		return 0, 0, false
	}
	parts := strings.Split(declType[open+1:open+end], ",")
	p, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, false
	}
	if len(parts) > 1 {
		if s, err = strconv.Atoi(strings.TrimSpace(parts[1])); err != nil {
			return 0, 0, false
		}
	}
	return p, s, true
}

// xerialError — SQLITE_ERROR в формате xerial: "[SQLITE_ERROR] SQL error or missing database (msg)".
func xerialError(err error) error {
	var e *sqlite3.Error
	if err == nil || !errors.As(err, &e) || e.Code() != sqlite3.ERROR {
		return err
	}
	msg := strings.TrimPrefix(e.Error(), sqlite3.ERROR.Error()+": ")
	return errors.New("[SQLITE_ERROR] SQL error or missing database (" + msg + ")")
}

// rawName — объявленный тип словами драйвера: "varchar(50)" -> "VARCHAR"; "" — не объявлен.
func rawName(declType string) string {
	if i := strings.IndexByte(declType, '('); i >= 0 {
		declType = declType[:i]
	}
	return strings.ToUpper(strings.TrimSpace(declType))
}
