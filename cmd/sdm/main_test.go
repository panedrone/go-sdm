package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/panedrone/go-sdm/internal/meta/sqlite"
)

// init, затем gen и val. t.Chdir — без t.Parallel.
func TestInitGenerate(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("go.mod", []byte("module example.com/app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"init"}, &out, &errOut); code != 0 {
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
	if code := run([]string{"gen"}, &out, &errOut); code != 0 {
		t.Fatalf("generate: %d %s", code, errOut.String())
	}
	for _, f := range []string{"internal/dbal/models/task.go", "internal/dbal/tasks_dao.go"} {
		if !strings.Contains(out.String(), "created: "+f) {
			t.Errorf("%s not generated: %s", f, out.String())
		}
	}
	out.Reset()
	if code := run([]string{"val"}, &out, &errOut); code != 0 {
		t.Errorf("validate after generate: %d %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "up to date: 2 classes") {
		t.Errorf("validate output: %s", out.String())
	}
	if _, err := os.Stat(filepath.Join("go-sdm", "sdm.xsd")); err != nil {
		t.Error(err)
	}
}

// ls, выборочные gen и val: номера по порядку объявления, имена, запятые, флаги после классов.
func TestListAndSelect(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("go.mod", []byte("module example.com/app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"init", "-db", "sqlite"}, &out, &errOut); code != 0 {
		t.Fatalf("init: %d %s", code, errOut.String())
	}
	db, err := sqlite.Open("app.sqlite3")
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		"create table tasks (t_id integer primary key autoincrement, t_subject text not null)",
		"create table groups (g_id integer primary key autoincrement, g_name text not null)",
	} {
		if err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	_ = db.Close()
	// модели и DAO вперемешку: номера — в порядке объявления
	sdmXML := `<sdm>
<dto-class name="Task" ref="tasks"/>
<dao-class name="TasksDao"><crud dto="Task"/></dao-class>
<dto-class name="Group" ref="groups"/>
<dao-class name="GroupsDao"><crud dto="Group"/></dao-class>
</sdm>`
	if err := os.WriteFile(filepath.Join("go-sdm", "sdm.xml"), []byte(sdmXML), 0o644); err != nil {
		t.Fatal(err)
	}

	ls := func(args ...string) string {
		t.Helper()
		out.Reset()
		errOut.Reset()
		if code := run(append([]string{"ls"}, args...), &out, &errOut); code != 0 {
			t.Fatalf("ls %v: %d %s", args, code, errOut.String())
		}
		return strings.TrimSpace(out.String())
	}
	for args, want := range map[string]string{
		"":       "1:Task  2:TasksDao  3:Group  4:GroupsDao",
		"-m":     "1:Task  3:Group",
		"-d":     "2:TasksDao  4:GroupsDao",
		"-m -d":  "1:Task  2:TasksDao  3:Group  4:GroupsDao",
		"4,1":    "1:Task  4:GroupsDao",
		"Group":  "3:Group",
		"-d 1,2": "2:TasksDao",
	} {
		if got := ls(strings.Fields(args)...); got != want {
			t.Errorf("ls %s: %q, want %q", args, got, want)
		}
	}

	out.Reset()
	if code := run([]string{"gen", "1,2", "-dir", "go-sdm"}, &out, &errOut); code != 0 {
		t.Fatalf("gen 1,2: %d %s", code, errOut.String())
	}
	if got := out.String(); !strings.Contains(got, "created: internal/dbal/models/task.go") ||
		!strings.Contains(got, "created: internal/dbal/tasks_dao.go") || strings.Contains(got, "group") {
		t.Errorf("gen 1,2: %s", got)
	}
	out.Reset()
	if code := run([]string{"val", "Task", "2"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "up to date: 2 classes") {
		t.Errorf("val Task 2: %d %s %s", code, out.String(), errOut.String())
	}
	out.Reset()
	if code := run([]string{"validate"}, &out, &errOut); code != 1 || !strings.Contains(out.String(), "missing: internal/dbal/groups_dao.go") {
		t.Errorf("validate: %d %s", code, out.String())
	}

	table := ls("-l")
	lines := strings.Split(table, "\n")
	if len(lines) != 5 || !strings.HasPrefix(lines[0], "#") {
		t.Fatalf("ls -l:\n%s", table)
	}
	if f := strings.Fields(lines[1]); len(f) != 6 || f[0] != "1" || f[1] != "model" || f[2] != "Task" || f[3] != "internal/dbal/models/task.go" {
		t.Errorf("ls -l, generated: %q", lines[1])
	}
	if f := strings.Fields(lines[4]); len(f) != 5 || f[1] != "dao" || f[4] != "-" {
		t.Errorf("ls -l, not generated: %q", lines[4])
	}

	// -m/-d у gen и val: модели отдельно от DAO
	if err := os.Remove(filepath.Join("internal", "dbal", "models", "task.go")); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if code := run([]string{"val", "-d", "1,2"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "up to date: 1 classes") {
		t.Errorf("val -d 1,2: %d %s %s", code, out.String(), errOut.String())
	}
	out.Reset()
	if code := run([]string{"gen", "-m"}, &out, &errOut); code != 0 ||
		strings.TrimSpace(out.String()) != "created: internal/dbal/models/task.go\ncreated: internal/dbal/models/group.go" {
		t.Errorf("gen -m: %d %q %s", code, out.String(), errOut.String())
	}

	for _, args := range [][]string{{"gen", "5"}, {"val", "Nope"}, {"ls", "0"}} {
		errOut.Reset()
		if code := run(args, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "no class") {
			t.Errorf("%v: %d %q", args, code, errOut.String())
		}
	}
}

// без команды — подсказка и код 2, ни init, ни генерации
func TestNoCommand(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, args := range [][]string{nil, {"-db", "sqlite"}} {
		var out, errOut bytes.Buffer
		if code := run(args, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "usage:") {
			t.Errorf("%q: %d %q", args, code, errOut.String())
		}
		if entries, _ := os.ReadDir("."); len(entries) != 0 {
			t.Errorf("%q: files created", args)
		}
	}
}

func TestVersion(t *testing.T) {
	for _, a := range []string{"version", "-v", "--version"} {
		var out, errOut bytes.Buffer
		if code := run([]string{a}, &out, &errOut); code != 0 || strings.TrimSpace(out.String()) != version {
			t.Errorf("%s: %d %q %q", a, code, out.String(), errOut.String())
		}
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
