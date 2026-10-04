package hygiene_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

func TestTypeScriptMethodCoverage(t *testing.T) {
	root := filepath.Join("..", "..")
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "internal", "ipc", "protocol.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	native := map[string]bool{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "methodPermission" {
			continue
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			clause, ok := node.(*ast.CaseClause)
			if !ok {
				return true
			}
			for _, expr := range clause.List {
				literal, ok := expr.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					t.Fatal("expected literal method names in methodPermission")
				}
				name, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatal(err)
				}
				native[name] = true
			}
			return true
		})
	}
	if len(native) == 0 {
		t.Fatal("native method table not found")
	}
	declaration, err := os.ReadFile(filepath.Join(root, "types", "velox.d.ts"))
	if err != nil {
		t.Fatal(err)
	}
	// MethodMap uses one quoted key per line; the TypeScript compiler checks its syntax.
	keys := regexp.MustCompile(`(?m)^  "([a-z]+\.[A-Za-z]+)": \{ params:`).FindAllSubmatch(declaration, -1)
	typed := map[string]bool{}
	for _, key := range keys {
		name := string(key[1])
		if typed[name] {
			t.Errorf("duplicate TypeScript method %s", name)
		}
		typed[name] = true
		if !native[name] {
			t.Errorf("TypeScript declares unsupported method %s", name)
		}
	}
	for name := range native {
		if !typed[name] {
			t.Errorf("TypeScript is missing native method %s", name)
		}
	}
	t.Logf("checked %d native methods against TypeScript declarations", len(native))
}
