package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/panedrone/go-sdm/internal/meta/sqlite"
)

// sdm без команды: в пустом проекте — init, во втором запуске — generate.
// t.Chdir — без t.Parallel.
func TestAuto(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("go.mod", []byte("module example.com/app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := run(nil, &out, &errOut); code != 0 {
		t.Fatalf("init: %d %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "created: ") || !strings.Contains(out.String(), "next: ") {
		t.Errorf("init output: %s", out.String())
	}

	// база для генерации: SQLite-файл с таблицей, settings.xml — на неё
	db, err := sqlite.Open("app.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("create table tasks (t_id integer primary key autoincrement, t_subject text not null)"); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	settings := filepath.Join("go-sdm", "settings.xml")
	b, err := os.ReadFile(settings)
	if err != nil {
		t.Fatal(err)
	}
	s := strings.Replace(string(b), `url="postgres://localhost:5432/postgres"`, `url="sqlite:app.sqlite"`, 1)
	if err := os.WriteFile(settings, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
	sdmXML := `<sdm><dto-class name="Task" ref="tasks"/><dao-class name="TasksDao"><crud dto="Task"/></dao-class></sdm>`
	if err := os.WriteFile(filepath.Join("go-sdm", "sdm.xml"), []byte(sdmXML), 0o644); err != nil {
		t.Fatal(err)
	}

	out.Reset()
	errOut.Reset()
	if code := run(nil, &out, &errOut); code != 0 {
		t.Fatalf("generate: %d %s", code, errOut.String())
	}
	for _, f := range []string{"internal/dbal/models/task.go", "internal/dbal/tasks_dao.go"} {
		if !strings.Contains(out.String(), "created: "+f) {
			t.Errorf("%s not generated: %s", f, out.String())
		}
	}
	out.Reset()
	if code := run([]string{"check"}, &out, &errOut); code != 0 {
		t.Errorf("check after generate: %d %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "up to date: 2 classes") {
		t.Errorf("check output: %s", out.String())
	}
	if _, err := os.Stat(filepath.Join("go-sdm", "sdm.xsd")); err != nil {
		t.Error(err)
	}
}

// init в готовом проекте: ошибка, ничего не тронуто.
func TestInitExisting(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("go.mod", []byte("module m\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"init"}, &out, &errOut); code != 0 {
		t.Fatalf("init: %d %s", code, errOut.String())
	}
	sdmXML := filepath.Join("go-sdm", "sdm.xml")
	if err := os.WriteFile(sdmXML, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errOut.Reset()
	if code := run([]string{"init"}, &out, &errOut); code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "already exists") || out.Len() != 0 {
		t.Errorf("stdout %q, stderr %q", out.String(), errOut.String())
	}
	if b, _ := os.ReadFile(sdmXML); string(b) != "mine" {
		t.Error("sdm.xml overwritten")
	}
}

// строка состояния не показывает логин и пароль
func TestDbLabel(t *testing.T) {
	for url, want := range map[string]string{
		"jdbc:postgresql://localhost:5432/my_tests":  "localhost:5432/my_tests",
		"postgres://u:secret@db.host:5432/app":       "db.host:5432/app",
		"jdbc:mysql://h:3306/orders?user=u&pwd=p":    "h:3306/orders",
		"jdbc:sqlite:$PROJECT_DIR$/todolist.sqlite3": "$PROJECT_DIR$/todolist.sqlite3",
	} {
		if got := dbLabel(url); got != want {
			t.Errorf("%s: %q", url, got)
		}
	}
}

// без <go-db> — ошибка с подсказкой: <jdbc> (он в settings.xml есть) go-sdm не читает
func TestNoGoDB(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile("go.mod", []byte("module example.com/app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"init", "-db", "sqlite"}, &out, &errOut); code != 0 {
		t.Fatalf("init: %d %s", code, errOut.String())
	}
	settings := filepath.Join("go-sdm", "settings.xml")
	b, err := os.ReadFile(settings)
	if err != nil {
		t.Fatal(err)
	}
	s := strings.Replace(string(b), `<go-db url="sqlite:app.sqlite3" user="" pwd=""/>`, "", 1) // остаётся только <jdbc>
	if s == string(b) {
		t.Fatal("<go-db> not found in settings.xml")
	}
	if err := os.WriteFile(settings, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
	errOut.Reset()
	if code := run([]string{"generate"}, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "<go-db") {
		t.Errorf("generate without <go-db>: %d %q", code, errOut.String())
	}
}
