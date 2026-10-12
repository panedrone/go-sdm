package pg

import "strings"

// javaRaw — Java-имена плагина словами pg_type.
var javaRaw = map[string]string{
	"java.lang.Short":         "int2",
	"java.lang.Integer":       "int4",
	"java.lang.Long":          "int8",
	"java.lang.Float":         "float4",
	"java.lang.Double":        "float8",
	"java.math.BigDecimal":    "numeric",
	"java.math.BigInteger":    "numeric",
	"java.lang.String":        "varchar",
	"java.lang.Boolean":       "bool",
	"java.util.Date":          "timestamp",
	"java.sql.Timestamp":      "timestamp",
	"java.time.LocalDateTime": "timestamp",
	"java.sql.Date":           "date",
	"java.time.LocalDate":     "date",
	"java.sql.Time":           "time",
	"byte[]":                  "bytea",
	"java.util.UUID":          "uuid",
	"java.sql.Array":          "_text",
}

func (db *DB) RawOfJava(javaType string) string { return javaRaw[strings.TrimSpace(javaType)] }
