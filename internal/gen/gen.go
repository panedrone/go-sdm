// Package gen — прогон генерации для одного проекта: settings.xml и sdm.xml из папки sdm,
// метаданные — из базы <jdbc url>, результат — тексты файлов DTO и DAO.
package gen

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/panedrone/go-sdm/internal/cg"
	"github.com/panedrone/go-sdm/internal/gogen"
	"github.com/panedrone/go-sdm/internal/meta"
	"github.com/panedrone/go-sdm/internal/meta/mysql"
	"github.com/panedrone/go-sdm/internal/meta/pg"
	"github.com/panedrone/go-sdm/internal/meta/sqlite"
	"github.com/panedrone/go-sdm/internal/sdmxml"
)

type File struct {
	Class   string
	Name    string // имя файла
	Content string
	Err     error // класс не сгенерирован
}

type Result struct {
	DtoDir, DaoDir string // относительно проекта, как scope в settings.xml
	Dto, Dao       []File
}

// OpenMetadata — база из <go-db url user pwd> (или <jdbc> — у cli/genharness). $PROJECT_DIR$ и ${project_loc} — папка проекта,
// относительный путь SQLite — тоже от неё.
func OpenMetadata(projectDir string, j sdmxml.DB) (meta.Metadata, error) {
	url := strings.NewReplacer("$PROJECT_DIR$", projectDir, "${project_loc}", projectDir).Replace(strings.TrimSpace(j.URL))
	if file, ok := strings.CutPrefix(strings.TrimPrefix(url, "jdbc:"), "sqlite:"); ok {
		if file == ":memory:" {
			return nil, errors.New("sqlite::memory: has no tables to generate from")
		}
		if !filepath.IsAbs(file) {
			file = filepath.Join(projectDir, file)
		}
		if _, err := os.Stat(file); err != nil {
			return nil, fmt.Errorf("SQLite database not found: %s (<jdbc url> in settings.xml)", file)
		}
		return sqlite.OpenReadOnly(file)
	}
	if strings.HasPrefix(url, "jdbc:mysql:") || strings.HasPrefix(url, "mysql:") {
		addr, db, u, p, err := mysql.ConnParams(url, j.User, j.Pwd)
		if err != nil {
			return nil, err
		}
		return mysql.Open(addr, db, u, p)
	}
	cs, err := pg.ConnString(url, j.User, j.Pwd)
	if err != nil {
		return nil, errors.New("unsupported database URL (PostgreSQL, MySQL and SQLite only): " + j.URL)
	}
	return pg.Open(context.Background(), cs)
}

// Progress — вызывается перед генерацией каждого класса: n-й из total.
type Progress func(n, total int, class string)

// Run — генерация всех DTO и DAO проекта. Ошибка конкретного класса — в File.Err,
// общая ошибка (настройки, XML) — в error. progress может быть nil.
func Run(projectDir, sdmDir string, md meta.Metadata, progress Progress) (*Result, error) {
	if progress == nil {
		progress = func(int, int, string) {}
	}
	s, err := sdmxml.LoadSettings(sdmDir)
	if err != nil {
		return nil, err
	}
	sdm, err := sdmxml.LoadSdm(sdmDir)
	if err != nil {
		return nil, err
	}
	res := &Result{
		DtoDir: strings.TrimSpace(strings.ReplaceAll(s.Dto.Scope, `\`, "/")),
		DaoDir: strings.TrimSpace(strings.ReplaceAll(s.Dao.Scope, `\`, "/")),
	}
	if res.DtoDir == "" || res.DaoDir == "" {
		return nil, errors.New(cg.GolangScopesErr)
	}
	sqlRoot := projectDir + "/" + s.Folders.SQL
	dtoMode := cg.FieldNamesModeOf(s.Dto.FieldNamesMode)
	// у Java-плагина DTO и DAO — два отдельных прогона со своим JdbcUtils
	dtoGen, err := gogen.NewDTO(cg.NewCore(md, s, dtoMode, cg.LowerCamelCase, sqlRoot), s, sdm)
	if err != nil {
		return nil, err
	}
	total := len(sdm.DtoClasses) + len(sdm.DaoClasses)
	for _, c := range sdm.DtoClasses {
		progress(len(res.Dto)+1, total, c.Name)
		f := File{Class: c.Name, Name: fileName(c.Name)}
		f.Content, f.Err = dtoGen.Translate(c.Name)
		res.Dto = append(res.Dto, f)
	}
	daoGen, err := gogen.NewDAO(cg.NewCore(md, s, dtoMode, cg.LowerCamelCase, sqlRoot), s, sdm)
	if err != nil {
		return nil, err
	}
	for _, c := range sdm.DaoClasses {
		progress(len(res.Dto)+len(res.Dao)+1, total, c.Name)
		f := File{Class: c.Name, Name: fileName(c.Name)}
		f.Content, f.Err = daoGen.Translate(c)
		res.Dao = append(res.Dao, f)
	}
	return res, nil
}

func fileName(className string) string {
	return cg.FileNameSnakeCase(cg.ClassNameWithoutModel(className), "go")
}
