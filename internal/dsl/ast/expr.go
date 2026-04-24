package ast

import "github.com/infraflakes/sro/internal/dsl/token"

type StringLit struct {
	Token token.Token
	Value string
}

func (sl *StringLit) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLit) Pos() (int, int)      { return sl.Token.Line, sl.Token.Col }
func (sl *StringLit) exprNode()            {}

type VarRef struct {
	Token token.Token
	Name  string
}

func (vr *VarRef) TokenLiteral() string { return vr.Token.Literal }
func (vr *VarRef) Pos() (int, int)      { return vr.Token.Line, vr.Token.Col }
func (vr *VarRef) exprNode()            {}
