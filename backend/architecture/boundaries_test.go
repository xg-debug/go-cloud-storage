// Package architecture checks source ownership in addition to Go's internal rules.
package architecture

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const module = "go-cloud-storage/backend/"

func owner(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	if parts[0] == "gateway" {
		return "gateway"
	}
	if len(parts) > 1 && parts[0] == "services" {
		return "services/" + parts[1]
	}
	return ""
}

func TestServiceBoundaries(t *testing.T) {
	root := ".."
	for _, removed := range []string{"internal", "infrastructure", "services/runtime", "cmd"} {
		if _, err := os.Stat(filepath.Join(root, removed)); !os.IsNotExist(err) {
			t.Errorf("central implementation %s must not return: %v", removed, err)
		}
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == "tmp" || entry.Name() == "vendor") {
			return filepath.SkipDir
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		tree, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		sourceOwner := owner(rel)
		for _, imp := range tree.Imports {
			imported, _ := strconv.Unquote(imp.Path.Value)
			if !strings.HasPrefix(imported, module) {
				continue
			}
			target := strings.TrimPrefix(imported, module)
			targetOwner := owner(target)
			if targetOwner != "" && sourceOwner != targetOwner {
				t.Errorf("%s imports another service: %s", rel, imported)
			}
			if strings.HasPrefix(rel, "pkg/") && !strings.HasPrefix(target, "pkg/") {
				t.Errorf("shared package %s depends on implementation %s", rel, target)
			}
			if strings.Contains(rel, "/internal/domain/") && (strings.Contains(target, "/application") || strings.Contains(target, "/adapters") || strings.Contains(target, "/transport") || strings.Contains(target, "/bootstrap")) {
				t.Errorf("domain %s depends on outer layer %s", rel, target)
			}
		}
		if strings.HasPrefix(rel, "pkg/contracts/") {
			ast.Inspect(tree, func(n ast.Node) bool {
				if field, ok := n.(*ast.Field); ok && field.Tag != nil && strings.Contains(field.Tag.Value, "gorm:") {
					t.Errorf("wire contract %s contains persistence metadata", rel)
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
