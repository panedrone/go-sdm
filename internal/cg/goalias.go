package cg

import (
	"sort"
	"strings"
)

// Go-имена типов для <type detected="...">: то, что о колонке сообщают метаданные,
// словами Go-мира. Java-имена (как у IDE-плагина) понимаются по-прежнему.
var javaToGo = map[string]string{
	"java.lang.Short":                "int16",
	"java.lang.Integer":              "int32",
	"java.lang.Long":                 "int64",
	"java.math.BigInteger":           "uint64", // MySQL bigint unsigned
	"java.lang.Float":                "float32",
	"java.lang.Double":               "float64",
	"java.math.BigDecimal":           "decimal",
	"java.lang.String":               "string",
	"byte[]":                         "[]byte",
	"java.lang.Boolean":              "bool",
	"java.util.Date":                 "time.Time",
	"java.time.LocalDateTime":        "time.Time",
	"java.time.ZonedDateTime":        "time.Time",
	"java.util.UUID":                 "uuid",
	"java.sql.Array":                 "array",
	"java.sql.ResultSet":             "cursor", // PostgreSQL refcursor
	"java.lang.Object":               "any",    // тип не определён
	"java.sql.SQLXML":                "xml",
	"org.postgresql.util.PGobject":   "json",
	"org.postgresql.util.PGInterval": "interval",
	"org.postgresql.util.PGmoney":    "money",
}

var goNames = func() map[string]bool {
	res := map[string]bool{}
	for _, g := range javaToGo {
		res[g] = true
	}
	return res
}()

// GoTypeNames — словарь Go-имён (для документации и проверок).
func GoTypeNames() []string {
	res := make([]string, 0, len(goNames))
	for g := range goNames {
		res = append(res, g)
	}
	sort.Strings(res)
	return res
}

// splitDetected — "gorm-java.util.UUID+auto" -> "gorm-", "java.util.UUID", "+auto".
func splitDetected(d string) (model, core, suffix string) {
	core = d
	if i := strings.IndexByte(core, '-'); i >= 0 {
		model, core = core[:i+1], core[i+1:]
	}
	if i := strings.IndexByte(core, '+'); i >= 0 {
		core, suffix = core[:i], core[i:]
	}
	return model, core, suffix
}

// GoAlias — Go-имя для типа из метаданных: "gorm-java.lang.Integer" -> "gorm-int32".
func GoAlias(detected string) (string, bool) { return goAlias(detected) }

func goAlias(detected string) (string, bool) {
	model, core, suffix := splitDetected(detected)
	g, ok := javaToGo[core]
	if !ok {
		return "", false
	}
	return model + g + suffix, true
}

// isAliasKey — ключ type-map записан Go-именем.
func isAliasKey(key string) bool {
	_, core, _ := splitDetected(key)
	return goNames[core]
}
