package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"unicode/utf8"
)

// status — одна строка состояния, переписываемая через \r. Только в терминал: в CI и в
// перенаправленный вывод не пишет ничего.
type status struct {
	w   io.Writer
	len int // длина последней строки — затереть пробелами
}

func newStatus(w io.Writer) *status {
	if f, ok := w.(*os.File); ok {
		if fi, err := f.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
			return &status{w: w}
		}
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

// clear — убрать строку перед обычным выводом.
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

// dbLabel — куда подключаемся, без логина и пароля.
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
