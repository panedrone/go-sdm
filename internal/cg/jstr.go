// Package cg is the language-independent core of the generator: a port of com.sqldalmaker.cg of the Java plug-in.
//
// The port is deliberately close to the original (the same names, the same branches, the same error
// messages): the output is compared with the plug-in byte for byte, and a divergence is easier to find when
// the code can be put next to the Java source.
package cg

import (
	"regexp"
	"strings"
)

// jsplit is String.split(regex) of Java: no matches give {s}, trailing empty strings
// are dropped (and the result may become empty: ",".split(",") == {}).
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

// jsplitLit is the same for a string separator.
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
