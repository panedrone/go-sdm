package pg

import "testing"

func TestConnString(t *testing.T) {
	cases := []struct {
		url, user, pwd, want string
	}{
		{"jdbc:postgresql://localhost:5432/my_tests", "postgres", "sa", "postgres://postgres:sa@localhost:5432/my_tests"},
		{"jdbc:postgresql://h/db?sslmode=disable", "u", "", "postgres://u@h/db?sslmode=disable"},
		{"jdbc:postgresql://h/db?currentSchema=s1", "u", "p", "postgres://u:p@h/db?search_path=s1"},
		{"jdbc:postgresql://h/db", "u", "p@ss:/w", "postgres://u:p%40ss%3A%2Fw@h/db"},
		{"postgres://x:y@h/db", "u", "p", "postgres://x:y@h/db"}, // свои данные в URL главнее
	}
	for _, c := range cases {
		got, err := ConnString(c.url, c.user, c.pwd)
		if err != nil {
			t.Errorf("%q: %v", c.url, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q:\n got  %q\n want %q", c.url, got, c.want)
		}
	}
	if _, err := ConnString("jdbc:mysql://h/db", "u", "p"); err == nil {
		t.Error("error expected for a non-PostgreSQL URL")
	}
}
