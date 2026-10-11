package cg

import (
	"testing"

	"github.com/panedrone/go-sdm/internal/sdmxml"
)

func TestTypeMapRaw(t *testing.T) {
	tm := NewTypeMap(sdmxml.TypeMap{Types: []sdmxml.Type{
		{Detected: "int4", Target: "int32${json} -> SetInt32"},
		{Detected: "gorm-int4", Target: "int32${json-gorm} -> SetInt32"},
		{Detected: "gorm-uuid+auto", Target: "uuid.UUID|1:;default:gen_random_uuid()"},
		{Detected: "java.lang.Long", Target: "int64-from-java"}, // Java-ключ главнее сырого
		{Detected: "int8", Target: "int64-from-raw"},
		{Detected: "text", Target: "string${json} -> SetString"},
	}})
	cases := []struct{ detected, raw, want string }{
		{"java.lang.Integer", "int4", "int32${json} -> SetInt32"},
		{"gorm-java.lang.Integer", "gorm-int4", "int32${json-gorm} -> SetInt32"},
		{"gorm-java.util.UUID+auto", "gorm-uuid+auto", "uuid.UUID|1:;default:gen_random_uuid()"},
		{"java.lang.Long", "int8", "int64-from-java"},
		{"java.lang.String", "varchar", "varchar"}, // нет ни того ни другого — имя драйвера
		// второй проход (raw == ""): ключи — одна карта, как у плагина: совпадение строки
		// с ключом перемапливает (для совместимости с конфигами плагина: "Object", "int", ...)
		{"text", "", "string${json} -> SetString"},
	}
	for _, c := range cases {
		if got := tm.TargetRaw(c.detected, c.raw); got != c.want {
			t.Errorf("%s / %s: got %q, want %q", c.detected, c.raw, got, c.want)
		}
	}
}

func TestTypeMapEmpty(t *testing.T) {
	tm := NewTypeMap(sdmxml.TypeMap{})
	if got := tm.TargetRaw("java.lang.Integer", "INTEGER"); got != "INTEGER" {
		t.Errorf("empty type-map, raw known: %q", got)
	}
	if got := tm.TargetRaw("java.lang.Object", ""); got != "java.lang.Object" {
		t.Errorf("empty type-map, raw unknown: %q", got)
	}
}
