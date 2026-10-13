package gen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Change is a file that generate will write (or validate considers out of date).
type Change struct {
	Path    string // relative to the project, with '/'
	Missing bool   // no file yet
}

// Sync compares the result with the project files and, if write, writes the changed ones.
// Line endings are not compared: after git with autocrlf a file on Windows may be in CRLF.
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
