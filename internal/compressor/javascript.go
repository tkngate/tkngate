package compressor

import (
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/js"
)

// compressJS performs structural minification for JS/TS using a robust lexer.
// It losslessly drops comments and strips deep function/class method bodies,
// leaving signatures intact to save massive amounts of context tokens.
func compressJS(content string) string {
	l := js.NewLexer(parse.NewInputString(content))

	var result strings.Builder
	// Pre-allocate for performance
	result.Grow(len(content) / 2)

	inOmitBlock := false
	omitDepth := 0

	var lastSigToken js.TokenType
	var lastSigText string

	// Stack to remember what token preceded a '('
	type parenCtx struct {
		token js.TokenType
		text  string
	}
	var parenStack []parenCtx
	var lastParenOwner parenCtx

	awaitingFuncBody := false

	for {
		tt, textBytes := l.Next()
		if tt == js.ErrorToken {
			break
		}

		text := string(textBytes)

		// 1. Drop comments unconditionally
		if tt == js.CommentToken {
			continue
		}

		// 2. Handle block omitting
		if inOmitBlock {
			if tt == js.OpenBraceToken {
				omitDepth++
			} else if tt == js.CloseBraceToken {
				omitDepth--
				if omitDepth == 0 {
					inOmitBlock = false
					result.WriteString("}") // close the block
				}
			}
			continue
		}

		// 3. Track Parens to distinguish functions from control flow
		if tt == js.OpenParenToken {
			parenStack = append(parenStack, parenCtx{lastSigToken, lastSigText})
			awaitingFuncBody = false
		} else if tt == js.CloseParenToken {
			if len(parenStack) > 0 {
				lastParenOwner = parenStack[len(parenStack)-1]
				parenStack = parenStack[:len(parenStack)-1]
			} else {
				lastParenOwner = parenCtx{}
			}
			
			owner := lastParenOwner.text
			if owner != "if" && owner != "for" && owner != "while" && owner != "switch" && owner != "catch" && owner != "with" {
				awaitingFuncBody = true
			}
		} else if tt == js.ArrowToken {
			awaitingFuncBody = true
		} else if tt == js.SemicolonToken || tt == js.EqToken || tt == js.CommaToken || tt == js.CloseBraceToken ||
			tt == js.LetToken || tt == js.ConstToken || tt == js.VarToken || tt == js.ClassToken ||
			tt == js.IfToken || tt == js.ForToken || tt == js.WhileToken || tt == js.SwitchToken {
			awaitingFuncBody = false
		}

		// 4. Detect function/method bodies
		if tt == js.OpenBraceToken {
			if awaitingFuncBody {
				inOmitBlock = true
				omitDepth = 1
				awaitingFuncBody = false
				result.WriteString("{ /* body omitted */ ")
				continue
			}
		}

		// Write the token
		result.WriteString(text)

		// Update significant token
		if tt != js.WhitespaceToken && tt != js.LineTerminatorToken {
			lastSigToken = tt
			lastSigText = text
		}
	}

	return strings.TrimSpace(result.String())
}
