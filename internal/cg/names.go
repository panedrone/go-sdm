package cg

import (
	"errors"
	"strings"
	"unicode"
)

// FieldNamesMode is how fields and parameters are named (Names.java, FieldNamesMode.java).
type FieldNamesMode int

const (
	AsIs FieldNamesMode = iota
	LowerCamelCase
	SnakeCase
	TitleCase
)

// FieldNamesModeOf maps <dto field-names-mode>: 0 is TitleCase, 1 is camelCase, otherwise snake_case.
func FieldNamesModeOf(mode int) FieldNamesMode {
	switch mode {
	case 0:
		return TitleCase
	case 1:
		return LowerCamelCase
	default:
		return SnakeCase
	}
}

func CamelCaseToLowerSnakeCase(src string) string {
	return strings.ToLower(reCamel.ReplaceAllString(src, "${1}_${2}"))
}

func ToKebabCase(src string) string {
	return strings.Join(jsplit(CamelCaseToLowerSnakeCase(src), reUnderSc), "-")
}

// FileNameSnakeCase is the file name of a class: "TasksDao" -> "tasks_dao.go".
func FileNameSnakeCase(className, ext string) string {
	return CamelCaseToLowerSnakeCase(className) + "." + ext
}

func replaceFirstRune(s string, f func(rune) rune) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	r[0] = f(r[0])
	return string(r)
}

func isUpperCase(s string) bool {
	for _, r := range s {
		if unicode.IsLower(r) {
			return false
		}
	}
	return s != ""
}

func MethodName(name string, mode FieldNamesMode) string {
	switch mode {
	case LowerCamelCase:
		return LowerCamel(name)
	case TitleCase:
		return Title(name)
	case SnakeCase:
		return CamelCaseToLowerSnakeCase(name)
	}
	return name
}

var ids = []string{"Id", "Uuid"}

func fixIds(res string) string {
	for _, id := range ids {
		if res == id {
			return strings.ToUpper(res)
		}
		if strings.HasSuffix(res, id) {
			return res[:len(res)-len(id)] + strings.ToUpper(id)
		}
	}
	return res
}

func LowerCamel(s string) string {
	return fixIds(lowerCamelOrTitle(s, false))
}

func Title(s string) string {
	return fixIds(lowerCamelOrTitle(s, true))
}

func lowerCamelOrTitle(str string, title bool) string {
	if strings.ToUpper(str) == str {
		str = strings.ToLower(str) // "PROJECTS" --> "projects"
	}
	if !strings.Contains(str, "_") {
		if title {
			return replaceFirstRune(str, unicode.ToTitle)
		}
		if isUpperCase(str) {
			return strings.ToLower(str)
		}
		return replaceFirstRune(str, unicode.ToLower)
	}
	var sb strings.Builder
	for i, s := range jsplit(str, reUnderSc) {
		if s == "" {
			continue // _ALL_FILE_GROUPS
		}
		r := []rune(s)
		switch {
		case i == 0 && title:
			r[0] = unicode.ToTitle(r[0])
		case i == 0:
			r[0] = unicode.ToLower(r[0])
		default:
			r[0] = unicode.ToTitle(r[0])
		}
		sb.WriteRune(r[0])
		if len(r) > 1 {
			tail := string(r[1:])
			if isUpperCase(tail) {
				sb.WriteString(strings.ToLower(tail))
			} else {
				sb.WriteString(tail)
			}
		}
	}
	return sb.String()
}

// Model: "sa-Task" -> "sa"; no model without '-'.
func Model(dtoClassName string) string {
	if i := strings.IndexByte(dtoClassName, '-'); i >= 0 {
		return dtoClassName[:i]
	}
	return ""
}

func ClassNameWithoutModel(dtoClassName string) string {
	if i := strings.IndexByte(dtoClassName, '-'); i >= 0 {
		return dtoClassName[i+1:]
	}
	return dtoClassName
}

// pkColNameAlias: xerial gives the PK in lower case and as '[employeeid] asc' (composite PK).
func pkColNameAlias(pk string) (string, error) {
	pk = strings.TrimSpace(strings.NewReplacer("[", "", "]", "").Replace(strings.ToLower(pk)))
	if strings.HasSuffix(pk, " asc") {
		pk = jsplit(pk, reAsc)[0]
	}
	if strings.HasSuffix(pk, " desc") {
		parts := jsplit(pk, reDesc)
		if len(parts) < 2 { // as in Java: split(" desc")[1]
			return "", errors.New("Index 1 out of bounds for length 1")
		}
		pk = parts[1]
	}
	return strings.TrimSpace(pk), nil
}
