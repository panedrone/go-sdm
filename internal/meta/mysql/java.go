package mysql

import "strings"

// javaRaw maps the plug-in's Java names to the words of the MySQL driver (as rawName).
var javaRaw = map[string]string{
	"java.lang.Byte":          "TINYINT",
	"java.lang.Short":         "SMALLINT",
	"java.lang.Integer":       "INT",
	"java.lang.Long":          "BIGINT",
	"java.math.BigInteger":    "UNSIGNED BIGINT",
	"java.lang.Float":         "FLOAT",
	"java.lang.Double":        "DOUBLE",
	"java.math.BigDecimal":    "DECIMAL",
	"java.lang.String":        "VARCHAR",
	"java.lang.Boolean":       "BIT",
	"java.util.Date":          "DATETIME",
	"java.sql.Timestamp":      "DATETIME",
	"java.time.LocalDateTime": "DATETIME",
	"java.sql.Date":           "DATE",
	"java.time.LocalDate":     "DATE",
	"java.sql.Time":           "TIME",
	"byte[]":                  "BLOB",
}

func (db *DB) RawOfJava(javaType string) string { return javaRaw[strings.TrimSpace(javaType)] }
