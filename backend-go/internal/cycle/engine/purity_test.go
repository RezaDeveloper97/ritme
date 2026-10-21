package engine

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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPurity enforces doc.go: no clock, no mutable package-level state and no I/O or config imports in the engine
// package and its sub-packages (non-test files).
func TestPurity(t *testing.T) {
	forbidden := func(path string) bool {
		const mod = "github.com/ritme/backend-go/"
		switch {
		case path == "os" || strings.HasPrefix(path, "os/"),
			path == "net" || strings.HasPrefix(path, "net/"),
			path == "log" || path == "log/slog",
			strings.HasPrefix(path, "database/sql"),
			strings.Contains(path, "redis"),
			strings.HasPrefix(path, "math/rand"),
			path == mod+"internal/clinicalconfig",
			path == mod+"internal/enums",
			path == mod+"internal/platform/db" || strings.HasPrefix(path, mod+"internal/platform/db/"),
			path == mod+"internal/platform/clock":
			return true
		}
		return false
	}
	clock := map[string]bool{"Now": true, "Since": true, "Until": true, "Sleep": true, "Tick": true, "After": true,
		"AfterFunc": true, "NewTimer": true, "NewTicker": true}

	files := 0
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		files++
		timeName := ""
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			assert.False(t, forbidden(p), "%s imports %s", path, p)
			if p == "time" {
				timeName = "time"
				if imp.Name != nil {
					timeName = imp.Name.Name
				}
			}
		}
		for _, decl := range f.Decls {
			if g, ok := decl.(*ast.GenDecl); ok && g.Tok == token.VAR {
				assert.Fail(t, "mutable package-level var", "%s", fset.Position(g.Pos()))
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok || timeName == "" {
				return true
			}
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == timeName && clock[sel.Sel.Name] {
				assert.Fail(t, "clock use", "%s: time.%s", fset.Position(sel.Pos()), sel.Sel.Name)
			}
			return true
		})
		return nil
	})
	require.NoError(t, err)
	assert.Positive(t, files)
}
