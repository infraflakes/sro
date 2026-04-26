package ast

import "github.com/infraflakes/sro/internal/dsl/token"

type BacktickLit struct {
	Token token.Token
	Value string
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
