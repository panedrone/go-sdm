package pg

import "testing"

func TestToNative(t *testing.T) {
	cases := []struct {
		in, want string
		n        int
	}{
		{"select * from tests", "select * from tests", 0},
		{"select * from tests where t_id = ? and t_name = ?", "select * from tests where t_id = $1 and t_name = $2", 2},
		{"CALL sp_out_ref_cursor2(?, 'ref2', ?)", "CALL sp_out_ref_cursor2($1, 'ref2', $2)", 2},
		{"{call get_tests(?)}", "select * from get_tests($1) as result", 1},
		{" { CALL get_tests(?, ?) } ", "select * from get_tests($1, $2) as result", 2},
		// внутри строк, идентификаторов, $$-строк и комментариев '?' не трогается
		{"select '?', \"a?\", ? from t", "select '?', \"a?\", $1 from t", 1},
		{"select 'it''s ?' || ?", "select 'it''s ?' || $1", 1},
		{"select E'\\' ?' || ?", "select E'\\' ?' || $1", 1},
		{"select $$ ? $$, $tag$ ? $tag$, ?", "select $$ ? $$, $tag$ ? $tag$, $1", 1},
		{"select ? -- ?\n, ?", "select $1 -- ?\n, $2", 2},
		{"select /* ? /* ? */ ? */ ?", "select /* ? /* ? */ ? */ $1", 1},
		// '??' — экранированный '?', так в JDBC пишут jsonb-операторы
		{"select d ?? 'k', d ??| array['a'] from t where id = ?", "select d ? 'k', d ?| array['a'] from t where id = $1", 1},
		// '$1' и a$b — не $-строки
		{"select a$b, $1 from t", "select a$b, $1 from t", 0},
	}
	for _, c := range cases {
		got, n, err := toNative(c.in)
		if err != nil {
			t.Errorf("%q: %v", c.in, err)
			continue
		}
		if got != c.want || n != c.n {
			t.Errorf("%q:\n got  %q (%d)\n want %q (%d)", c.in, got, n, c.want, c.n)
		}
	}
}

func TestToNativeErrors(t *testing.T) {
	for _, in := range []string{
		"{? = call f(?)}",
		"select 'unterminated",
		"select \"unterminated",
		"select /* unterminated",
	} {
		if _, _, err := toNative(in); err == nil {
			t.Errorf("%q: error expected", in)
		}
	}
}
