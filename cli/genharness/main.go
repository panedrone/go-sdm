// genharness — design-time сверка генератора с Java-плагином.
//
// Читает тот же список проектов, что sdm-plugins/harness (label | project dir | sdm folder),
// и пишет вывод в той же раскладке, что HarnessMain.java: <out>/<label>/dto|dao/<файл>,
// _output_dir.txt и <Class>.ERROR.txt. Сверка — diff -r с выводом Java-харнесса.
//
//	go run ./cli/genharness <projects-list-file> <out-dir>
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/panedrone/go-sdm/internal/gen"
	"github.com/panedrone/go-sdm/internal/sdmxml"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: genharness <projects-list-file> <out-dir>")
		os.Exit(2)
	}
	projects, err := readProjects(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	failed := 0
	for _, p := range projects {
		if err := runProject(p[0], p[1], p[2], filepath.Join(os.Args[2], p[0])); err != nil {
			failed++
			fmt.Println(p[0] + ": FAILED: " + err.Error())
		}
	}
	fmt.Printf("--- projects: %d, failed: %d\n", len(projects), failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func readProjects(path string) ([][3]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var res [][3]string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) != 3 {
			return nil, fmt.Errorf("expected 'label | project dir | sdm folder': %s", line)
		}
		res = append(res, [3]string{strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2])})
	}
	return res, sc.Err()
}

func runProject(label, projectDir, sdmFolder, out string) error {
	sdmDir := sdmFolder
	if !filepath.IsAbs(sdmDir) {
		sdmDir = filepath.Join(projectDir, sdmFolder)
	}
	s, err := sdmxml.LoadSettings(sdmDir)
	if err != nil {
		return err
	}
	md, err := gen.OpenMetadata(projectDir, s.Jdbc)
	if err != nil {
		return err
	}
	defer md.Close()
	res, err := gen.Run(projectDir, sdmDir, md, nil)
	if err != nil {
		return err
	}
	errors := write(filepath.Join(out, "dto"), res.DtoDir, res.Dto)
	errors += write(filepath.Join(out, "dao"), res.DaoDir, res.Dao)
	fmt.Printf("%s: go.dal, errors: %d\n", label, errors)
	return nil
}

func write(dir, outputDir string, files []gen.File) int {
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "_output_dir.txt"), []byte(outputDir), 0o644)
	errors := 0
	for _, f := range files {
		if f.Err != nil {
			errors++
			_ = os.WriteFile(filepath.Join(dir, f.Class+".ERROR.txt"), []byte("java.lang.Exception: "+f.Err.Error()), 0o644)
			continue
		}
		_ = os.WriteFile(filepath.Join(dir, f.Name), []byte(f.Content), 0o644)
	}
	return errors
}
