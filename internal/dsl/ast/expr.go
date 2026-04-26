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

type ShellExec struct {
	Token   token.Token
	Command string
}

func (se *ShellExec) TokenLiteral() string { return se.Token.Literal }
func (se *ShellExec) Pos() (int, int)      { return se.Token.Line, se.Token.Col }
func (se *ShellExec) exprNode()            {}
