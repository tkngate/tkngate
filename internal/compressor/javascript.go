package compressor

import (
	"strings"
)

// compressJS performs heuristic bracket-counting minification for JS/TS.
// It looks for common function signatures and drops their bodies.
func compressJS(content string) string {
	var result strings.Builder
	
	inFunctionBlock := false
	bracketDepth := 0
	
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Are we currently ignoring lines inside a function body?
		if inFunctionBlock {
			// We need to count brackets on this line to see if we exit the function
			for _, char := range line {
				switch char {
				case '{':
					bracketDepth++
				case '}':
					bracketDepth--
				}
			}
			
			if bracketDepth <= 0 {
				inFunctionBlock = false
				// Maintain indentation
				indent := len(line) - len(strings.TrimLeft(line, " \t"))
				result.WriteString(line[:indent])
				result.WriteString("}\n")
			}
			continue
		}
		
		// Does this line declare a function and open a block?
		// We look for typical JS/TS function signatures ending with '{'
		isFunc := strings.Contains(trimmed, "function") || 
		           strings.Contains(trimmed, "=>") || 
				   (strings.Contains(trimmed, "(") && strings.Contains(trimmed, ")") && strings.HasSuffix(trimmed, "{"))
				   
		if isFunc && strings.HasSuffix(trimmed, "{") {
			result.WriteString(line)
			result.WriteByte('\n')
			
			// Calculate starting depth for this line
			bracketDepth = 0
			for _, char := range line {
				switch char {
				case '{':
					bracketDepth++
				case '}':
					bracketDepth--
				}
			}
			
			if bracketDepth > 0 {
				inFunctionBlock = true
				indent := len(line) - len(strings.TrimLeft(line, " \t"))
				result.WriteString(strings.Repeat(" ", indent+2))
				result.WriteString("/* body omitted */\n")
			}
			continue
		}
		
		result.WriteString(line)
		result.WriteByte('\n')
	}
	
	// Trim the final trailing newline
	return strings.TrimSuffix(result.String(), "\n")
}
