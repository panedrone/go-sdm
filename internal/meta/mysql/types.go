package mysql

import (
	"strings"
)

// Типы MySQL так, как их видит Connector/J 8.4 (после JavaTypes.refine_java_type_name
// Java-плагина). Сверено дампом cli/metadump против MetaDump.java (cli/metadump/mysql.txt).

// Типы полей протокола.
const (
	tDecimal    = 0
	tTiny       = 1
	tShort      = 2
	tLong       = 3
	tFloat      = 4
	tDouble     = 5
	tNull       = 6
	tTimestamp  = 7
	tLongLong   = 8
	tInt24      = 9
	tDate       = 10
	tTime       = 11
	tDateTime   = 12
	tYear       = 13
	tNewDate    = 14
	tVarChar    = 15
	tBit        = 16
	tTimestamp2 = 17
	tDateTime2  = 18
	tTime2      = 19
	tJSON       = 245
	tNewDecimal = 246
	tEnum       = 247
	tSet        = 248
	tTinyBlob   = 249
	tMediumBlob = 250
	tLongBlob   = 251
	tBlob       = 252
	tVarString  = 253
	tString     = 254
	tGeometry   = 255
)

const (
	flagUnsigned      = 32
	flagAutoIncrement = 512
	binaryCharset     = 63
	maxInt            = 2147483647
)

type field struct {
	typ      uint8
	flags    uint16
	length   uint32
	decimals uint8
	charset  uint16
}

func (f field) unsigned() bool { return f.flags&flagUnsigned != 0 }
func (f field) binary() bool   { return f.charset == binaryCharset }

func (f field) isDecimal() bool { return f.typ == tDecimal || f.typ == tNewDecimal }

// javaType — getColumnClassName.
func (f field) javaType() string {
	switch f.typ {
	case tTiny:
		if f.length == 1 && !f.unsigned() { // tinyint(1): tinyInt1isBit
			return "java.lang.Boolean"
		}
		return "java.lang.Integer"
	case tShort, tInt24:
		return "java.lang.Integer"
	case tLong:
		if f.unsigned() {
			return "java.lang.Long"
		}
		return "java.lang.Integer"
	case tLongLong:
		if f.unsigned() {
			return "java.math.BigInteger"
		}
		return "java.lang.Long"
	case tDecimal, tNewDecimal:
		return "java.math.BigDecimal"
	case tFloat:
		return "java.lang.Float"
	case tDouble:
		return "java.lang.Double"
	case tBit:
		return "java.lang.Boolean"
	case tDateTime, tDateTime2:
		return "java.time.LocalDateTime"
	case tDate, tNewDate, tTimestamp, tTimestamp2, tTime, tTime2, tYear:
		return "java.util.Date" // java.sql.Date/Time/Timestamp
	case tJSON:
		return "java.lang.String"
	case tGeometry:
		return "byte[]"
	case tVarChar, tVarString, tString, tEnum, tSet, tTinyBlob, tBlob, tMediumBlob, tLongBlob:
		if f.binary() {
			return "byte[]"
		}
		return "java.lang.String"
	}
	return "java.lang.Object" // NULL
}

// Точность целых у Connector/J — по типу, а не по длине поля (у литерала 1 — 19).
var intPrecision = map[uint8][2]int{ // signed, unsigned
	tTiny:     {3, 3},
	tShort:    {5, 5},
	tInt24:    {7, 8},
	tLong:     {10, 10},
	tLongLong: {19, 20},
}

// precision — getPrecision (и getColumnDisplaySize: у Connector/J они совпадают).
func (f field) precision(mbMaxLen func(uint16) int) int {
	length := int(f.length)
	if f.length > maxInt {
		length = maxInt
	}
	switch f.typ {
	case tTiny:
		if f.length == 1 && !f.unsigned() {
			return 1
		}
	case tDecimal, tNewDecimal:
		if !f.unsigned() {
			length--
		}
		if f.decimals > 0 {
			length--
		}
		return length
	case tFloat:
		return 12
	case tDouble:
		return 22
	case tNull:
		return 0
	}
	if p, ok := intPrecision[f.typ]; ok {
		if f.unsigned() {
			return p[1]
		}
		return p[0]
	}
	m := mbMaxLen(f.charset)
	if m <= 1 {
		return length
	}
	switch f.typ {
	case tTinyBlob, tBlob, tMediumBlob, tLongBlob: // TEXT: Connector/J делит длину дважды,
		if f.length <= maxInt { // если она не упёрлась в Integer.MAX_VALUE (longtext)
			length /= m
		}
	}
	return length / m
}

func (f field) scale() int {
	if f.isDecimal() {
		return int(f.decimals)
	}
	return 0
}

// Колонки таблиц: DatabaseMetaData.getColumns по information_schema.COLUMNS.

type column struct {
	dataType   string // DATA_TYPE
	columnType string // COLUMN_TYPE: "int unsigned", "tinyint(1)", ...
	charLen    *int64 // CHARACTER_MAXIMUM_LENGTH
	numPrec    *int64 // NUMERIC_PRECISION
	numScale   *int64 // NUMERIC_SCALE
	dtPrec     *int64 // DATETIME_PRECISION
}

// sqlType — DATA_TYPE (java.sql.Types).
func (c column) sqlType() int {
	switch c.dataType {
	case "tinyint":
		if strings.HasPrefix(c.columnType, "tinyint(1)") {
			return -7 // BIT
		}
		return -6
	case "smallint":
		return 5
	case "mediumint", "int", "integer":
		return 4
	case "bigint":
		return -5
	case "decimal", "numeric":
		return 3
	case "float":
		return 7
	case "double", "real":
		return 8
	case "bit", "boolean":
		return -7
	case "date", "year":
		return 91
	case "datetime", "timestamp":
		return 93
	case "time":
		return 92
	case "char", "enum", "set":
		return 1
	case "varchar", "tinytext":
		return 12
	case "text", "mediumtext", "longtext", "json":
		return -1
	case "binary", "geometry", "point", "linestring", "polygon", "multipoint", "multilinestring",
		"multipolygon", "geometrycollection", "geomcollection":
		return -2
	case "varbinary", "tinyblob":
		return -3
	case "blob", "mediumblob", "longblob":
		return -4
	}
	return 1111 // OTHER
}

func clamp(v *int64) int {
	if v == nil {
		return 0
	}
	if *v > maxInt {
		return maxInt
	}
	return int(*v)
}

// columnSize — COLUMN_SIZE.
func (c column) columnSize() int {
	fsp := clamp(c.dtPrec)
	frac := 0
	if fsp > 0 {
		frac = fsp + 1
	}
	switch c.dataType {
	case "tinyint":
		if strings.HasPrefix(c.columnType, "tinyint(1)") {
			return 1
		}
	case "date":
		return 10
	case "datetime", "timestamp":
		return 19 + frac
	case "time":
		return 8 + frac
	case "year":
		return 4
	case "json":
		return 1073741824
	case "geometry", "point", "linestring", "polygon", "multipoint", "multilinestring",
		"multipolygon", "geometrycollection", "geomcollection":
		return 65535
	}
	if c.numPrec != nil {
		return clamp(c.numPrec)
	}
	return clamp(c.charLen)
}

// decimalDigits — DECIMAL_DIGITS: только у decimal.
func (c column) decimalDigits() int {
	if c.dataType == "decimal" || c.dataType == "numeric" {
		return clamp(c.numScale)
	}
	return 0
}

// paramJavaType — тип параметра процедуры (information_schema.PARAMETERS).
func paramJavaType(dataType, dtd string) string {
	unsigned := strings.Contains(dtd, "unsigned")
	switch dataType {
	case "tinyint":
		if strings.HasPrefix(dtd, "tinyint(1)") {
			return "java.lang.Boolean"
		}
		return "java.lang.Integer"
	case "smallint", "mediumint":
		return "java.lang.Integer"
	case "int", "integer":
		if unsigned {
			return "java.lang.Long"
		}
		return "java.lang.Integer"
	case "bigint":
		if unsigned {
			return "java.math.BigInteger"
		}
		return "java.lang.Long"
	case "decimal", "numeric":
		return "java.math.BigDecimal"
	case "float":
		return "java.lang.Float"
	case "double", "real":
		return "java.lang.Double"
	case "bit", "boolean":
		return "java.lang.Boolean"
	case "datetime":
		return "java.time.LocalDateTime"
	case "date", "time", "timestamp", "year":
		return "java.util.Date"
	case "char", "varchar", "tinytext", "text", "mediumtext", "longtext", "enum", "set", "json":
		return "java.lang.String"
	case "binary", "varbinary", "tinyblob", "blob", "mediumblob", "longblob", "geometry":
		return "byte[]"
	}
	return "java.lang.Object"
}
