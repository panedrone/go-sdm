package main

import (
	"fmt"
	"io"
	"net/url"
	"strings"
	"unicode/utf8"
)

// status is one status line, rewritten via \r. Terminal only: in CI and in
// redirected output it writes nothing.
type status struct {
	w   io.Writer
	len int // length of the last line, to blank it out with spaces
}

func newStatus(w io.Writer) *status {
	if isTerminal(w) {
		return &status{w: w}
	}
	return &status{}
}

func (s *status) show(format string, a ...any) {
	if s.w == nil {
		return
	}
	line := fmt.Sprintf(format, a...)
	n := utf8.RuneCountInString(line)
	pad := ""
	if s.len > n {
		pad = strings.Repeat(" ", s.len-n)
	}
	fmt.Fprint(s.w, "\r"+line+pad)
	s.len = n
}

// clear removes the line before regular output.
func (s *status) clear() {
	if s.w == nil || s.len == 0 {
		return
	}
	fmt.Fprint(s.w, "\r"+strings.Repeat(" ", s.len)+"\r")
	s.len = 0
}

func (s *status) progress(n, total int, class string) {
	s.show("[%d/%d] %s", n, total, class)
}

// dbLabel is where we connect, without login and password.
func dbLabel(raw string) string {
	raw = strings.TrimSpace(raw)
	s := strings.TrimPrefix(raw, "jdbc:")
	if f, ok := strings.CutPrefix(s, "sqlite:"); ok {
		return f
	}
	if u, err := url.Parse(s); err == nil && u.Host != "" {
		return u.Host + u.Path
	}
	return "the database"
}
