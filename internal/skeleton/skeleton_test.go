package skeleton

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/panedrone/go-sdm/internal/sdmxml"
)

func newProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestInit(t *testing.T) {
	dir := newProject(t)
	o := Options{ProjectDir: dir, SdmDir: "go-sdm", DaoDir: "internal/dbal"}
	if Exists(dir, "go-sdm") {
		t.Fatal("Exists before init")
	}
	written, err := Init(o)
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 7 || !Exists(dir, "go-sdm") {
		t.Fatalf("written %d, exists %t", len(written), Exists(dir, "go-sdm"))
	}

	// settings.xml разбирается и указывает туда, куда выложен скелет
	s, err := sdmxml.LoadSettings(filepath.Join(dir, "go-sdm"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Folders.Target != "example.com/app" || s.Folders.SQL != "go-sdm/sql" ||
		s.Dto.Scope != "internal/dbal/models" || s.Dao.Scope != "internal/dbal" {
		t.Errorf("settings: %+v / %+v / %+v", s.Folders, s.Dto, s.Dao)
	}
	if _, err := sdmxml.LoadSdm(filepath.Join(dir, "go-sdm")); err != nil {
		t.Error(err)
	}
	ds, err := os.ReadFile(filepath.Join(dir, "internal", "dbal", "data_store_no_orm.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(ds), "package dbal\n") {
		t.Errorf("package: %q", strings.SplitN(string(ds), "\n", 2)[0])
	}
}

// Повторный init — ошибка, и не тронуто ничего, даже XSD.
func TestInitTwice(t *testing.T) {
	dir := newProject(t)
	o := Options{ProjectDir: dir, SdmDir: "go-sdm", DaoDir: "dbal"}
	if _, err := Init(o); err != nil {
		t.Fatal(err)
	}
	userXML := filepath.Join(dir, "go-sdm", "sdm.xml")
	xsd := filepath.Join(dir, "go-sdm", "sdm.xsd")
	for _, f := range []string{userXML, xsd} {
		if err := os.WriteFile(f, []byte("mine"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	written, err := Init(o)
	if err == nil || !strings.Contains(err.Error(), "already exists") || len(written) != 0 {
		t.Fatalf("error expected, got %v (written %d)", err, len(written))
	}
	for _, f := range []string{userXML, xsd} {
		if b, _ := os.ReadFile(f); string(b) != "mine" {
			t.Errorf("%s overwritten", f)
		}
	}
}

// Init при чужом data store не создаёт и конфигурацию: всё или ничего.
func TestInitAllOrNothing(t *testing.T) {
	dir := newProject(t)
	ds := filepath.Join(dir, "dbal", "data_store_no_orm.go")
	if err := os.MkdirAll(filepath.Dir(ds), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ds, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(Options{ProjectDir: dir, SdmDir: "go-sdm", DaoDir: "dbal"}); err == nil {
		t.Fatal("error expected")
	}
	if Exists(dir, "go-sdm") {
		t.Error("configuration created in spite of the error")
	}
}

func TestRefreshXSD(t *testing.T) {
	dir := newProject(t)
	if _, err := Init(Options{ProjectDir: dir, SdmDir: "go-sdm", DaoDir: "dbal"}); err != nil {
		t.Fatal(err)
	}
	if res, err := RefreshXSD(dir, "go-sdm"); err != nil || len(res) != 0 {
		t.Fatalf("fresh XSD rewritten: %v %v", res, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go-sdm", "dao.xsd"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if res, err := RefreshXSD(dir, "go-sdm"); err != nil || len(res) != 1 {
		t.Fatalf("outdated XSD: %v %v", res, err)
	}
}

func TestInitNoGoMod(t *testing.T) {
	_, err := Init(Options{ProjectDir: t.TempDir(), SdmDir: "go-sdm", DaoDir: "dbal"})
	if err == nil || !strings.Contains(err.Error(), "go.mod") {
		t.Fatalf("go.mod error expected, got %v", err)
	}
}

func TestInitBadPackage(t *testing.T) {
	if _, err := Init(Options{ProjectDir: newProject(t), SdmDir: "go-sdm", DaoDir: "my-dbal"}); err == nil {
		t.Fatal("error expected for 'my-dbal'")
	}
}
