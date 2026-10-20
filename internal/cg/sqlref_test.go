package cg

import "testing"

// SQL in Python code: as SqlUtils.jdbc_sql_to_python_string with escapeJava of commons-lang 2.4.
func TestFormatJdbcSQLForPython(t *testing.T) {
	for sql, want := range map[string]string{
		"select * from t":                    "select * from t",
		"select *\r\n\tfrom t\n\n where a=?": "select * \n                 from t \n                 where a=?",
		`select 'a\b "c" d/e' from t`:        `select 'a\\b \"c\" d/e' from t`,
		"select 'é ∑ \U0001F600'":            `select '\u00E9 \u2211 \uD83D\uDE00'`,
		"select '\x01\x1f'":                  `select '\u0001\u001F'`,
	} {
		if got := FormatJdbcSQLForPython(sql); got != want {
			t.Errorf("%q:\n got  %q\n want %q", sql, got, want)
		}
	}
}
