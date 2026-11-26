// Command checkrules enforces the repo's mechanical rules, which a reviewer
// would otherwise have to remember:
//
//   - Time and randomness come from the injected app.Deps. Outside main.go, no
//     non-test code reads the real clock (time.Now and friends, tea.Tick,
//     tea.Every) or the global random source (package-level math/rand
//     functions). Building a source from injected values (rand.New,
//     rand.NewPCG) is fine.
//   - There is one test seam. Every test file is in internal/app and declares
//     package app_test.
//   - Tests run in parallel. Every top-level Test function (TestMain aside)
//     starts with t.Parallel(), so the suite stays quick as it grows. State
//     shared by all tests, like the colour profile, is set once in TestMain.
//
// Run it from the repo root: go run ./tools/checkrules
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// banned lists, per import path, the package-level names that bypass the
// injected Clock or random source.
var banned = map[string]map[string]bool{
	"time":                               set("Now", "Since", "Until", "Sleep", "Tick", "After", "AfterFunc", "NewTicker", "NewTimer"),
	"github.com/charmbracelet/bubbletea": set("Tick", "Every"),
}

// randAllowed are the math/rand names that build a source from given values,
// rather than drawing from the global one.
var randAllowed = set("New", "NewPCG", "NewChaCha8", "NewSource", "Rand", "Source", "PCG", "ChaCha8")

const (
	testDir     = "internal/app"
	testPackage = "app_test"
)

func main() {
	var problems []string
	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); path != "." && (strings.HasPrefix(name, ".") || name == "testdata" || name == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, "_test.go") {
			problems = append(problems, checkTestFile(path, f)...)
			problems = append(problems, checkParallel(fset, f)...)
		} else if path != "main.go" {
			problems = append(problems, checkInjection(fset, f)...)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "checkrules:", err)
		os.Exit(2)
	}
	for _, p := range problems {
		fmt.Fprintln(os.Stderr, p)
	}
	if len(problems) > 0 {
		os.Exit(1)
	}
}

// checkTestFile keeps every test behind the one seam.
func checkTestFile(path string, f *ast.File) []string {
	if filepath.ToSlash(filepath.Dir(path)) != testDir || f.Name.Name != testPackage {
		return []string{fmt.Sprintf("%s: tests go only in %s as package %s, driven through app.New and the harness", path, testDir, testPackage)}
	}
	return nil
}

// checkParallel flags a top-level Test function whose first statement isn't
// t.Parallel().
func checkParallel(fset *token.FileSet, f *ast.File) []string {
	var problems []string
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Body == nil || !strings.HasPrefix(fn.Name.Name, "Test") || fn.Name.Name == "TestMain" {
			continue
		}
		if !startsWithParallel(fn.Body) {
			problems = append(problems, fmt.Sprintf("%s: %s must start with t.Parallel()", fset.Position(fn.Pos()), fn.Name.Name))
		}
	}
	return problems
}

// startsWithParallel reports whether body's first statement calls Parallel.
func startsWithParallel(body *ast.BlockStmt) bool {
	if len(body.List) == 0 {
		return false
	}
	expr, ok := body.List[0].(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := expr.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "Parallel"
}

// checkInjection flags reads of the real clock or the global random source.
func checkInjection(fset *token.FileSet, f *ast.File) []string {
	// Map each local import name to its import path.
	imports := map[string]string{}
	for _, spec := range f.Imports {
		path, _ := strconv.Unquote(spec.Path.Value)
		name := filepath.Base(path)
		switch path {
		case "math/rand/v2":
			name = "rand"
		case "github.com/charmbracelet/bubbletea":
			name = "tea"
		}
		if spec.Name != nil {
			name = spec.Name.Name
		}
		imports[name] = path
	}

	var problems []string
	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		path, ok := imports[pkg.Name]
		if !ok {
			return true
		}
		name := sel.Sel.Name
		var bad bool
		switch path {
		case "math/rand", "math/rand/v2":
			bad = !randAllowed[name]
		default:
			bad = banned[path][name]
		}
		if bad {
			problems = append(problems, fmt.Sprintf("%s: %s.%s bypasses app.Deps; take time from Deps.Clock and randomness from Deps.Rand (only main.go wires the real ones)",
				fset.Position(sel.Pos()), pkg.Name, name))
		}
		return true
	})
	return problems
}

func set(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}
