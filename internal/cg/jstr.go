// Package cg — языконезависимое ядро генератора: порт com.sqldalmaker.cg Java-плагина.
//
// Порт намеренно близок к оригиналу (те же имена, те же развилки, те же сообщения об
// ошибках): вывод сверяется с плагином побайтово, и расхождение проще найти, когда
// код можно положить рядом с Java-исходником.
package cg

import (
	"regexp"
	"strings"
)

// jsplit — String.split(regex) из Java: без совпадений — {s}, пустые строки в конце
// отбрасываются (и результат может стать пустым: ",".split(",") == {}).
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

// jsplitLit — то же для разделителя-строки.
func jsplitLit(s, sep string) []string {
	if !strings.Contains(s, sep) {
		return []string{s}
	}
	parts := strings.Split(s, sep)
	n := len(parts)
	for n > 0 && parts[n-1] == "" {
		n--
	}
	return parts[:n]
}

var (
	reWS      = regexp.MustCompile(`\s+`)
	reNL      = regexp.MustCompile(`\n+`)
	reCRLF    = regexp.MustCompile(`[\r\n]+`)
	reCamel   = regexp.MustCompile(`([a-z])([A-Z]+)`)
	reDollar  = regexp.MustCompile(`[$]`)
	reBar     = regexp.MustCompile(`[|]`)
	reAsc     = regexp.MustCompile(` asc`)
	reDesc    = regexp.MustCompile(` desc`)
	reUnderSc = regexp.MustCompile(`_`)
)
