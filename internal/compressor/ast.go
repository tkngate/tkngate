package compressor

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
)

// Compress evaluates the content and structurally reduces it based on its language.
func Compress(content string) string {
	if isGoCode(content) {
		return compressGo(content)
	} else if isPythonCode(content) {
		return compressPython(content)
	} else if isJSCode(content) {
		return compressJS(content)
	}
	
	// Fallback: return unchanged
	return content
}

func isGoCode(content string) bool {
	return strings.Contains(content, "package ") && strings.Contains(content, "func ")
}

func isPythonCode(content string) bool {
	lines := strings.Split(content, "\n")
	defCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "class ") {
			defCount++
		}
	}
	return defCount >= 2
}

func isJSCode(content string) bool {
	lines := strings.Split(content, "\n")
	fnCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "function ") || strings.Contains(trimmed, "=> {") || strings.HasPrefix(trimmed, "const ") || strings.HasPrefix(trimmed, "export ") {
			fnCount++
		}
	}
	return fnCount >= 2
}

func compressGo(content string) string {
	// If it fails to parse (e.g. it's natural language or Python), we return it unchanged.
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", content, 0)
	if err != nil {
		// Not valid Go code, or just natural language. Return as is for now.
		return content
	}

	// Phase I: Lossless Pruning (Strip Comments)
	// We do this by simply not attaching comments to the reconstructed AST.
	node.Comments = nil

	// Phase II: Condensation (Prune deep function bodies)
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if x.Body != nil {
				// Replace the body with an empty block and a semantic marker.
				// We can't insert a comment easily since we stripped them, but we can insert a dummy statement or just leave it empty.
				// A clean way is just to make the body empty, saving tokens.
				x.Body.List = nil 
				// A completely empty block '{}' implies the body is omitted.
			}
		}
		return true
	})

	var buf bytes.Buffer
	// Print the modified AST back to a string
	if err := printer.Fprint(&buf, fset, node); err != nil {
		return content // fallback
	}

	return buf.String()
}
