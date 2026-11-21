package type4

// The guard on the defect class the review of pull request #44 found, rather
// than on the five sites it found.
//
// PDFunctionType4.eval catches EmptyStackException and ClassCastException
// (PDFBOX-6320) and turns them into an IOException its callers handle. The port
// reaches that boundary by panicking with an error wrapping ErrEmptyStack or
// ErrWrongType and recovering it. A plain Go type assertion, `x.(T)` in a
// single-value context, panics with *runtime.TypeAssertionError instead: it is
// an error, it wraps nothing, and `errors.Is` says no, so the recover re-panics
// and a malformed function takes the page the Java only loses a shading of.
//
// Every cast in this package stands for a Java cast, and every Java cast in it
// throws ClassCastException, so none of them may be the single-value form. The
// test below reads the package's own source and says so, which is a rule a
// reviewer does not have to remember and a new operator cannot quietly break.
// gofmt, go vet and go test do not check this; forcetypeassert, which is not in
// this repository's gate, would.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"strings"
	"testing"
)

// TestEveryTypeAssertionIsChecked fails for a `v := x.(T)` or an `x.(T)` used
// as a value, and passes for `v, ok := x.(T)` and for the `x.(type)` of a type
// switch, neither of which panics.
func TestEveryTypeAssertionIsChecked(t *testing.T) {
	fileSet := token.NewFileSet()
	packages, err := parser.ParseDir(fileSet, ".", func(info fs.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parsing the package: %v", err)
	}
	if len(packages) == 0 {
		t.Fatal("the parser found no package to read")
	}

	for _, parsed := range packages {
		for _, file := range parsed.Files {
			// the two-value assignments, by the position of their assertion
			checked := map[token.Pos]bool{}
			ast.Inspect(file, func(node ast.Node) bool {
				assignment, isAssignment := node.(*ast.AssignStmt)
				if !isAssignment || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 1 {
					return true
				}
				if assertion, isAssertion := assignment.Rhs[0].(*ast.TypeAssertExpr); isAssertion {
					checked[assertion.Pos()] = true
				}
				return true
			})

			ast.Inspect(file, func(node ast.Node) bool {
				assertion, isAssertion := node.(*ast.TypeAssertExpr)
				if !isAssertion {
					return true
				}
				if assertion.Type == nil {
					return true // x.(type), the head of a type switch
				}
				if checked[assertion.Pos()] {
					return true
				}
				t.Errorf("%s: unchecked type assertion. It panics with "+
					"*runtime.TypeAssertionError, which PDFunctionType4.Eval "+
					"cannot tell from a defect of the port; take the second "+
					"value and raise ErrWrongType, as ExecutionContext's "+
					"accessors do", fileSet.Position(assertion.Pos()))
				return true
			})
		}
	}
}
