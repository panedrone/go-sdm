package cg

import (
	"fmt"
	"strings"

	"github.com/panedrone/go-sdm/internal/meta"
	"github.com/panedrone/go-sdm/internal/sdmxml"
)

// Core — то, что Java-плагин держит в JdbcUtils на один прогон генерации:
// метаданные, type-map, макросы и режим имён полей DTO.
type Core struct {
	md        meta.Metadata
	tm        *TypeMap
	mc        *Macros
	dtoMode   FieldNamesMode // имена полей DTO
	paramMode FieldNamesMode // имена параметров методов exec-dml
	sqlRoot   string
	crud      map[string]*tableFields // read/update/delete одного <crud> спрашивают одну таблицу
}

func NewCore(md meta.Metadata, s *sdmxml.Settings, dtoMode, paramMode FieldNamesMode, sqlRoot string) *Core {
	return &Core{
		md:        md,
		tm:        NewTypeMap(s.TypeMap),
		mc:        NewMacros(s.Macros),
		dtoMode:   dtoMode,
		paramMode: paramMode,
		sqlRoot:   sqlRoot,
		crud:      map[string]*tableFields{},
	}
}

func (c *Core) DtoFieldNamesMode() FieldNamesMode { return c.dtoMode }
func (c *Core) SQLRoot() string                   { return c.sqlRoot }

// TargetTypeByTypeMap — тип без функции присваивания.
func (c *Core) TargetTypeByTypeMap(detected string) string {
	t, _, _ := SplitTargetType(c.tm.Target(detected))
	return t
}

// DtoFields — поля DTO-класса.
func (c *Core) DtoFields(dto *sdmxml.DtoClass) ([]FieldInfo, error) {
	return c.dtoFields(false, dto)
}

func (c *Core) dtoFields(ignoreModel bool, dto *sdmxml.DtoClass) ([]FieldInfo, error) {
	fields, err := c.fieldsByRef(ignoreModel, dto)
	if err == nil {
		fields, err = c.refineByFieldXML(dto, fields)
	}
	if err != nil {
		return nil, fmt.Errorf("<dto-class name=\"%s\"... java.lang.Exception: %s", dto.Name, err.Error())
	}
	res := make([]FieldInfo, len(fields))
	for i, fi := range fields {
		res[i] = fi.WithScalarType(c.tm.Target(fi.scalarType))
	}
	return res, nil
}

func (c *Core) fieldsByRef(ignoreModel bool, dto *sdmxml.DtoClass) ([]FieldInfo, error) {
	model := ""
	if !ignoreModel {
		model = Model(dto.Name)
	}
	if model != "" {
		model += "-" // префикс имён типов: "sa-int"
	}
	ref := dto.Ref
	if IsEmptyRef(ref) {
		return nil, nil
	}
	jdbcSQL, err := JdbcSQLByDtoClassRef(ref, c.sqlRoot)
	if err != nil {
		return nil, err
	}
	// сначала поля из SQL: невидимые и специфичные для базы колонки не попадут
	cols, err := c.md.SQLColumns(jdbcSQL)
	if err != nil {
		return nil, err
	}
	sqlFields, err := columnFields(cols, model, c.dtoMode, dto.PK)
	if err != nil {
		return nil, err
	}
	var table string
	switch {
	case IsTableRef(ref):
		table = ref
	case IsSQLShortcutRef(ref):
		shc, err := parseSQLShortcutRef(ref)
		if err != nil {
			return nil, err
		}
		table = shc.tableName
	default:
		return sqlFields, nil
	}
	// колонки таблицы, которые возвращает SQL, но с метаданными таблицы
	sqlCols := map[string]bool{}
	for _, n := range columnNames(sqlFields) {
		sqlCols[n] = true
	}
	t, err := readTableFields(c.md, c.tm, c.dtoMode, model, table, "*", dto.Auto)
	if err != nil {
		return nil, err
	}
	var res []FieldInfo
	for _, fi := range t.all {
		if sqlCols[fi.columnName] {
			res = append(res, fi)
		}
	}
	return res, nil
}

func (c *Core) refineByFieldXML(dto *sdmxml.DtoClass, fields []FieldInfo) ([]FieldInfo, error) {
	res := append([]FieldInfo(nil), fields...)
	// колонка уточняется там, где встретилась последней (если SQL вернул её дважды)
	index := map[string]int{}
	for i, fi := range res {
		index[fi.columnName] = i
	}
	refined := map[string]bool{}
	excluded := map[int]bool{}
	for _, xf := range dto.Fields {
		xt := xf.Type
		i, ok := index[xf.Column]
		if !ok { // объявлено в XML, но SQL такого не вернул
			if c.tm.Unmapped(xt) {
				xt = meta.JavaObject // Java-имя без ключа и без метаданных: unknown
			}
			fi, err := NewFieldInfo(c.dtoMode, xt, xf.Column, "xml("+xf.Column+")")
			if err != nil {
				return nil, err
			}
			res = append(res, fi)
			i = len(res) - 1
			index[xf.Column] = i
		}
		if strings.TrimSpace(xf.Type) == "-" {
			excluded[i] = true
		}
		if ok && c.tm.Unmapped(xt) {
			xt = res[i].originalType // Java-имя без ключа: тип колонки из метаданных
		}
		fi, err := c.refineByTypeMapAndMacros(res[i], xt)
		if err != nil {
			return nil, err
		}
		res[i] = fi.WithScalarType(xt)
		refined[xf.Column] = true
	}
	var out []FieldInfo
	for i, fi := range res {
		if excluded[i] {
			continue
		}
		if !refined[fi.columnName] {
			var err error
			// именно Type(), а не OriginalType()
			if fi, err = c.refineByTypeMapAndMacros(fi, strings.TrimSpace(fi.renderedType)); err != nil {
				return nil, err
			}
		}
		out = append(out, fi)
	}
	return out, nil
}

func (c *Core) refineByTypeMapAndMacros(fi FieldInfo, typeName string) (FieldInfo, error) {
	typeName = strings.TrimSpace(typeName)
	var target string
	var err error
	switch start := strings.IndexByte(typeName, '|'); {
	case start == -1: // только тип: "int64"
		mapped := c.tm.Target(typeName)
		if typeName == strings.TrimSpace(fi.originalType) { // первый проход: тип из метаданных
			mapped = c.tm.TargetRaw(typeName, fi.rawKey(""))
		}
		target, err = c.mc.ParseTargetType(mapped, fi)
	case start == 0: // "|0: hello0|1:hello1" — к типу из метаданных
		target, err = c.mc.ParseTargetType(c.tm.TargetRaw(fi.originalType, fi.rawKey(""))+typeName, fi)
	default: // "{type}|0:Integer|1:hello1"
		target, err = c.mc.ParseTargetType(typeName, fi)
	}
	if err != nil {
		return fi, err
	}
	return fi.WithRenderedType(target), nil
}
