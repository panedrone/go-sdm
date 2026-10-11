// Package gen — прогон генерации для одного проекта: settings.xml и sdm.xml из папки sdm,
// метаданные — из базы <jdbc url>, результат — тексты файлов DTO и DAO.
package gen

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/panedrone/go-sdm/internal/cg"
	"github.com/panedrone/go-sdm/internal/gogen"
	"github.com/panedrone/go-sdm/internal/meta"
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

// OpenMetadata — база из <jdbc url user pwd>. $PROJECT_DIR$ и ${project_loc} — папка проекта,
// относительный путь SQLite — тоже от неё.
func OpenMetadata(projectDir string, j sdmxml.Jdbc) (meta.Metadata, error) {
	url := strings.NewReplacer("$PROJECT_DIR$", projectDir, "${project_loc}", projectDir).Replace(strings.TrimSpace(j.URL))
	if file, ok := strings.CutPrefix(url, "jdbc:sqlite:"); ok {
		if file == ":memory:" {
			return nil, errors.New("jdbc:sqlite::memory: has no tables to generate from")
		}
		if !filepath.IsAbs(file) {
			file = filepath.Join(projectDir, file)
		}
		return sqlite.OpenReadOnly(file)
	}
	cs, err := pg.ConnString(url, j.User, j.Pwd)
	if err != nil {
		return nil, errors.New("unsupported database URL (PostgreSQL and SQLite only): " + j.URL)
	}
	return pg.Open(context.Background(), cs)
}

// Run — генерация всех DTO и DAO проекта. Ошибка конкретного класса — в File.Err,
// общая ошибка (настройки, XML) — в error.
func Run(projectDir, sdmDir string, md meta.Metadata) (*Result, error) {
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
	for _, c := range sdm.DtoClasses {
		f := File{Class: c.Name, Name: fileName(c.Name)}
		f.Content, f.Err = dtoGen.Translate(c.Name)
		res.Dto = append(res.Dto, f)
	}
	daoGen, err := gogen.NewDAO(cg.NewCore(md, s, dtoMode, cg.LowerCamelCase, sqlRoot), s, sdm)
	if err != nil {
		return nil, err
	}
	for _, c := range sdm.DaoClasses {
		f := File{Class: c.Name, Name: fileName(c.Name)}
		f.Content, f.Err = daoGen.Translate(c)
		res.Dao = append(res.Dao, f)
	}
	return res, nil
}

func fileName(className string) string {
	return cg.FileNameSnakeCase(cg.ClassNameWithoutModel(className), "go")
}
