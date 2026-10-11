// Package skeleton — то, что `sdm init` выкладывает в проект.
//
// Init ничего не перезаписывает: если хоть один из его файлов уже есть, это ошибка,
// и не создаётся ничего (запуск по ошибке в готовом проекте безвреден). XSD принадлежат
// версии sdm, их обновляет RefreshXSD при каждой генерации.
//
// data_store_no_orm.go.txt — копия src/resources/data_store_no_orm.go из sdm-plugins;
// при правке шаблона там — обновить и здесь.
package skeleton

import (
	"bufio"
	"bytes"
	"embed"
	"errors"
	"fmt"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//go:embed files
var files embed.FS

type Options struct {
	ProjectDir string // папка go.mod
	SdmDir     string // относительно ProjectDir, с '/'
	DaoDir     string // относительно ProjectDir, с '/'; модели — в DaoDir/models
}

var xsds = []string{"sdm.xsd", "dao.xsd", "settings.xsd"}

// Exists — в папке уже есть конфигурация (sdm.xml или settings.xml).
func Exists(projectDir, sdmDir string) bool {
	dir := filepath.Join(projectDir, filepath.FromSlash(sdmDir))
	for _, n := range []string{"sdm.xml", "settings.xml"} {
		if _, err := os.Stat(filepath.Join(dir, n)); err == nil {
			return true
		}
	}
	return false
}

// Init создаёт конфигурацию и data store. Возвращает созданные файлы.
func Init(o Options) ([]string, error) {
	module, err := readModule(filepath.Join(o.ProjectDir, "go.mod"))
	if err != nil {
		return nil, err
	}
	pkg := path.Base(o.DaoDir)
	if !token.IsIdentifier(pkg) {
		return nil, fmt.Errorf("the last element of the DAO folder must be a Go package name: %q", o.DaoDir)
	}
	vars := strings.NewReplacer(
		"{{module}}", module,
		"{{sql}}", path.Join(o.SdmDir, "sql"),
		"{{dto}}", path.Join(o.DaoDir, "models"),
		"{{dao}}", o.DaoDir,
		"{{package}}", pkg,
	)
	sdmDir := filepath.Join(o.ProjectDir, filepath.FromSlash(o.SdmDir))
	daoDir := filepath.Join(o.ProjectDir, filepath.FromSlash(o.DaoDir))
	type step struct{ src, dst string }
	steps := []step{
		{"sdm.xml", filepath.Join(sdmDir, "sdm.xml")},
		{"settings.xml", filepath.Join(sdmDir, "settings.xml")},
		{"data_store_no_orm.go.txt", filepath.Join(daoDir, "data_store_no_orm.go")},
		{"data_store_no_orm_ex.go.txt", filepath.Join(daoDir, "data_store_no_orm_ex.go")},
	}
	for _, x := range xsds {
		steps = append(steps, step{x, filepath.Join(sdmDir, x)})
	}
	// сначала проверка всего: либо создаётся всё, либо ничего
	var existing []string
	for _, s := range steps {
		if _, err := os.Stat(s.dst); err == nil {
			existing = append(existing, s.dst)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	if len(existing) > 0 {
		return nil, fmt.Errorf("already exists, nothing is written: %s", strings.Join(existing, ", "))
	}
	for _, dir := range []string{filepath.Join(sdmDir, "sql"), daoDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	var written []string
	for _, s := range steps {
		data, err := files.ReadFile("files/" + s.src)
		if err != nil {
			return written, err
		}
		if err := os.WriteFile(s.dst, []byte(vars.Replace(string(data))), 0o644); err != nil {
			return written, err
		}
		written = append(written, s.dst)
	}
	return written, nil
}

// RefreshXSD — XSD этой версии sdm: подсказки в IDE должны соответствовать генератору.
// Возвращает обновлённые (созданные) файлы; совпадающие не трогает.
func RefreshXSD(projectDir, sdmDir string) ([]string, error) {
	dir := filepath.Join(projectDir, filepath.FromSlash(sdmDir))
	var res []string
	for _, x := range xsds {
		want, err := files.ReadFile("files/" + x)
		if err != nil {
			return res, err
		}
		dst := filepath.Join(dir, x)
		if got, err := os.ReadFile(dst); err == nil && bytes.Equal(got, want) {
			continue
		}
		if err := os.WriteFile(dst, want, 0o644); err != nil {
			return res, err
		}
		res = append(res, dst)
	}
	return res, nil
}

// readModule — значение "module" из go.mod.
func readModule(goMod string) (string, error) {
	data, err := os.ReadFile(goMod)
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("%s not found: run sdm in the folder of go.mod (or 'go mod init' first)", goMod)
	}
	if err != nil {
		return "", err
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) >= 2 && f[0] == "module" {
			return strings.Trim(f[1], `"`), nil
		}
	}
	return "", fmt.Errorf("%s: 'module' not found", goMod)
}
