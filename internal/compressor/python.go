package compressor

import (
	"strings"
)

// compressPython performs heuristic indentation-tracking minification for Python.
// It looks for 'def ' or 'class ' and drops the deeply indented implementation.
func compressPython(content string) string {
	var result strings.Builder
	
	lines := strings.Split(content, "\n")
	
	inFunctionBlock := false
	baseIndentLevel := 0
	
	for _, line := range lines {
		// Calculate current indentation level (spaces/tabs at the start)
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		trimmed := strings.TrimSpace(line)
		
		if trimmed == "" {
			// Skip empty lines inside a dropped block, preserve otherwise
			if !inFunctionBlock {
				result.WriteString(line)
				result.WriteByte('\n')
			}
			continue
		}
		
		if inFunctionBlock {
			if indent > baseIndentLevel {
				// We are still inside the function body, drop this line
				continue
			} else {
				// The indentation has returned to or below the base level, exit the block
				inFunctionBlock = false
			}
		}
		
		// If we are here, we are not dropping the line.
		result.WriteString(line)
		result.WriteByte('\n')
		
		// Does this line declare a function or class?
		if (strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "class ")) && strings.HasSuffix(trimmed, ":") {
			inFunctionBlock = true
			baseIndentLevel = indent
			
			// Insert the dummy pass statement at the next indentation level
			// (assume 4 spaces for Python standard)
			result.WriteString(line[:indent])
			result.WriteString("    pass  # body omitted\n")
		}
	}
	
	return strings.TrimSuffix(result.String(), "\n")
}
