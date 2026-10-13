package sqlite

import "strings"

// javaRaw maps the plug-in's Java names to SQLite declared types.
var javaRaw = map[string]string{
	"java.lang.Short":         "INTEGER",
	"java.lang.Integer":       "INTEGER",
	"java.lang.Long":          "INTEGER",
	"java.lang.Float":         "REAL",
	"java.lang.Double":        "REAL",
	"java.math.BigDecimal":    "NUMERIC",
	"java.lang.String":        "TEXT",
	"java.lang.Boolean":       "BOOLEAN",
	"java.util.Date":          "DATETIME",
	"java.sql.Timestamp":      "DATETIME",
	"java.time.LocalDateTime": "DATETIME",
	"java.sql.Date":           "DATE",
	"java.time.LocalDate":     "DATE",
	"byte[]":                  "BLOB",
}

func (db *DB) RawOfJava(javaType string) string { return javaRaw[strings.TrimSpace(javaType)] }
