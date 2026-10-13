package meta

import (
	"slices"
	"strconv"
)

// Cached remembers the answers of md for one generation run: the schema does not change
// meanwhile, and the same table or SQL is asked for many times (the model, its <crud>,
// the DTO and the DAO passes of the Java plug-in), each time a few round trips to a
// remote database. Errors are remembered too: the same question gets the same answer.
// Every call returns its own copy of the slices.
func Cached(md Metadata) Metadata {
	return &cached{
		Metadata: md,
		columns:  map[string]result[[]Column]{},
		tables:   map[string]result[*Table]{},
		pks:      map[string]result[[]string]{},
		params:   map[string]result[*Params]{},
	}
}

type result[T any] struct {
	v   T
	err error
}

type cached struct {
	Metadata // RawOfJava and Close as is
	columns  map[string]result[[]Column]
	tables   map[string]result[*Table]
	pks      map[string]result[[]string]
	params   map[string]result[*Params]
}

func remember[T any](m map[string]result[T], key string, f func() (T, error)) (T, error) {
	r, ok := m[key]
	if !ok {
		r.v, r.err = f()
		m[key] = r
	}
	return r.v, r.err
}

func (c *cached) SQLColumns(jdbcSQL string) ([]Column, error) {
	v, err := remember(c.columns, jdbcSQL, func() ([]Column, error) { return c.Metadata.SQLColumns(jdbcSQL) })
	return slices.Clone(v), err
}

func (c *cached) Table(tableRef string) (*Table, error) {
	v, err := remember(c.tables, tableRef, func() (*Table, error) { return c.Metadata.Table(tableRef) })
	if v == nil {
		return nil, err
	}
	return &Table{Columns: slices.Clone(v.Columns), TableColumns: slices.Clone(v.TableColumns)}, err
}

func (c *cached) PKColNames(tableRef string) ([]string, error) {
	v, err := remember(c.pks, tableRef, func() ([]string, error) { return c.Metadata.PKColNames(tableRef) })
	return slices.Clone(v), err
}

func (c *cached) SQLParams(jdbcSQL string, typeCount int) (*Params, error) {
	key := strconv.Itoa(typeCount) + "\x00" + jdbcSQL
	v, err := remember(c.params, key, func() (*Params, error) { return c.Metadata.SQLParams(jdbcSQL, typeCount) })
	if v == nil {
		return nil, err
	}
	p := *v
	p.JavaTypes, p.RawTypes = slices.Clone(v.JavaTypes), slices.Clone(v.RawTypes)
	return &p, err
}
