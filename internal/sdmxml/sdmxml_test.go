package sdmxml

import (
	"strings"
	"testing"
)

func TestLoadSdm(t *testing.T) {
	s, err := LoadSdm("testdata/ok")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.DtoClasses) != 2 || len(s.DaoClasses) != 2 {
		t.Fatalf("dto: %d, dao: %d", len(s.DtoClasses), len(s.DaoClasses))
	}

	var order []string
	for _, c := range s.Classes {
		order = append(order, c.Name+map[bool]string{false: ":dto", true: ":dao"}[c.Dao])
	}
	if got := strings.Join(order, ","); got != "Test:dto,Test1:dto,InlineDao:dao,SpDao:dao" {
		t.Errorf("classes: %s", got)
	}

	// умолчания XSD и явные значения, в том числе пустые
	d0, d1 := s.DtoClasses[0], s.DtoClasses[1]
	if d0.PK != "*" || d0.Auto != "*" {
		t.Errorf("defaults: pk=%q auto=%q", d0.PK, d0.Auto)
	}
	if d1.PK != "t_id" || d1.Auto != "" || d1.Header == nil || *d1.Header != "// header" || len(d1.Fields) != 1 || d1.Fields[0].Column != "sign1" {
		t.Errorf("explicit: %+v", d1)
	}

	// порядок элементов разных типов сохраняется
	inline := s.DaoClasses[0]
	var kinds []string
	for _, it := range inline.Items {
		kinds = append(kinds, it.Kind())
	}
	if got := strings.Join(kinds, ","); got != "query-dto-list,crud,exec-dml,crud" {
		t.Errorf("order: %s", got)
	}
	crud := inline.Items[1]
	if crud.Table != "tests" || crud.FetchGenerated || crud.Create != nil || crud.ReadAll == nil ||
		crud.ReadAll.Method != "read_all" || crud.Update == nil || crud.Update.Method != "" {
		t.Errorf("crud: %+v", crud)
	}
	if c := inline.Items[3]; c.Table != "*" || !c.FetchGenerated {
		t.Errorf("crud defaults: table=%q fetch-generated=%t", c.Table, c.FetchGenerated)
	}
	if !inline.Items[2].ExternalSQL || inline.Items[0].ExternalSQL {
		t.Error("external-sql")
	}

	// внешний файл подтянут, порядок как в файле
	sp := s.DaoClasses[1]
	if sp.Ref != "SpDao.xml" || len(sp.Items) == 0 {
		t.Fatalf("SpDao.xml not loaded: %+v", sp)
	}
	first, last := sp.Items[0], sp.Items[len(sp.Items)-1]
	if first.Kind() != "query" || first.Method != "get_test_rating(t_id)" || first.ReturnType != "java.math.BigDecimal" {
		t.Errorf("first: %+v", first)
	}
	if last.Kind() != "exec-dml" || last.Method != "set_test_rating(t_id, t_rating)" {
		t.Errorf("last: %+v", last)
	}
}

func TestLoadSdmUnknownElement(t *testing.T) {
	_, err := LoadSdm("testdata/bad")
	if err == nil || !strings.Contains(err.Error(), "<querry>") {
		t.Fatalf("error about <querry> expected, got %v", err)
	}
}

func TestLoadSettings(t *testing.T) {
	s, err := LoadSettings("testdata/ok")
	if err != nil {
		t.Fatal(err)
	}
	if s.Folders.SQL != "sqlx/sdm/sql" || s.Dao.FieldAssignMode != 3 || s.Jdbc.URL != "jdbc:postgresql://localhost/orders" {
		t.Errorf("%+v", s)
	}
	if len(s.TypeMap.Types) == 0 || s.TypeMap.Types[0].Detected != "java.util.UUID" {
		t.Errorf("type-map: %+v", s.TypeMap)
	}
	var velocity, value int
	for _, m := range s.Macros {
		if m.IsVelocity() {
			velocity++
		} else if m.Value != nil && *m.Value != "" {
			value++
		}
	}
	if velocity != 1 || value == 0 { // ${ora-num} — единственный <vm-xml> в пресете
		t.Errorf("macros: velocity=%d value=%d", velocity, value)
	}
}
