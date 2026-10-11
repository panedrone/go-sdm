package cg

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/panedrone/go-sdm/internal/sdmxml"
)

// SplitTargetType — "type->assign_func" из type-map. ok == false, если функции нет.
func SplitTargetType(target string) (typ, assign string, ok bool) {
	parts := jsplitLit(target, "->")
	if len(parts) > 1 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
	}
	return strings.TrimSpace(target), "", false
}

// TypeMap — <type-map> (JaxbTypeMap.java).
//
// <type detected="..."> — два словаря наравне: Java-имена, как у IDE-плагина
// (java.lang.Integer), и сырые имена драйвера (pg: int4, MySQL: INT, SQLite: INTEGER).
// Ключи — одна карта, как у плагина. Сырое имя метаданных подставляется только на первом
// проходе (TargetRaw); на повторном проходе ядра ищется сама строка типа, как в Java.
type TypeMap struct {
	detected    map[string]string
	defaultType string
}

func NewTypeMap(tm sdmxml.TypeMap) *TypeMap {
	res := &TypeMap{detected: map[string]string{}, defaultType: tm.Default}
	for _, t := range tm.Types {
		res.detected[t.Detected] = t.Target
	}
	return res
}

// Unmapped — тип, объявленный в XML Java-именем плагина (java.lang.Integer), для которого
// в type-map нет ключа: вместо него берётся тип колонки из метаданных.
func (m *TypeMap) Unmapped(typ string) bool {
	typ = strings.TrimSpace(typ)
	if !strings.HasPrefix(typ, "java.") {
		return false
	}
	_, ok := m.detected[typ]
	return !ok
}

// Target — detected — уже отображённый тип или явно объявленный в XML.
func (m *TypeMap) Target(detected string) string {
	return m.TargetRaw(detected, "")
}

// TargetRaw — тип из метаданных: Java-имя detected и сырое имя драйвера raw ("" — нет).
// Пустой type-map — сырое имя как есть: видно, какие ключи нужны (misc.html#type-map).
func (m *TypeMap) TargetRaw(detected, raw string) string {
	if len(m.detected) == 0 {
		if raw != "" {
			return raw
		}
		return detected // нет переопределений — как есть, независимо от default
	}
	if t, ok := m.detected[detected]; ok {
		return t
	}
	if t, ok := m.detected[raw]; ok && raw != "" {
		return t
	}
	if strings.TrimSpace(m.defaultType) != "" {
		return m.defaultType
	}
	if raw != "" {
		return raw // ключа нет — как есть, имя драйвера, а не Java
	}
	return detected
}

// Macros — <macros> (JaxbMacros.java). <vm-xml> исполняется (vmxml.go), текстового
// Velocity (<vm>) в Go-версии нет.
type Macros struct {
	custom  map[string]string
	names   []string // custom по порядку: в Java HashMap, результат от порядка не зависит
	vmXML   map[string][]vmNode
	vmNames []string
	vmText  map[string]bool
	vmErr   map[string]error // <vm-xml>, который не разобрался: ошибка — при использовании
}

func NewMacros(list []sdmxml.Macro) *Macros {
	res := &Macros{custom: map[string]string{}, vmXML: map[string][]vmNode{}, vmText: map[string]bool{},
		vmErr: map[string]error{}}
	for _, m := range list {
		switch {
		case m.Value != nil:
			if *m.Value != "=built-in=" {
				res.custom[m.Name] = *m.Value
			}
		case m.VM != nil:
			res.vmText[m.Name] = true
		case m.VMXML != nil:
			nodes, err := parseVMXML(m.VMXML.Inner)
			if err != nil {
				res.vmErr[m.Name] = err
			}
			res.vmXML[m.Name] = nodes
		}
	}
	for n := range res.custom {
		res.names = append(res.names, n)
	}
	sort.Strings(res.names)
	for n := range res.vmXML {
		res.vmNames = append(res.vmNames, n)
	}
	sort.Strings(res.vmNames)
	return res
}

// ParseTargetType — подставляет макросы в целевой тип поля.
func (m *Macros) ParseTargetType(target string, fi FieldInfo) (string, error) {
	parsed, err := m.parseRecursive(0, target, fi)
	if err != nil {
		return "", err
	}
	return substituteTypeParams(substituteBuiltIn(parsed, fi)), nil
}

func (m *Macros) parseRecursive(depth int, target string, fi FieldInfo) (string, error) {
	if depth > 100 {
		return "", errors.New("depth overflow: " + target)
	}
	found := false
	for _, n := range m.names {
		if strings.Contains(target, n) { // одна замена за проход
			target = strings.ReplaceAll(target, n, m.custom[n])
			found = true
		}
	}
	if !found {
		tmp, ok, err := m.substituteVM(target, fi)
		if err != nil {
			return "", err
		}
		if !ok {
			return target, nil
		}
		target = tmp // одна замена за проход
	}
	return m.parseRecursive(depth+1, target, fi)
}

func (m *Macros) substituteVM(target string, fi FieldInfo) (string, bool, error) {
	for _, n := range m.vmNames {
		if !strings.Contains(target, n) {
			continue
		}
		if err := m.vmErr[n]; err != nil {
			return "", false, fmt.Errorf("%s: %w", n, err)
		}
		var out strings.Builder
		if err := execVM(m.vmXML[n], fi, &out); err != nil {
			return "", false, fmt.Errorf("%s: %w", n, err)
		}
		return strings.ReplaceAll(target, n, out.String()), true, nil
	}
	for n := range m.vmText {
		if strings.Contains(target, n) {
			return "", false, errors.New("Velocity macros (<vm>) are not supported by go-sdm, use <vm-xml>: " + n)
		}
	}
	return target, false, nil
}

var builtIn = []struct {
	name string
	fn   func(FieldInfo) string
}{
	{"${lower_snake_case(column)}", func(f FieldInfo) string { return CamelCaseToLowerSnakeCase(f.columnName) }},
	{"${camelCase(column)}", func(f FieldInfo) string { return LowerCamel(f.columnName) }},
	{"${TitleCase(column)}", func(f FieldInfo) string { return Title(f.columnName) }},
	{"${kebab-case(column)}", func(f FieldInfo) string { return ToKebabCase(f.columnName) }},
	{"${column}", func(f FieldInfo) string { return f.columnName }},
}

func substituteBuiltIn(t string, fi FieldInfo) string {
	if t == "" {
		return t
	}
	for _, b := range builtIn {
		if strings.Contains(t, b.name) {
			t = strings.ReplaceAll(t, b.name, b.fn(fi))
		}
	}
	return t
}

// substituteTypeParams — "type${0}|0:value" -> "typevalue"; неподставленные ${...} убираются.
func substituteTypeParams(t string) string {
	if start := strings.IndexByte(t, '|'); start != -1 {
		params := map[string]string{}
		for _, p := range jsplit(t[start:], reBar) {
			i := strings.IndexByte(p, ':')
			if i == -1 {
				continue
			}
			params["${"+p[:i]+"}"] = p[i+1:]
		}
		t = t[:start]
		names := make([]string, 0, len(params))
		for n := range params {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			t = strings.ReplaceAll(t, n, params[n])
		}
	}
	for _, call := range findMacroCalls(t) {
		t = strings.ReplaceAll(t, call, "")
	}
	return t
}

func findMacroCalls(s string) []string {
	var res []string
	inside, start := false, -1
	for i := 0; i < len(s); i++ {
		if strings.HasPrefix(s[i:], "${") {
			inside, start = true, i
			continue
		}
		if inside && s[i] == '}' {
			res = append(res, s[start:i+1])
			inside, start = false, -1
		}
	}
	return res
}
