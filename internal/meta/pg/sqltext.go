package pg

import (
	"errors"
	"strconv"
	"strings"
)

// toNative turns JDBC SQL into what PostgreSQL understands, as pgjdbc does:
//   - '{call f(...)}' -> 'select * from f(...) as result';
//   - '?' -> '$1', '$2', ... outside strings, quoted identifiers, $$-strings and comments;
//   - '??' -> '?' (this is how jsonb operators '?', '?|', '?&' are written in JDBC).
//
// It also returns the number of parameters.
func toNative(jdbcSQL string) (string, int, error) {
	sql, err := unescapeCall(jdbcSQL)
	if err != nil {
		return "", 0, err
	}
	return placeholders(sql)
}

func unescapeCall(sql string) (string, error) {
	s := strings.TrimSpace(sql)
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return sql, nil
	}
	inner := strings.TrimSpace(s[1 : len(s)-1])
	if len(inner) > 5 && strings.EqualFold(inner[:5], "call ") {
		return "select * from " + strings.TrimSpace(inner[5:]) + " as result", nil
	}
	return "", errors.New("unsupported JDBC escape, use 'call ...' or 'select ...' without '{}': " + sql)
}

func placeholders(sql string) (string, int, error) {
	var b strings.Builder
	b.Grow(len(sql) + 16)
	n := 0
	i := 0
	for i < len(sql) {
		c := sql[i]
		switch {
		case c == '\'':
			// E'...' understands backslash, a plain string only ''
			backslash := i > 0 && (sql[i-1] == 'E' || sql[i-1] == 'e') && (i < 2 || !isIdent(sql[i-2]))
			end := skipQuoted(sql, i, '\'', backslash)
			if end < 0 {
				return "", 0, errors.New("unterminated string literal: " + sql)
			}
			b.WriteString(sql[i:end])
			i = end
		case c == '"':
			end := skipQuoted(sql, i, '"', false)
			if end < 0 {
				return "", 0, errors.New("unterminated quoted identifier: " + sql)
			}
			b.WriteString(sql[i:end])
			i = end
		case c == '$' && (i == 0 || !isIdent(sql[i-1])):
			end := skipDollarQuoted(sql, i)
			if end < 0 {
				b.WriteByte(c) // not a $-string: $1, $tag without a pair, etc.
				i++
				continue
			}
			b.WriteString(sql[i:end])
			i = end
		case c == '-' && i+1 < len(sql) && sql[i+1] == '-':
			end := strings.IndexByte(sql[i:], '\n')
			if end < 0 {
				end = len(sql)
			} else {
				end += i + 1
			}
			b.WriteString(sql[i:end])
			i = end
		case c == '/' && i+1 < len(sql) && sql[i+1] == '*':
			end := skipBlockComment(sql, i)
			if end < 0 {
				return "", 0, errors.New("unterminated comment: " + sql)
			}
			b.WriteString(sql[i:end])
			i = end
		case c == '?':
			if i+1 < len(sql) && sql[i+1] == '?' {
				b.WriteByte('?')
				i += 2
				continue
			}
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			i++
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String(), n, nil
}

func isIdent(c byte) bool {
	return c == '_' || c == '$' || '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || c >= 0x80
}

// skipQuoted returns the index after the closing quote or -1.
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
	return -1
}

// skipDollarQuoted returns the index after the closing $tag$, or -1 if no $-string starts at start.
func skipDollarQuoted(sql string, start int) int {
	j := start + 1
	if j < len(sql) && '0' <= sql[j] && sql[j] <= '9' {
		return -1 // $1 is a parameter, not a tag
	}
	for j < len(sql) && sql[j] != '$' {
		if !isIdent(sql[j]) {
			return -1
		}
		j++
	}
	if j >= len(sql) {
		return -1
	}
	tag := sql[start : j+1]
	end := strings.Index(sql[j+1:], tag)
	if end < 0 {
		return -1
	}
	return j + 1 + end + len(tag)
}

// skipBlockComment: /* */ comments nest in PostgreSQL.
func skipBlockComment(sql string, start int) int {
	depth := 0
	for i := start; i+1 < len(sql); i++ {
		switch {
		case sql[i] == '/' && sql[i+1] == '*':
			depth++
			i++
		case sql[i] == '*' && sql[i+1] == '/':
			depth--
			i++
			if depth == 0 {
				return i + 1
			}
		}
	}
	return -1
}
