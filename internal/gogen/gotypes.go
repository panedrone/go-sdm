// Package gogen — генерация Go-кода: порт GoCG, GoTypes, GoStructFormatter и go.vm.
package gogen

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	reWS     = regexp.MustCompile(`\s+`)
	reCRLF   = regexp.MustCompile(`[\r\n]+`)
	reDollar = regexp.MustCompile(`[$]`)
)

// jsplit — String.split(regex) из Java (см. cg.jsplit).
func jsplit(s string, re *regexp.Regexp) []string {
	if !re.MatchString(s) {
		return []string{s}
	}
	parts := re.Split(s, -1)
	n := len(parts)
	for n > 0 && parts[n-1] == "" {
		n--
	}
	return parts[:n]
}

func first(parts []string) string {
	if len(parts) == 0 {
		return "" // в Java здесь было бы ArrayIndexOutOfBounds
	}
	return parts[0]
}

// typeImport — "time:time.Time `json:...`" -> "time"; "" если импорта нет.
func typeImport(t string) string {
	part := first(jsplit(strings.TrimSpace(t), reWS))
	i := strings.IndexByte(part, ':')
	if i == -1 {
		return ""
	}
	return part[:i]
}

// typeWithoutImport — "time:time.Time `json:"t_date"`${0}" -> "time.Time `json:"t_date"`".
func typeWithoutImport(t string) string {
	initial := first(jsplit(t, reDollar)) // для параметров
	i := strings.IndexByte(first(jsplit(initial, reWS)), ':')
	if i == -1 {
		return initial
	}
	return initial[i+1:]
}

// typeWithoutImportAndTag — "time:time.Time `json:...`" -> "time.Time".
func typeWithoutImportAndTag(t string) string {
	return first(jsplit(typeWithoutImport(t), reWS))
}

// padRight — String.format("%-N.Ns", s): дополнить пробелами и обрезать до N символов.
func padRight(s string, n int) string {
	if utf8.RuneCountInString(s) > n {
		return string([]rune(s)[:n])
	}
	return fmt.Sprintf("%-*s", n, s)
}
