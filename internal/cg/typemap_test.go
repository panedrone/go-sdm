package cg

import (
	"testing"

	"github.com/panedrone/go-sdm/internal/sdmxml"
)

func TestTypeMapGoNames(t *testing.T) {
	tm := NewTypeMap(sdmxml.TypeMap{Types: []sdmxml.Type{
		{Detected: "int32", Target: "int32${json} -> SetInt32"},
		{Detected: "gorm-int32", Target: "int32${json-gorm} -> SetInt32"},
		{Detected: "gorm-uuid+auto", Target: "uuid.UUID|1:;default:gen_random_uuid()"},
		{Detected: "cursor", Target: "string"},
		{Detected: "string", Target: "string${json} -> SetString"},
		{Detected: "java.lang.Long", Target: "int64-from-java"}, // Java-ключ главнее перевода
		{Detected: "int64", Target: "int64-from-go"},
	}})
	cases := map[string]string{
		"java.lang.Integer":        "int32${json} -> SetInt32",
		"gorm-java.lang.Integer":   "int32${json-gorm} -> SetInt32",
		"gorm-java.util.UUID+auto": "uuid.UUID|1:;default:gen_random_uuid()",
		"java.sql.ResultSet":       "string",
		"java.lang.Long":           "int64-from-java",
		// Go-ключи — только через перевод: повторный проход ядра с целью "string"
		// не должен её перемапить, как не перемапил бы Java-плагин
		"string":       "string",
		"int32":        "int32",
		"sqlx-int32":   "sqlx-int32",
		"java.foo.Bar": "java.foo.Bar",
	}
	for in, want := range cases {
		if got := tm.Target(in); got != want {
			t.Errorf("%s: got %q, want %q", in, got, want)
		}
	}
}

func TestGoAliasesCoverMetadataTypes(t *testing.T) {
	for _, j := range []string{"java.lang.Integer", "java.util.Date", "java.time.LocalDateTime", "byte[]", "java.lang.Object"} {
		if _, ok := goAlias(j); !ok {
			t.Errorf("%s: no Go name", j)
		}
	}
}
