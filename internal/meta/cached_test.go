package meta

import (
	"errors"
	"testing"
)

type countingMD struct {
	calls int
}

func (c *countingMD) SQLColumns(string) ([]Column, error) {
	c.calls++
	return []Column{{Name: "id"}}, nil
}

func (c *countingMD) Table(ref string) (*Table, error) {
	c.calls++
	if ref == "missing" {
		return nil, errors.New("Not found: 'missing'")
	}
	return &Table{Columns: []Column{{Name: "id"}}, TableColumns: []TableColumn{{Name: "id"}}}, nil
}

func (c *countingMD) PKColNames(string) ([]string, error) {
	c.calls++
	return []string{"id"}, nil
}

func (c *countingMD) SQLParams(_ string, n int) (*Params, error) {
	c.calls++
	return &Params{Count: n, JavaTypes: make([]string, n), RawTypes: make([]string, n)}, nil
}

func (c *countingMD) RawOfJava(string) string { return "" }
func (c *countingMD) Close() error            { return nil }

func TestCached(t *testing.T) {
	md := &countingMD{}
	c := Cached(md)
	for range 3 {
		cols, _ := c.SQLColumns("select 1")
		cols[0].Name = "changed" // a caller's change does not reach the cache
		tbl, _ := c.Table("t")
		tbl.Columns[0].Name = "changed"
		tbl.TableColumns[0].Name = "changed"
		pk, _ := c.PKColNames("t")
		pk[0] = "changed"
		p, _ := c.SQLParams("q", 2)
		p.JavaTypes[0] = "changed"
		if _, err := c.Table("missing"); err == nil {
			t.Fatal("the error is not remembered")
		}
	}
	if md.calls != 5 {
		t.Errorf("%d calls to the database, want 5 (one per question)", md.calls)
	}
	if p, _ := c.SQLParams("q", 3); p.Count != 3 || md.calls != 6 {
		t.Errorf("another typeCount is another question: %+v, %d calls", p, md.calls)
	}
	cols, _ := c.SQLColumns("select 1")
	tbl, _ := c.Table("t")
	pk, _ := c.PKColNames("t")
	p, _ := c.SQLParams("q", 2)
	if cols[0].Name != "id" || tbl.Columns[0].Name != "id" || tbl.TableColumns[0].Name != "id" ||
		pk[0] != "id" || p.JavaTypes[0] != "" {
		t.Error("the cache was changed through a returned slice")
	}
}
