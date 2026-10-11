// Package meta — что база говорит о SQL-запросах и таблицах, как есть: без settings,
// без type-map, без полей DTO. Повторяет IDbMetadata Java-плагина один в один, чтобы
// вывод генератора можно было сверять побайтово (cli/metadump против MetaDump.java).
//
// Типы называются Java-именами (java.lang.Integer, java.util.Date, ...): на них
// завязаны ключи <type detected=...> в settings.xml существующих проектов.
//
// RawType — то же словами драйвера: второй словарь ключей type-map, наравне с Java-именами.
package meta

// Metadata — одна реализация на диалект (pg, позже mysql и sqlite).
type Metadata interface {
	// SQLColumns — колонки результата jdbcSQL (SQL с '?').
	SQLColumns(jdbcSQL string) ([]Column, error)

	// Table — 'select * from tableRef' + то, что каталог говорит о колонках.
	Table(tableRef string) (*Table, error)

	// PKColNames — имена PK-колонок в порядке ключа.
	PKColNames(tableRef string) ([]string, error)

	// SQLParams — параметры jdbcSQL; типы читаются для индексов 0..typeCount-1.
	SQLParams(jdbcSQL string, typeCount int) (*Params, error)

	Close() error
}

const (
	JavaObject = "java.lang.Object"
	// RawUnknown — сырое имя типа, о котором драйвер ничего не сказал (ключ type-map)
	RawUnknown = "unknown"
)

// Column — колонка результата (ResultSetMetaData).
type Column struct {
	Name        string
	JavaType    string // JavaObject, если не определён
	RawType     string // тип словами драйвера (pg: int4, MySQL: INT, SQLite: INTEGER); "" — не сообщил
	IsAI        bool
	Scale       int
	Precision   int
	DisplaySize int
}

// TableColumn — строка DatabaseMetaData.getColumns. nil — драйвер не смог сообщить.
type TableColumn struct {
	Name          string
	DataType      *int   // java.sql.Types
	RawType       string // тип словами драйвера; нужен там, где колонка результата его не знает (SQLite)
	ColumnSize    *int
	DecimalDigits *int
	IsAI          *bool
	IsNullable    *bool
	FK            string // "pk_table.pk_column" или ""
	IsIndexed     bool
	IsUnique      bool
}

type Table struct {
	Columns []Column // 'select * from table'
	// в порядке getColumns; одна и та же колонка может прийти несколько раз (одноимённые
	// таблицы в разных схемах), и тогда каждая строка уточняет её ещё раз
	TableColumns []TableColumn
}

type Params struct {
	CountUnknown bool // метаданные параметров есть, но их число узнать не удалось
	Count        int
	JavaTypes    []string // JavaObject там, где не определён
	RawTypes     []string // типы словами драйвера; "" — не сообщил (длина — как у JavaTypes)
}
