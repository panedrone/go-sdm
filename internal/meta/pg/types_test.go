package pg

import "testing"

// The expectations are taken from pgjdbc 42.2.9 (MetaDump.java on PostgreSQL); typmod as in pg_attribute.
func TestSizes(t *testing.T) {
	cases := []struct {
		name                      string
		oid                       uint32
		typmod                    int32
		precision, scale, display int
	}{
		{"int2", oidInt2, -1, 5, 0, 6},
		{"int4", oidInt4, -1, 10, 0, 11},
		{"int8", oidInt8, -1, 19, 0, 20},
		{"oid", oidOID, -1, 10, 0, 10},
		{"numeric(10,2)", oidNumeric, (10<<16 | 2) + 4, 10, 2, 12},
		{"numeric", oidNumeric, -1, 0, 0, numericNoLimit},
		{"float4", oidFloat4, -1, 8, 8, 15},
		{"float8", oidFloat8, -1, 17, 17, 25},
		{"bool", oidBool, -1, 1, 0, 1},
		{"varchar(200)", oidVarchar, 204, 200, 0, 200},
		{"bpchar(3)", oidBpchar, 7, 3, 0, 3},
		{"text", oidText, -1, unknownLength, 0, unknownLength},
		{"date", oidDate, -1, 13, 0, 13},
		{"time", oidTime, -1, 15, 6, 15},
		{"timetz", oidTimetz, -1, 21, 6, 21},
		{"timestamp", oidTimestamp, -1, 29, 6, 29},
		{"timestamptz", oidTimestamptz, -1, 35, 6, 35},
		{"timestamp(0)", oidTimestamp, 0, 22, 0, 22},
		{"interval", oidInterval, -1, 49, 6, 49},
		{"bit(1)", oidBit, 1, 1, 0, 1},
	}
	for _, c := range cases {
		p, s, d := precision(c.oid, c.typmod), scale(c.oid, c.typmod), displaySize(c.oid, c.typmod)
		if p != c.precision || s != c.scale || d != c.display {
			t.Errorf("%s: got %d/%d/%d, want %d/%d/%d", c.name, p, s, d, c.precision, c.scale, c.display)
		}
	}
}

func TestJavaType(t *testing.T) {
	cases := map[uint32]string{
		oidInt2:      "java.lang.Integer",
		oidTimestamp: "java.util.Date",
		oidBytea:     "byte[]",
		oidRefcursor: "java.sql.ResultSet",
		3802:         javaObject, // jsonb: pgjdbc 42.2 does not know it
	}
	for oid, want := range cases {
		if got := javaType(oid); got != want {
			t.Errorf("%d: got %s, want %s", oid, got, want)
		}
	}
}
