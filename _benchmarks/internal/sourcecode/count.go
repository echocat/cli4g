// Package sourcecode measures readable adapter bodies independently of timing.
package sourcecode

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"unicode"

	"github.com/uudashr/gocognit"
)

// Metrics describes the selected constructor and required supporting methods.
type Metrics struct {
	Characters          int
	CognitiveComplexity int
}

// AnalyzeFile counts non-whitespace Unicode characters in the named bodies
// and sums their gocognit scores. Nested closures are included by gocognit.
// Signatures and outer braces do not count as characters. Methods use
// Receiver.Method names; unselected declarations are excluded.
func AnalyzeFile(path string, names ...string) (Metrics, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Metrics{}, err
	}
	return analyzeSource(data, names...)
}

func analyzeSource(data []byte, names ...string) (Metrics, error) {
	formatted, err := format.Source(data)
	if err != nil {
		return Metrics{}, err
	}
	positions := token.NewFileSet()
	file, err := parser.ParseFile(positions, "", formatted, 0)
	if err != nil {
		return Metrics{}, err
	}
	bodies := make(map[string]Metrics)
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		name := function.Name.Name
		if function.Recv != nil {
			name = receiverName(function.Recv.List[0].Type) + "." + name
		}
		start := positions.Position(function.Body.Lbrace).Offset + 1
		end := positions.Position(function.Body.Rbrace).Offset
		characters := 0
		for _, character := range string(formatted[start:end]) {
			if !unicode.IsSpace(character) {
				characters++
			}
		}
		bodies[name] = Metrics{
			Characters:          characters,
			CognitiveComplexity: gocognit.Complexity(function),
		}
	}
	var total Metrics
	for _, name := range names {
		metrics, found := bodies[name]
		if !found {
			return Metrics{}, fmt.Errorf("missing function body %q", name)
		}
		total.Characters += metrics.Characters
		total.CognitiveComplexity += metrics.CognitiveComplexity
	}
	return total, nil
}

func receiverName(expression ast.Expr) string {
	switch receiver := expression.(type) {
	case *ast.Ident:
		return receiver.Name
	case *ast.StarExpr:
		return receiverName(receiver.X)
	default:
		return ""
	}
}
