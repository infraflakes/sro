package parser

import (
	"fmt"
	"strings"

	"github.com/infraflakes/sro/internal/dsl/ast"
	"github.com/infraflakes/sro/internal/dsl/token"
)

// parseBacktickParts splits raw backtick content into TemplateParts for interpolation.
func parseBacktickParts(raw string, line, col int) ([]ast.TemplatePart, error) {
	var parts []ast.TemplatePart
	i := 0
	for i < len(raw) {
		// Find next ${
		idx := strings.Index(raw[i:], "${")
		if idx == -1 {
			// Rest is literal text
			parts = append(parts, ast.TemplatePart{IsVar: false, Value: raw[i:]})
			break
		}
		// Add literal text before ${
		if idx > 0 {
			parts = append(parts, ast.TemplatePart{IsVar: false, Value: raw[i : i+idx]})
		}
		// Find closing }
		i += idx + 2 // skip past ${
		end := strings.Index(raw[i:], "}")
		if end == -1 {
			return nil, fmt.Errorf("unterminated ${} at %d:%d", line, col)
		}
		name := raw[i : i+end]
		if name == "" {
			return nil, fmt.Errorf("empty ${} at %d:%d", line, col)
		}
		parts = append(parts, ast.TemplatePart{IsVar: true, Value: name})
		i += end + 1 // skip past }
	}
	if len(parts) == 0 {
		parts = append(parts, ast.TemplatePart{IsVar: false, Value: ""})
	}
	return parts, nil
}

// parseExpr parses a single expression (backtick literal or variable reference).
func (p *Parser) parseExpr() ast.Expr {
	switch p.curToken.Type {
	case token.BACKTICK:
		tok := p.curToken
		parts, err := parseBacktickParts(p.curToken.Literal, p.curToken.Line, p.curToken.Col)
		if err != nil {
			p.errors = append(p.errors, ParseError{
				Message: err.Error(),
				Line:    p.curToken.Line,
				Col:     p.curToken.Col,
			})
			return nil
		}
		p.nextToken() // consume BACKTICK
		return &ast.BacktickLit{Token: tok, Parts: parts}
	case token.DOLLAR:
		p.nextToken()
		if p.curToken.Type != token.IDENT {
			p.errors = append(p.errors, ParseError{
				Message: "expected identifier after $",
				Line:    p.curToken.Line,
				Col:     p.curToken.Col,
			})
			return nil
		}
		tok := p.curToken
		p.nextToken() // consume IDENT
		return &ast.VarRef{Token: tok, Name: tok.Literal}
	default:
		p.errors = append(p.errors, ParseError{
			Message: "expected backtick literal or variable reference",
			Line:    p.curToken.Line,
			Col:     p.curToken.Col,
		})
		return nil
	}
}
