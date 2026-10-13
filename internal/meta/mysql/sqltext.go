package mysql

import (
	"strings"
)

// nullParams replaces '?' with NULL outside strings, `identifiers` and comments: this is how Connector/J
// (client-side prepare) runs the query to find out the result columns. It also returns
// the number of parameters.
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

// skipQuoted returns the index after the closing quote (or the end of the string).
func skipQuoted(sql string, start int, q byte, backslash bool) int {
	for i := start + 1; i < len(sql); i++ {
		switch sql[i] {
		case '\\':
			if backslash {
				i++
			}
		case q:
			if i+1 < len(sql) && sql[i+1] == q {
				i++ // doubled quote
				continue
			}
			return i + 1
		}
	}
	return len(sql)
}

// unescapeCall turns "{call p(?)}" into "call p(?)", as the escape processing of Connector/J does.
func unescapeCall(sql string) string {
	s := strings.TrimSpace(sql)
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		return strings.TrimSpace(s[1 : len(s)-1])
	}
	return sql
}

// firstWord returns the first word of the query in lower case, skipping leading comments.
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

// callArgs returns the top-level arguments of "call p(a, ?, 'x')"; nil if it is not a procedure call.
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
