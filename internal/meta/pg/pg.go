// Package pg — meta.Metadata поверх pgx. Запросы не выполняются: колонки и параметры
// берутся из Prepare (описание statement-а), остальное — из pg_catalog по тем же правилам,
// что у pgjdbc в DatabaseMetaData, чтобы совпадать с Java-плагином.
package pg

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/panedrone/go-sdm/internal/meta"
)

type pgType struct {
	name    string // typname: int4, varchar, _int4, ...
	isArray bool
	elem    uint32
}

type DB struct {
	ctx   context.Context
	conn  *pgx.Conn
	types map[uint32]pgType // pg_type по OID
}

var _ meta.Metadata = (*DB)(nil)

func Open(ctx context.Context, connString string) (*DB, error) {
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return nil, err
	}
	return &DB{ctx: ctx, conn: conn, types: map[uint32]pgType{}}, nil
}

func (db *DB) Close() error {
	return db.conn.Close(db.ctx)
}

// SQL statement ---------------------------------------------------

func (db *DB) SQLColumns(jdbcSQL string) ([]meta.Column, error) {
	native, _, err := toNative(jdbcSQL)
	if err != nil {
		return nil, err
	}
	sd, err := db.conn.PgConn().Prepare(db.ctx, "", native, nil)
	if err != nil {
		return nil, err
	}
	if len(sd.Fields) == 0 {
		// так это выглядит в Java: ResultSetMetaData == null у statement-а без строк
		return nil, errors.New("Cannot detect Prepared Statement MetaData")
	}
	res := make([]meta.Column, 0, len(sd.Fields))
	for i, f := range sd.Fields {
		name := f.Name
		if name == "" {
			name = "col_" + strconv.Itoa(i+1)
		}
		javaType, oid, err := db.resolve(f.DataTypeOID)
		if err != nil {
			return nil, err
		}
		raw, err := db.pgType(f.DataTypeOID)
		if err != nil {
			return nil, err
		}
		ai := false
		if f.TableOID != 0 {
			ai, err = db.isAutoIncrement(f.TableOID, f.TableAttributeNumber)
			if err != nil {
				return nil, err
			}
		}
		res = append(res, meta.Column{
			Name:        name,
			JavaType:    javaType,
			RawType:     raw.name,
			IsAI:        ai,
			Scale:       scale(oid, f.TypeModifier),
			Precision:   precision(oid, f.TypeModifier),
			DisplaySize: displaySize(oid, f.TypeModifier),
		})
	}
	return res, nil
}

func (db *DB) SQLParams(jdbcSQL string, typeCount int) (*meta.Params, error) {
	native, _, err := toNative(jdbcSQL)
	if err != nil {
		return nil, err
	}
	sd, err := db.conn.PgConn().Prepare(db.ctx, "", native, nil)
	if err != nil {
		return nil, err
	}
	res := &meta.Params{Count: len(sd.ParamOIDs), JavaTypes: make([]string, 0, typeCount), RawTypes: make([]string, 0, typeCount)}
	for i := 0; i < typeCount; i++ {
		if i >= len(sd.ParamOIDs) {
			res.JavaTypes = append(res.JavaTypes, meta.JavaObject)
			res.RawTypes = append(res.RawTypes, "")
			continue
		}
		javaType, _, err := db.resolve(sd.ParamOIDs[i])
		if err != nil {
			return nil, err
		}
		raw, err := db.pgType(sd.ParamOIDs[i])
		if err != nil {
			return nil, err
		}
		res.JavaTypes = append(res.JavaTypes, javaType)
		res.RawTypes = append(res.RawTypes, raw.name)
	}
	return res, nil
}

// resolve — Java-тип и OID, по которому считаются precision/scale/display
// (у массива — OID элемента).
func (db *DB) resolve(oid uint32) (string, uint32, error) {
	if t, ok := javaTypes[oid]; ok {
		return t, oid, nil
	}
	t, err := db.pgType(oid)
	if err != nil {
		return "", 0, err
	}
	if t.isArray {
		return javaArray, t.elem, nil
	}
	return javaObject, oid, nil
}

func (db *DB) pgType(oid uint32) (pgType, error) {
	if t, ok := db.types[oid]; ok {
		return t, nil
	}
	var t pgType
	err := db.conn.QueryRow(db.ctx,
		`select t.typname, t.typcategory = 'A', t.typelem from pg_catalog.pg_type t where t.oid = $1`, oid).
		Scan(&t.name, &t.isArray, &t.elem)
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}
	if err != nil {
		return t, err
	}
	db.types[oid] = t
	return t, nil
}

const aiExpr = `(a.attidentity in ('a', 'd') or coalesce(pg_catalog.pg_get_expr(d.adbin, d.adrelid) like '%nextval(%', false))`

func (db *DB) isAutoIncrement(tableOID uint32, attnum uint16) (bool, error) {
	var ai bool
	err := db.conn.QueryRow(db.ctx, `select `+aiExpr+`
		from pg_catalog.pg_attribute a
		left join pg_catalog.pg_attrdef d on d.adrelid = a.attrelid and d.adnum = a.attnum
		where a.attrelid = $1 and a.attnum = $2`, tableOID, int16(attnum)).Scan(&ai)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return ai, err
}

// Table -----------------------------------------------------------

func (db *DB) Table(tableRef string) (*meta.Table, error) {
	if err := db.validateTableName(tableRef); err != nil {
		return nil, err
	}
	columns, err := db.SQLColumns("select * from " + tableRef)
	if err != nil {
		return nil, err
	}
	tableColumns, err := db.tableColumns(tableRef)
	if err != nil {
		return nil, err
	}
	return &meta.Table{Columns: columns, TableColumns: tableColumns}, nil
}

// splitSchema — "schema.table" ровно из двух частей, как getTables/getPrimaryKeys в Java.
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

func (db *DB) validateTableName(tableRef string) error {
	schema, table, err := splitSchema(tableRef)
	if err != nil {
		return err
	}
	// getTables(TABLE, VIEW): шаблоны LIKE, системные схемы — не TABLE
	var found bool
	err = db.conn.QueryRow(db.ctx, `select exists(
		select 1 from pg_catalog.pg_class c
		join pg_catalog.pg_namespace n on n.oid = c.relnamespace
		where c.relname like $1 and ($2 = '' or n.nspname like $2)
		  and (c.relkind = 'v' or c.relkind in ('r', 'p')
		       and n.nspname not like 'pg\_%' and n.nspname <> 'information_schema'))`,
		table, schema).Scan(&found)
	if err != nil {
		return err
	}
	if !found {
		return errors.New("Not found: '" + tableRef + "'. The search is case-sensitive.")
	}
	return nil
}

func (db *DB) tableColumns(tableRef string) ([]meta.TableColumn, error) {
	// как _read_table_columns в Java: схема — всё до последней точки
	schema, table := "", tableRef
	if i := strings.LastIndexByte(tableRef, '.'); i >= 0 {
		schema, table = tableRef[:i], tableRef[i+1:]
	}
	// Java передаёт в getIndexInfo/getImportedKeys полное имя (со схемой), и для
	// "schema.table" индексы и FK не находятся. Повторено ради побайтового совпадения.
	unique, err := db.indexedColumns(schema, tableRef, true)
	if err != nil {
		return nil, err
	}
	indexed, err := db.indexedColumns(schema, tableRef, false)
	if err != nil {
		return nil, err
	}
	fk, err := db.importedKeys(schema, tableRef)
	if err != nil {
		return nil, err
	}
	rows, err := db.conn.Query(db.ctx, `select a.attname,
			case when t.typtype = 'd' then t.typbasetype else a.atttypid end,
			case when t.typtype = 'd' then t.typtypmod else a.atttypmod end,
			`+aiExpr+`,
			not (a.attnotnull or (t.typtype = 'd' and t.typnotnull))
		from pg_catalog.pg_attribute a
		join pg_catalog.pg_class c on c.oid = a.attrelid
		join pg_catalog.pg_namespace n on n.oid = c.relnamespace
		join pg_catalog.pg_type t on t.oid = a.atttypid
		left join pg_catalog.pg_attrdef d on d.adrelid = a.attrelid and d.adnum = a.attnum
		where c.relkind in ('r', 'p', 'v', 'f', 'm') and a.attnum > 0 and not a.attisdropped
		  and c.relname like $1 and ($2 = '' or n.nspname like $2)
		order by n.nspname, c.relname, a.attnum`, table, schema)
	if err != nil {
		return nil, err
	}
	type row struct {
		name     string
		oid      uint32
		typmod   int32
		ai       bool
		nullable bool
	}
	scanned, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (row, error) {
		var x row
		err := r.Scan(&x.name, &x.oid, &x.typmod, &x.ai, &x.nullable)
		return x, err
	})
	if err != nil {
		return nil, err
	}
	res := make([]meta.TableColumn, 0, len(scanned))
	for _, x := range scanned {
		dataType := sqlType(x.oid)
		sizeOID := x.oid
		if _, known := javaTypes[x.oid]; !known {
			t, err := db.pgType(x.oid)
			if err != nil {
				return nil, err
			}
			if t.isArray {
				dataType = sqlTypeArray
				sizeOID = t.elem
			}
		}
		size := precision(sizeOID, x.typmod)
		digits := scale(sizeOID, x.typmod)
		lo := strings.ToLower(x.name)
		_, isIndexed := indexed[lo]
		_, isUnique := unique[lo]
		res = append(res, meta.TableColumn{
			Name:          x.name,
			DataType:      &dataType,
			ColumnSize:    &size,
			DecimalDigits: &digits,
			IsAI:          &x.ai,
			IsNullable:    &x.nullable,
			FK:            fk[lo],
			IsIndexed:     isIndexed,
			IsUnique:      isUnique,
		})
	}
	return res, nil
}

// indexedColumns — getIndexInfo: имена колонок в нижнем регистре; table сравнивается точно.
func (db *DB) indexedColumns(schema, table string, uniqueOnly bool) (map[string]struct{}, error) {
	rows, err := db.conn.Query(db.ctx, `select a.attname
		from pg_catalog.pg_index i
		join pg_catalog.pg_class ct on ct.oid = i.indrelid
		join pg_catalog.pg_namespace n on n.oid = ct.relnamespace
		join pg_catalog.pg_attribute a on a.attrelid = ct.oid and a.attnum = any(i.indkey)
		where ct.relname = $1 and ($2 = '' or n.nspname = $2) and (not $3 or i.indisunique)`,
		table, schema, uniqueOnly)
	if err != nil {
		return nil, err
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	res := make(map[string]struct{}, len(names))
	for _, s := range names {
		res[strings.ToLower(s)] = struct{}{}
	}
	return res, nil
}

// importedKeys — getImportedKeys: FK-колонка в нижнем регистре -> "pk_table.pk_column".
func (db *DB) importedKeys(schema, table string) (map[string]string, error) {
	rows, err := db.conn.Query(db.ctx, `select fa.attname, pc.relname, pa.attname
		from pg_catalog.pg_constraint con
		join pg_catalog.pg_class fc on fc.oid = con.conrelid
		join pg_catalog.pg_namespace n on n.oid = fc.relnamespace
		join pg_catalog.pg_class pc on pc.oid = con.confrelid
		cross join lateral unnest(con.conkey, con.confkey) as k(fk_attnum, pk_attnum)
		join pg_catalog.pg_attribute fa on fa.attrelid = con.conrelid and fa.attnum = k.fk_attnum
		join pg_catalog.pg_attribute pa on pa.attrelid = con.confrelid and pa.attnum = k.pk_attnum
		where con.contype = 'f' and fc.relname = $1 and ($2 = '' or n.nspname = $2)
		order by pc.relname, con.conname, k.fk_attnum`, table, schema)
	if err != nil {
		return nil, err
	}
	res := map[string]string{}
	var fkCol, pkTable, pkCol string
	_, err = pgx.ForEachRow(rows, []any{&fkCol, &pkTable, &pkCol}, func() error {
		res[strings.ToLower(fkCol)] = pkTable + "." + pkCol
		return nil
	})
	return res, err
}

func (db *DB) PKColNames(tableRef string) ([]string, error) {
	schema, table, err := splitSchema(tableRef)
	if err != nil {
		return nil, err
	}
	rows, err := db.conn.Query(db.ctx, `select a.attname
		from pg_catalog.pg_index i
		join pg_catalog.pg_class ct on ct.oid = i.indrelid
		join pg_catalog.pg_namespace n on n.oid = ct.relnamespace
		cross join lateral unnest(i.indkey::int2[]) with ordinality as k(attnum, seq)
		join pg_catalog.pg_attribute a on a.attrelid = ct.oid and a.attnum = k.attnum
		where i.indisprimary and ct.relname = $1 and ($2 = '' or n.nspname = $2)
		order by n.nspname, ct.relname, k.seq`, table, schema)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}
