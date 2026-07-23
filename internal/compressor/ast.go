package compressor

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
	"tkngate/internal/config"
)

// Compress evaluates the content and structurally reduces it based on its language.
//
// v2.7.0: JSON payloads (tool-call arguments, structured output schemas) are
// explicitly excluded from compression to prevent corruption.
func Compress(content string) string {
	// v2.7.0: Never compress raw JSON — tool_calls arguments and
	// response_format schemas must pass through untouched.
	if isJSON(content) {
		return content
	}

	if isGoCode(content) && (config.Cfg.Compressor.EnableGo == nil || *config.Cfg.Compressor.EnableGo) {
		return compressGo(content)
	} else if isPythonCode(content) && (config.Cfg.Compressor.EnablePython == nil || *config.Cfg.Compressor.EnablePython) {
		return compressPython(content)
	} else if isJSCode(content) && (config.Cfg.Compressor.EnableJS == nil || *config.Cfg.Compressor.EnableJS) {
		return compressJS(content)
	}

	// Fallback: return unchanged
	return content
}

// isJSON returns true if the content looks like a raw JSON object or array.
// This prevents the compressor from mutating structured outputs and tool-call
// argument strings that must remain valid JSON.
func isJSON(content string) bool {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) < 2 {
		return false
	}
	return (trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}') ||
		(trimmed[0] == '[' && trimmed[len(trimmed)-1] == ']')
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
