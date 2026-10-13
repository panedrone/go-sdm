// Package gogen generates Go code: a port of GoCG, GoTypes, GoStructFormatter and go.vm.
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

// jsplit is String.split(regex) of Java (see cg.jsplit).
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
		return "" // Java would throw ArrayIndexOutOfBounds here
	}
	return parts[0]
}

// typeImport: "time:time.Time `json:...`" -> "time"; "" if there is no import.
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
	initial := first(jsplit(t, reDollar)) // for parameters
	i := strings.IndexByte(first(jsplit(initial, reWS)), ':')
	if i == -1 {
		return initial
	}
	return initial[i+1:]
}

// typeWithoutImportAndTag — "time:time.Time `json:...`" -> "time.Time".
//
// The type is everything before the tag (before '`'), not before the first space as in Java: a driver name missing
// from type-map is written as is, and MySQL has ones of two words ("UNSIGNED SMALLINT").
func typeWithoutImportAndTag(t string) string {
	s := typeWithoutImport(t)
	if s != strings.TrimLeft(s, " \t\r\n") {
		return first(jsplit(s, reWS)) // as in Java: a leading space means an empty type
	}
	if i := strings.IndexByte(s, '`'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// padRight is String.format("%-N.Ns", s): pad with spaces and cut to N characters.
func padRight(s string, n int) string {
	if utf8.RuneCountInString(s) > n {
		return string([]rune(s)[:n])
	}
	return fmt.Sprintf("%-*s", n, s)
}
