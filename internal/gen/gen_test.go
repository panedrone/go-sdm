package gen

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/panedrone/go-sdm/internal/cg"
	"github.com/panedrone/go-sdm/internal/meta/sqlite"
)

// TestGolden — полный цикл на SQLite в памяти: схема (только DDL — метаданным данные не
// нужны), sdm.xml/settings.xml/SQL реальных демо-проектов, генерация, побайтовое сравнение
// с тем, что на тех же проектах сгенерировал Java-плагин (sdm-plugins/harness, HarnessMain).
//
// Эталон переснимается в sdm-plugins: harness с этими проектами, затем вывод
// build/<out>/<label>/ копируется в testdata/golden/<label>/.
var goldenCases = []struct {
	label, project, sdm string
}{
	{"go_northwind_no_orm", "northwind", "no_orm/sdm"},
	{"go_northwind_gorm", "northwind", "gorm/sdm"},
	{"go_northwind_sqlx", "northwind", "sqlx/sdm"},
	{"go_log_no_orm", "log", "no_orm/sdm"},
	{"go_log_gorm", "log", "gorm/sdm"},
	{"go_log_sqlx", "log", "sqlx/sdm"},
	{"refcursors_go", "task_board", "refcursors"},
}

func TestGolden(t *testing.T) {
	for _, c := range goldenCases {
		t.Run(c.label, func(t *testing.T) {
			project := filepath.Join("testdata", c.project)
			runGolden(t, c.label, project, filepath.Join(project, c.sdm))
		})
	}
}

// runGolden — генерация на SQLite в памяти и сравнение с эталоном плагина.
func runGolden(t *testing.T, label, project, sdmDir string) {
	t.Helper()
	schema, err := os.ReadFile(filepath.Join(project, "schema.sql"))
	if err != nil {
		t.Fatal(err)
	}
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Exec(string(schema)); err != nil {
		t.Fatal(err)
	}
	res, err := Run(filepath.ToSlash(project), sdmDir, db)
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "golden", label)
	check(t, filepath.Join(golden, "dto"), res.DtoDir, res.Dto)
	check(t, filepath.Join(golden, "dao"), res.DaoDir, res.Dao)
}

func check(t *testing.T, dir, outputDir string, files []File) {
	t.Helper()
	want := map[string]bool{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		want[e.Name()] = true
	}
	expect(t, filepath.Join(dir, "_output_dir.txt"), outputDir)
	delete(want, "_output_dir.txt")
	for _, f := range files {
		if f.Err != nil {
			t.Errorf("%s: %v", f.Class, f.Err)
			continue
		}
		if !want[f.Name] {
			t.Errorf("%s: not in the golden output", f.Name)
			continue
		}
		delete(want, f.Name)
		expect(t, filepath.Join(dir, f.Name), f.Content)
	}
	var missing []string
	for n := range want {
		missing = append(missing, n)
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("not generated: %v", missing)
	}
}

func expect(t *testing.T, path, got string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Error(err)
		return
	}
	want := string(b)
	if got == want {
		return
	}
	gl, wl := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := 0; i < len(gl) || i < len(wl); i++ {
		var g, w string
		if i < len(gl) {
			g = gl[i]
		}
		if i < len(wl) {
			w = wl[i]
		}
		if g != w {
			t.Errorf("%s, line %d:\n got  %q\n want %q", path, i+1, g, w)
			return
		}
	}
}

// TestGoTypeNames — те же проекты, но <type detected> записан Go-именами (int32,
// time.Time, gorm-uuid...), как его пишет sdm init: вывод обязан не измениться.
func TestGoTypeNames(t *testing.T) {
	for _, c := range goldenCases {
		t.Run(c.label, func(t *testing.T) {
			project := filepath.Join("testdata", c.project)
			sdmDir := t.TempDir()
			src := filepath.Join(project, c.sdm)
			entries, err := os.ReadDir(src)
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".xml") {
					continue
				}
				b, err := os.ReadFile(filepath.Join(src, e.Name()))
				if err != nil {
					t.Fatal(err)
				}
				if e.Name() == "settings.xml" {
					b = []byte(goTypeNames(t, string(b)))
				}
				if err := os.WriteFile(filepath.Join(sdmDir, e.Name()), b, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			runGolden(t, c.label, project, sdmDir)
		})
	}
}

var (
	reComment = regexp.MustCompile(`(?s)<!--.*?-->`)
	reType    = regexp.MustCompile(`<type\s+detected="([^"]*)"\s+target="([^"]*)"\s*/>`)
)

// goTypeNames — Java-ключи type-map -> Go-имена. Несколько Java-типов дают одно Go-имя
// (java.util.Date и java.time.LocalDateTime -> time.Time): одинаковые цели схлопываются,
// разные — остаются Java-ключом (смешанный конфиг тоже должен работать).
func goTypeNames(t *testing.T, settings string) string {
	t.Helper()
	settings = reComment.ReplaceAllString(settings, "")
	seen := map[string]string{}
	renamed := 0
	res := reType.ReplaceAllStringFunc(settings, func(el string) string {
		m := reType.FindStringSubmatch(el)
		alias, ok := cg.GoAlias(m[1])
		if !ok {
			return el
		}
		if target, dup := seen[alias]; dup {
			if target == m[2] {
				return ""
			}
			return el
		}
		seen[alias] = m[2]
		renamed++
		return `<type detected="` + alias + `" target="` + m[2] + `"/>`
	})
	if renamed == 0 {
		t.Fatal("no Java names in the type-map")
	}
	return res
}
