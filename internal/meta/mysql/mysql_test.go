package mysql

import (
	"reflect"
	"testing"
)

func TestNullParams(t *testing.T) {
	cases := []struct {
		in, want string
		n        int
	}{
		{"select * from t where a = ? and b = ?", "select * from t where a = NULL and b = NULL", 2},
		{"select '?', \"?\", `a?` from t where x = ?", "select '?', \"?\", `a?` from t where x = NULL", 1},
		{"select 'it''s ?', 'a\\'?' , ?", "select 'it''s ?', 'a\\'?' , NULL", 1},
		{"select ? -- ?\n, ? # ?\n, /* ? */ ?", "select NULL -- ?\n, NULL # ?\n, /* ? */ NULL", 3},
		{"call p(?, 'x', ?)", "call p(NULL, 'x', NULL)", 2},
	}
	for _, c := range cases {
		got, n := nullParams(c.in)
		if got != c.want || n != c.n {
			t.Errorf("%q:\n got  %q (%d)\n want %q (%d)", c.in, got, n, c.want, c.n)
		}
	}
}

func TestFirstWordAndCall(t *testing.T) {
	for in, want := range map[string]string{
		"  SELECT 1":           "select",
		"/* c */ call p()":     "call",
		"-- c\nwith x as (..)": "with",
		"insert into t":        "insert",
	} {
		if got := firstWord(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
	if got := unescapeCall(" {call p_rs(?)} "); got != "call p_rs(?)" {
		t.Errorf("unescape: %q", got)
	}
	proc, args := callArgs("CALL db.p_out(?, f(1, 2), 'a,b', ?)")
	if proc != "db.p_out" || !reflect.DeepEqual(args, []string{"?", "f(1, 2)", "'a,b'", "?"}) {
		t.Errorf("callArgs: %q %q", proc, args)
	}
	if proc, _ := callArgs("select 1"); proc != "" {
		t.Errorf("not a call: %q", proc)
	}
}

// Ожидания сняты с Connector/J 8.4 (MetaDump.java на cli/metadump/mysql.txt).
func TestPrecision(t *testing.T) {
	mb := func(cs uint16) int {
		if cs == 255 { // utf8mb4
			return 4
		}
		return 1
	}
	cases := []struct {
		name string
		f    field
		java string
		p, s int
	}{
		{"int", field{typ: tLong, length: 11}, "java.lang.Integer", 10, 0},
		{"int unsigned", field{typ: tLong, length: 10, flags: flagUnsigned}, "java.lang.Long", 10, 0},
		{"literal 1", field{typ: tLongLong, length: 1}, "java.lang.Long", 19, 0},
		{"bigint unsigned", field{typ: tLongLong, length: 20, flags: flagUnsigned}, "java.math.BigInteger", 20, 0},
		{"mediumint", field{typ: tInt24, length: 8}, "java.lang.Integer", 7, 0},
		{"tinyint(1)", field{typ: tTiny, length: 1}, "java.lang.Boolean", 1, 0},
		{"decimal(10,2)", field{typ: tNewDecimal, length: 12, decimals: 2}, "java.math.BigDecimal", 10, 2},
		{"decimal(12)", field{typ: tNewDecimal, length: 13}, "java.math.BigDecimal", 12, 0},
		{"float", field{typ: tFloat, length: 12, decimals: 31}, "java.lang.Float", 12, 0},
		{"datetime(3)", field{typ: tDateTime, length: 23, charset: binaryCharset}, "java.time.LocalDateTime", 23, 0},
		{"timestamp", field{typ: tTimestamp, length: 19, charset: binaryCharset}, "java.util.Date", 19, 0},
		{"varchar(200)", field{typ: tVarString, length: 800, charset: 255}, "java.lang.String", 200, 0},
		{"tinytext", field{typ: tBlob, length: 1020, charset: 255}, "java.lang.String", 63, 0},
		{"text", field{typ: tBlob, length: 262140, charset: 255}, "java.lang.String", 16383, 0},
		{"longtext", field{typ: tBlob, length: 4294967295, charset: 255}, "java.lang.String", 536870911, 0},
		{"longblob", field{typ: tBlob, length: 4294967295, charset: binaryCharset}, "byte[]", 2147483647, 0},
		{"json", field{typ: tJSON, length: 4294967295, charset: binaryCharset}, "java.lang.String", 2147483647, 0},
		{"null", field{typ: tNull, charset: binaryCharset}, "java.lang.Object", 0, 0},
	}
	for _, c := range cases {
		if j, p, s := c.f.javaType(), c.f.precision(mb), c.f.scale(); j != c.java || p != c.p || s != c.s {
			t.Errorf("%s: got %s %d/%d, want %s %d/%d", c.name, j, p, s, c.java, c.p, c.s)
		}
	}
}

func TestColumnSize(t *testing.T) {
	i := func(v int64) *int64 { return &v }
	cases := []struct {
		name        string
		c           column
		typ, size   int
		decimalDigs int
	}{
		{"tinyint(1)", column{dataType: "tinyint", columnType: "tinyint(1)", numPrec: i(3)}, -7, 1, 0},
		{"bigint unsigned", column{dataType: "bigint", columnType: "bigint unsigned", numPrec: i(20)}, -5, 20, 0},
		{"decimal(10,2)", column{dataType: "decimal", numPrec: i(10), numScale: i(2)}, 3, 10, 2},
		{"datetime(3)", column{dataType: "datetime", dtPrec: i(3)}, 93, 23, 0},
		{"time", column{dataType: "time", dtPrec: i(0)}, 92, 8, 0},
		{"longtext", column{dataType: "longtext", charLen: i(4294967295)}, -1, 2147483647, 0},
		{"json", column{dataType: "json"}, -1, 1073741824, 0},
		{"varchar(64)", column{dataType: "varchar", charLen: i(64)}, 12, 64, 0},
	}
	for _, c := range cases {
		if typ, size, dd := c.c.sqlType(), c.c.columnSize(), c.c.decimalDigits(); typ != c.typ || size != c.size || dd != c.decimalDigs {
			t.Errorf("%s: got %d/%d/%d, want %d/%d/%d", c.name, typ, size, dd, c.typ, c.size, c.decimalDigs)
		}
	}
}

func TestConnParams(t *testing.T) {
	addr, db, u, p, err := ConnParams("jdbc:mysql://localhost/orders", "root", "pw")
	if err != nil || addr != "localhost:3306" || db != "orders" || u != "root" || p != "pw" {
		t.Errorf("%s %s %s %s %v", addr, db, u, p, err)
	}
	addr, _, u, p, _ = ConnParams("mysql://x:y@h:3307/db", "root", "pw")
	if addr != "h:3307" || u != "x" || p != "y" {
		t.Errorf("%s %s %s", addr, u, p)
	}
	if _, _, _, _, err := ConnParams("jdbc:postgresql://h/db", "u", "p"); err == nil {
		t.Error("error expected")
	}
}
