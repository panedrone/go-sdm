package pg

// PostgreSQL types as pgjdbc (42.2.x) sees them after JavaTypes.refine_java_type_name
// of the Java plug-in: ResultSetMetaData.getColumnClassName, getPrecision/getScale/getColumnDisplaySize
// and DATA_TYPE from DatabaseMetaData.getColumns. Verified by a cli/metadump dump against MetaDump.java.

const (
	oidBool        = 16
	oidBytea       = 17
	oidChar        = 18
	oidName        = 19
	oidInt8        = 20
	oidInt2        = 21
	oidInt4        = 23
	oidText        = 25
	oidOID         = 26
	oidJSON        = 114
	oidXML         = 142
	oidPoint       = 600
	oidLseg        = 601
	oidPath        = 602
	oidBox         = 603
	oidPolygon     = 604
	oidLine        = 628
	oidFloat4      = 700
	oidFloat8      = 701
	oidUnknown     = 705
	oidCircle      = 718
	oidMoney       = 790
	oidBpchar      = 1042
	oidVarchar     = 1043
	oidDate        = 1082
	oidTime        = 1083
	oidTimestamp   = 1114
	oidTimestamptz = 1184
	oidInterval    = 1186
	oidTimetz      = 1266
	oidBit         = 1560
	oidNumeric     = 1700
	oidRefcursor   = 1790
	oidUUID        = 2950
)

const (
	javaObject     = "java.lang.Object"
	javaArray      = "java.sql.Array"
	unknownLength  = 2147483647
	sqlTypeArray   = 2003
	sqlTypeOther   = 1111
	numericNoLimit = 131089 // display size of numeric without typmod
)

var javaTypes = map[uint32]string{
	oidInt2:        "java.lang.Integer",
	oidInt4:        "java.lang.Integer",
	oidInt8:        "java.lang.Long",
	oidOID:         "java.lang.Long",
	oidNumeric:     "java.math.BigDecimal",
	oidFloat4:      "java.lang.Float",
	oidFloat8:      "java.lang.Double",
	oidMoney:       "org.postgresql.util.PGmoney",
	oidBool:        "java.lang.Boolean",
	oidBit:         "java.lang.Boolean",
	oidChar:        "java.lang.String",
	oidName:        "java.lang.String",
	oidBpchar:      "java.lang.String",
	oidVarchar:     "java.lang.String",
	oidText:        "java.lang.String",
	oidUnknown:     "java.lang.String", // 'select null c'
	oidDate:        "java.util.Date",   // java.sql.Date/Time/Timestamp -> java.util.Date
	oidTime:        "java.util.Date",
	oidTimetz:      "java.util.Date",
	oidTimestamp:   "java.util.Date",
	oidTimestamptz: "java.util.Date",
	oidBytea:       "byte[]",
	oidUUID:        "java.util.UUID",
	oidJSON:        "org.postgresql.util.PGobject", // while jsonb is java.lang.Object in pgjdbc 42.2
	oidXML:         "java.sql.SQLXML",
	oidInterval:    "org.postgresql.util.PGInterval",
	oidPoint:       "org.postgresql.geometric.PGpoint",
	oidLseg:        "org.postgresql.geometric.PGlseg",
	oidPath:        "org.postgresql.geometric.PGpath",
	oidBox:         "org.postgresql.geometric.PGbox",
	oidPolygon:     "org.postgresql.geometric.PGpolygon",
	oidLine:        "org.postgresql.geometric.PGline",
	oidCircle:      "org.postgresql.geometric.PGcircle",
	oidRefcursor:   "java.sql.ResultSet",
}

// java.sql.Types
var sqlTypes = map[uint32]int{
	oidInt2:        5,  // SMALLINT
	oidInt4:        4,  // INTEGER
	oidInt8:        -5, // BIGINT
	oidOID:         -5,
	oidNumeric:     2, // NUMERIC
	oidFloat4:      7, // REAL
	oidFloat8:      8, // DOUBLE
	oidMoney:       8,
	oidBool:        -7, // BIT
	oidBit:         -7,
	oidChar:        1, // CHAR
	oidBpchar:      1,
	oidName:        12, // VARCHAR
	oidVarchar:     12,
	oidText:        12,
	oidDate:        91, // DATE
	oidTime:        92, // TIME
	oidTimetz:      92,
	oidTimestamp:   93, // TIMESTAMP
	oidTimestamptz: 93,
	oidBytea:       -2,   // BINARY
	oidXML:         2009, // SQLXML
}

// javaType is the Java class name for a type that is not an array.
func javaType(oid uint32) string {
	if t, ok := javaTypes[oid]; ok {
		return t
	}
	return javaObject
}

func sqlType(oid uint32) int {
	if t, ok := sqlTypes[oid]; ok {
		return t
	}
	return sqlTypeOther
}

// secondSize is the fraction of a second in the display size of time types (as in pgjdbc).
func secondSize(typmod int32) int {
	switch typmod {
	case -1:
		return 6 + 1
	case 0:
		return 0
	case 1:
		return 2 + 1 // SELECT '0:0:0.1'::time(1) returns two digits
	default:
		return int(typmod) + 1
	}
}

func numericPrecision(typmod int32) int {
	if typmod == -1 {
		return 0
	}
	return int(((typmod - 4) >> 16) & 0xffff)
}

func numericScale(typmod int32) int {
	if typmod == -1 {
		return 0
	}
	return int((typmod - 4) & 0xffff)
}

func displaySize(oid uint32, typmod int32) int {
	switch oid {
	case oidInt2:
		return 6
	case oidInt4:
		return 11
	case oidOID:
		return 10
	case oidInt8:
		return 20
	case oidFloat4:
		return 15
	case oidFloat8:
		return 25
	case oidChar, oidBool:
		return 1
	case oidDate:
		return 13
	case oidTime:
		return 8 + secondSize(typmod)
	case oidTimetz:
		return 8 + secondSize(typmod) + 6
	case oidTimestamp:
		return 22 + secondSize(typmod)
	case oidTimestamptz:
		return 22 + secondSize(typmod) + 6
	case oidInterval:
		return 49
	case oidVarchar, oidBpchar:
		if typmod == -1 {
			return unknownLength
		}
		return int(typmod - 4)
	case oidNumeric:
		if typmod == -1 {
			return numericNoLimit
		}
		p := numericPrecision(typmod)
		s := numericScale(typmod)
		if s != 0 {
			return p + 1 + 1 // sign and point
		}
		return p + 1
	case oidBit:
		return int(typmod)
	default:
		return unknownLength
	}
}

func precision(oid uint32, typmod int32) int {
	switch oid {
	case oidInt2:
		return 5
	case oidInt4, oidOID:
		return 10
	case oidInt8:
		return 19
	case oidFloat4:
		return 8
	case oidFloat8:
		return 17
	case oidNumeric:
		return numericPrecision(typmod)
	case oidChar, oidBool:
		return 1
	case oidVarchar, oidBpchar:
		if typmod == -1 {
			return unknownLength
		}
		return int(typmod - 4)
	case oidBit:
		return int(typmod)
	case oidDate, oidTime, oidTimetz, oidTimestamp, oidTimestamptz, oidInterval:
		return displaySize(oid, typmod)
	default:
		return unknownLength
	}
}

func scale(oid uint32, typmod int32) int {
	switch oid {
	case oidFloat4:
		return 8
	case oidFloat8:
		return 17
	case oidNumeric:
		return numericScale(typmod)
	case oidTime, oidTimetz, oidTimestamp, oidTimestamptz:
		if typmod == -1 {
			return 6
		}
		return int(typmod)
	case oidInterval:
		if typmod == -1 {
			return 6
		}
		return int(typmod & 0xffff)
	default:
		return 0
	}
}
