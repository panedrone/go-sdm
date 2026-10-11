package mysql

import (
	"strings"
)

// nullParams — '?' -> NULL вне строк, `идентификаторов` и комментариев: так Connector/J
// (клиентский prepare) выполняет запрос, чтобы узнать колонки результата. Возвращает и
// число параметров.
func nullParams(sql string) (string, int) {
	var b strings.Builder
	n := 0
	for i := 0; i < len(sql); {
		c := sql[i]
		switch {
		case c == '\'' || c == '"' || c == '`':
			end := skipQuoted(sql, i, c, c != '`')
			b.WriteString(sql[i:end])
			i = end
		case c == '#' || c == '-' && strings.HasPrefix(sql[i:], "-- ") || c == '-' && sql[i:] == "--":
			end := strings.IndexByte(sql[i:], '\n')
			if end < 0 {
				end = len(sql)
			} else {
				end += i
			}
			b.WriteString(sql[i:end])
			i = end
		case c == '/' && strings.HasPrefix(sql[i:], "/*"):
			end := strings.Index(sql[i+2:], "*/")
			if end < 0 {
				end = len(sql)
			} else {
				end += i + 4
			}
			b.WriteString(sql[i:end])
			i = end
		case c == '?':
			n++
			b.WriteString("NULL")
			i++
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String(), n
}

// skipQuoted — индекс после закрывающей кавычки (или конец строки).
func skipQuoted(sql string, start int, q byte, backslash bool) int {
	for i := start + 1; i < len(sql); i++ {
		switch sql[i] {
		case '\\':
			if backslash {
				i++
			}
		case q:
			if i+1 < len(sql) && sql[i+1] == q {
				i++ // удвоенная кавычка
				continue
			}
			return i + 1
		}
	}
	return len(sql)
}

// unescapeCall — "{call p(?)}" -> "call p(?)", как escape-обработка Connector/J.
func unescapeCall(sql string) string {
	s := strings.TrimSpace(sql)
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		return strings.TrimSpace(s[1 : len(s)-1])
	}
	return sql
}

// firstWord — первое слово запроса в нижнем регистре, без комментариев в начале.
func firstWord(sql string) string {
	s := strings.TrimSpace(sql)
	for {
		switch {
		case strings.HasPrefix(s, "/*"):
			if end := strings.Index(s, "*/"); end >= 0 {
				s = strings.TrimSpace(s[end+2:])
				continue
			}
		case strings.HasPrefix(s, "#") || strings.HasPrefix(s, "-- "):
			if end := strings.IndexByte(s, '\n'); end >= 0 {
				s = strings.TrimSpace(s[end+1:])
				continue
			}
			return ""
		}
		break
	}
	end := strings.IndexFunc(s, func(r rune) bool { return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') })
	if end < 0 {
		end = len(s)
	}
	return strings.ToLower(s[:end])
}

// callArgs — аргументы "call p(a, ?, 'x')" верхнего уровня; nil, если это не вызов процедуры.
func callArgs(sql string) (proc string, args []string) {
	s := strings.TrimSpace(sql)
	if firstWord(s) != "call" {
		return "", nil
	}
	s = strings.TrimSpace(s[4:])
	open := strings.IndexByte(s, '(')
	if open < 0 {
		return strings.TrimSpace(strings.TrimSuffix(s, ";")), nil
	}
	proc = strings.TrimSpace(s[:open])
	body := s[open+1:]
	depth, start := 0, 0
	for i := 0; i < len(body); i++ {
		switch c := body[i]; {
		case c == '\'' || c == '"' || c == '`':
			i = skipQuoted(body, i, c, c != '`') - 1
		case c == '(':
			depth++
		case c == ')' && depth > 0:
			depth--
		case c == ')' || c == ',' && depth == 0:
			args = append(args, strings.TrimSpace(body[start:i]))
			start = i + 1
			if c == ')' {
				return proc, args
			}
		}
	}
	return proc, args
}
