package contracts

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestHandwrittenPackagesHaveDocGo(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	dirs, err := dirsWithNonTestGoFiles(
		filepath.Join(repoRoot, "cmd"),
		filepath.Join(repoRoot, "internal"),
		filepath.Join(repoRoot, "tools"),
	)
	if err != nil {
		t.Fatalf("list package dirs: %v", err)
	}

	var missing []string
	for _, dir := range dirs {
		docPath := filepath.Join(dir, "doc.go")
		if _, err := os.Stat(docPath); err != nil {
			if os.IsNotExist(err) {
				rel, relErr := filepath.Rel(repoRoot, dir)
				if relErr != nil {
					t.Fatalf("relative dir %s: %v", dir, relErr)
				}
				missing = append(missing, filepath.ToSlash(rel))
				continue
			}
			t.Fatalf("stat %s: %v", docPath, err)
		}
	}

	if len(missing) == 0 {
		return
	}
	sort.Strings(missing)
	t.Fatalf("package docs missing doc.go:\n%s", strings.Join(missing, "\n"))
}

func TestDocGoUsesPackageCommentConvention(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)
	dirs, err := dirsWithNonTestGoFiles(
		filepath.Join(repoRoot, "cmd"),
		filepath.Join(repoRoot, "internal"),
		filepath.Join(repoRoot, "tools"),
	)
	if err != nil {
		t.Fatalf("list package dirs: %v", err)
	}

	fset := token.NewFileSet()
	var violations []string
	for _, dir := range dirs {
		docPath := filepath.Join(dir, "doc.go")
		file, err := parser.ParseFile(fset, docPath, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", docPath, err)
		}

		if file.Doc == nil {
			violations = append(violations, fmt.Sprintf("%s: missing package comment", relPathOrFatal(t, repoRoot, docPath)))
			continue
		}
		docText := strings.TrimSpace(file.Doc.Text())
		prefix := "Package " + file.Name.Name
		if docText != prefix && !strings.HasPrefix(docText, prefix+" ") {
			violations = append(violations, fmt.Sprintf("%s: package comment must begin with %q", relPathOrFatal(t, repoRoot, docPath), prefix))
		}
	}

	if len(violations) == 0 {
		return
	}
	sort.Strings(violations)
	t.Fatalf("package comment convention violations:\n%s", strings.Join(violations, "\n"))
}

func repoRootFromThisFile(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
}

func dirsWithNonTestGoFiles(roots ...string) ([]string, error) {
	dirs := make(map[string]struct{})
	for _, root := range roots {
		walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			dirs[filepath.Dir(path)] = struct{}{}
			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}

	out := make([]string, 0, len(dirs))
	for dir := range dirs {
		out = append(out, dir)
	}
	sort.Strings(out)
	return out, nil
}

func relPathOrFatal(t *testing.T, repoRoot, path string) string {
	t.Helper()
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		t.Fatalf("relative path %s: %v", path, err)
	}
	return filepath.ToSlash(rel)
}
