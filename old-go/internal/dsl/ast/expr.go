package ast

import "github.com/infraflakes/sro/internal/dsl/token"

// TemplatePart represents a segment of an interpolated backtick string.
// If IsVar is true, Value is a variable name. Otherwise, Value is literal text.
type TemplatePart struct {
	IsVar bool
	Value string
}

type BacktickLit struct {
	Token token.Token
	Parts []TemplatePart
}

func (bl *BacktickLit) TokenLiteral() string { return bl.Token.Literal }
func (bl *BacktickLit) Pos() (int, int)      { return bl.Token.Line, bl.Token.Col }
func (bl *BacktickLit) exprNode()            {}

type VarRef struct {
	Token token.Token
	Name  string
}

func (vr *VarRef) TokenLiteral() string { return vr.Token.Literal }
func (vr *VarRef) Pos() (int, int)      { return vr.Token.Line, vr.Token.Col }
func (vr *VarRef) exprNode()            {}
