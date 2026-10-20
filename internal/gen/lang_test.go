package gen

import (
	"os"
	"path/filepath"
	"testing"
)

// The language of an sdm folder: python.dal means Python, without it Go, both is an error.
func TestLangOf(t *testing.T) {
	dir := t.TempDir()
	if lang, err := LangOf(dir); err != nil || lang != Go {
		t.Errorf("no root file: %s %v", lang, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.dal"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if lang, err := LangOf(dir); err != nil || lang != Go {
		t.Errorf("go.dal: %s %v", lang, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "python.dal"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LangOf(dir); err == nil {
		t.Error("go.dal and python.dal: no error")
	}
	if err := os.Remove(filepath.Join(dir, "go.dal")); err != nil {
		t.Fatal(err)
	}
	if lang, err := LangOf(dir); err != nil || lang != Python {
		t.Errorf("python.dal: %s %v", lang, err)
	}
}
