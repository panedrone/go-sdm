// Package gen — прогон генерации для одного проекта: settings.xml и sdm.xml из папки sdm,
// метаданные — из базы <jdbc url>, результат — тексты файлов DTO и DAO.
package gen

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

// Run — генерация DTO и DAO проекта: всех или только классов из only (имена, см. Select).
// Ошибка конкретного класса — в File.Err, общая ошибка (настройки, XML) — в error.
// progress может быть nil.
func Run(projectDir, sdmDir string, md meta.Metadata, only map[string]bool, progress Progress) (*Result, error) {
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
	res := &Result{}
	if res.DtoDir, res.DaoDir, err = scopes(s); err != nil {
		return nil, err
	}
	picked := func(name string) bool { return only == nil || only[name] }
	sqlRoot := projectDir + "/" + s.Folders.SQL
	dtoMode := cg.FieldNamesModeOf(s.Dto.FieldNamesMode)
	// у Java-плагина DTO и DAO — два отдельных прогона со своим JdbcUtils
	dtoGen, err := gogen.NewDTO(cg.NewCore(md, s, dtoMode, cg.LowerCamelCase, sqlRoot), s, sdm)
	if err != nil {
		return nil, err
	}
	total := 0
	for _, c := range sdm.Classes {
		if picked(c.Name) {
			total++
		}
	}
	for _, c := range sdm.DtoClasses {
		if !picked(c.Name) {
			continue
		}
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
		if !picked(c.Name) {
			continue
		}
		progress(len(res.Dto)+len(res.Dao)+1, total, c.Name)
		f := File{Class: c.Name, Name: fileName(c.Name)}
		f.Content, f.Err = daoGen.Translate(c)
		res.Dao = append(res.Dao, f)
	}
	return res, nil
}

func scopes(s *sdmxml.Settings) (dto, dao string, err error) {
	dto = strings.TrimSpace(strings.ReplaceAll(s.Dto.Scope, `\`, "/"))
	dao = strings.TrimSpace(strings.ReplaceAll(s.Dao.Scope, `\`, "/"))
	if dto == "" || dao == "" {
		return "", "", errors.New(cg.GolangScopesErr)
	}
	return dto, dao, nil
}

// Select — имена классов по номерам (с 1, в порядке объявления, как у sdm ls) и по
// именам; в одном аргументе может быть несколько через запятую. models/daos — только
// модели или только DAO; оба или ни одного — и те, и другие. Ни аргументов, ни
// models/daos — nil, все.
func Select(classes []sdmxml.Class, args []string, models, daos bool) (map[string]bool, error) {
	only, err := byArgs(classes, args)
	if err != nil || models == daos {
		return only, err
	}
	picked := map[string]bool{}
	for _, c := range classes {
		if c.Dao == daos && (only == nil || only[c.Name]) {
			picked[c.Name] = true
		}
	}
	return picked, nil
}

func byArgs(classes []sdmxml.Class, args []string) (map[string]bool, error) {
	var only map[string]bool
	for _, a := range args {
		for _, x := range strings.Split(a, ",") {
			x = strings.TrimSpace(x)
			if x == "" {
				continue
			}
			if only == nil {
				only = map[string]bool{}
			}
			if n, err := strconv.Atoi(x); err == nil {
				if n < 1 || n > len(classes) {
					return nil, fmt.Errorf("no class #%d: sdm.xml has %d", n, len(classes))
				}
				only[classes[n-1].Name] = true
				continue
			}
			if !hasClass(classes, x) {
				return nil, fmt.Errorf("no class %q in sdm.xml", x)
			}
			only[x] = true
		}
	}
	return only, nil
}

func hasClass(classes []sdmxml.Class, name string) bool {
	for _, c := range classes {
		if c.Name == name {
			return true
		}
	}
	return false
}

// Entry — класс sdm.xml и его файл.
type Entry struct {
	N int // номер, с 1, в порядке объявления
	sdmxml.Class
	Path     string    // относительно проекта, с '/'
	Modified time.Time // нулевое — файла нет
}

// List — классы sdm.xml в порядке объявления, без базы.
func List(projectDir, sdmDir string) ([]Entry, error) {
	s, err := sdmxml.LoadSettings(sdmDir)
	if err != nil {
		return nil, err
	}
	sdm, err := sdmxml.LoadSdm(sdmDir)
	if err != nil {
		return nil, err
	}
	dtoDir, daoDir, err := scopes(s)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(sdm.Classes))
	for i, c := range sdm.Classes {
		dir := dtoDir
		if c.Dao {
			dir = daoDir
		}
		e := Entry{N: i + 1, Class: c, Path: relPath(dir, fileName(c.Name))}
		if fi, err := os.Stat(filepath.Join(projectDir, filepath.FromSlash(e.Path))); err == nil {
			e.Modified = fi.ModTime()
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func relPath(dir, name string) string {
	return strings.TrimSuffix(dir, "/") + "/" + name
}

func fileName(className string) string {
	return cg.FileNameSnakeCase(cg.ClassNameWithoutModel(className), "go")
}
