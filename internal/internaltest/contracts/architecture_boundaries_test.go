package contracts

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
)

func TestCoreAndAdapterPackageBoundaries(t *testing.T) {
	repoRoot := repoRootFromThisFile(t)

	assertNoImportsUnder(t, repoRoot, "internal/service", []string{
		"database/sql",
		"modernc.org/sqlite",
		"google.golang.org/grpc",
		"google.golang.org/protobuf",
		"github.com/fracturing-space/game/api/gen/go/game/v1",
		"github.com/fracturing-space/game/internal/storage/",
	})
	assertNoImportsUnder(t, repoRoot, "internal/storage/memory", []string{
		"github.com/fracturing-space/game/internal/transport/",
	})
	assertNoImportsUnder(t, repoRoot, "internal/storage/sqlite", []string{
		"github.com/fracturing-space/game/internal/transport/",
	})
	assertNoImportsUnder(t, repoRoot, "internal/transport/grpc/gamev1", []string{
		"github.com/fracturing-space/game/internal/storage/",
	})
}

func assertNoImportsUnder(t *testing.T, repoRoot, relDir string, disallowed []string) {
	t.Helper()

	dir := filepath.Join(repoRoot, filepath.FromSlash(relDir))
	fset := token.NewFileSet()
	var violations []string
	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if slices.Contains(disallowed, importPath) {
				violations = append(violations, fmt.Sprintf("%s imports %s", relPathOrFatal(t, repoRoot, path), importPath))
				continue
			}
			for _, prefix := range disallowed {
				if strings.HasSuffix(prefix, "/") && strings.HasPrefix(importPath, prefix) {
					violations = append(violations, fmt.Sprintf("%s imports %s", relPathOrFatal(t, repoRoot, path), importPath))
					break
				}
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("walk %s: %v", relDir, err)
	}

	if len(violations) == 0 {
		return
	}
	sort.Strings(violations)
	t.Fatalf("architecture boundary violations:\n%s", strings.Join(violations, "\n"))
}
