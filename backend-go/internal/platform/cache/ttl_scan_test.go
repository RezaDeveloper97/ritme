package cache

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ttlFreeWrites lists the Go files (relative to backend-go/, slash separated) allowed to write a Redis key without
// an expiry. Redis runs with `maxmemory-policy volatile-lru` on stage/beta (T-M9-10): only keys with a TTL can be
// evicted, so every key without one is permanent memory. Add a file here only with a reason next to it.
var ttlFreeWrites = map[string]string{}

// setMethods are the cache.Client / go-redis writes taking (ctx, key, value, ttl).
var setMethods = map[string]bool{"Set": true, "SetNX": true, "SetXX": true}

// TestCacheWritesHaveTTL scans the non-test Go source for Set/SetNX/SetXX(ctx, key, value, ttl) calls whose ttl is
// a literal zero (0, 0*time.Second, time.Duration(0)) outside ttlFreeWrites.
func TestCacheWritesHaveTTL(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	var offenders []string
	scanned := 0
	for _, dir := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
			if err != nil {
				return nil //nolint:nilerr // a half-written file of another agent is not this test's concern
			}
			scanned++
			if _, ok := ttlFreeWrites[rel]; ok {
				return nil
			}
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) != 4 {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || !setMethods[sel.Sel.Name] {
					return true
				}
				if isZero(call.Args[3]) {
					offenders = append(offenders, rel+":"+strconv.Itoa(fset.Position(call.Pos()).Line)+" "+sel.Sel.Name)
				}
				return true
			})
			return nil
		})
		require.NoError(t, err)
	}
	require.Positive(t, scanned, "no Go files scanned: wrong root %s", root)
	assert.Empty(t, offenders, "Redis writes without a TTL are never evicted under volatile-lru; give them a TTL "+
		"or list the file in ttlFreeWrites with a reason")
}

func TestIsZero(t *testing.T) {
	for src, want := range map[string]bool{
		"0": true, "0 * time.Second": true, "time.Second * 0": true, "time.Duration(0)": true, "(0)": true,
		"time.Minute": false, "ttl": false, "5 * time.Second": false, "redis.KeepTTL": false, "-1": false,
	} {
		e, err := parser.ParseExpr(src)
		require.NoError(t, err, src)
		assert.Equal(t, want, isZero(e), src)
	}
}

// isZero reports whether e is a literal zero duration.
func isZero(e ast.Expr) bool {
	switch v := e.(type) {
	case *ast.BasicLit:
		return v.Kind == token.INT && strings.Trim(v.Value, "0_") == "" ||
			v.Kind == token.FLOAT && strings.Trim(v.Value, "0._") == ""
	case *ast.ParenExpr:
		return isZero(v.X)
	case *ast.BinaryExpr:
		return v.Op == token.MUL && (isZero(v.X) || isZero(v.Y))
	case *ast.CallExpr: // time.Duration(0)
		return len(v.Args) == 1 && isZero(v.Args[0])
	}
	return false
}
