package gen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Change — файл, который generate запишет (или validate считает устаревшим).
type Change struct {
	Path    string // относительно проекта, с '/'
	Missing bool   // файла ещё нет
}

// Sync — сравнивает результат с файлами проекта и, если write, записывает изменившиеся.
// Концы строк не сравниваются: после git с autocrlf файл на Windows может быть в CRLF.
func Sync(projectDir string, res *Result, write bool) ([]Change, error) {
	var changes []Change
	var errs []error
	for _, set := range []struct {
		dir   string
		files []File
	}{{res.DtoDir, res.Dto}, {res.DaoDir, res.Dao}} {
		for _, f := range set.files {
			if f.Err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", f.Class, f.Err))
				continue
			}
			rel := relPath(set.dir, f.Name)
			path := filepath.Join(projectDir, filepath.FromSlash(rel))
			old, err := os.ReadFile(path)
			missing := errors.Is(err, os.ErrNotExist)
			if err != nil && !missing {
				return nil, err
			}
			if !missing && strings.ReplaceAll(string(old), "\r\n", "\n") == f.Content {
				continue
			}
			changes = append(changes, Change{Path: rel, Missing: missing})
			if !write {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(path, []byte(f.Content), 0o644); err != nil {
				return nil, err
			}
		}
	}
	return changes, errors.Join(errs...)
}
