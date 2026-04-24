package ast

import "github.com/infraflakes/sro/internal/dsl/token"

// SeqStmt interface (statements inside seq/par blocks)
type SeqStmt interface {
	Stmt
	seqStmt()
}

type ParStmt interface {
	Stmt
	parStmt()
}

type FnCall struct {
	Token       token.Token
	FnName      string
	ProjectName string
}

func (fc *FnCall) TokenLiteral() string { return fc.Token.Literal }
func (fc *FnCall) Pos() (int, int)      { return fc.Token.Line, fc.Token.Col }
func (fc *FnCall) stmtNode()            {}
func (fc *FnCall) seqStmt()             {}
func (fc *FnCall) parStmt()             {}

type SeqRef struct {
	Token   token.Token
	SeqName string
}

func (sr *SeqRef) TokenLiteral() string { return sr.Token.Literal }
func (sr *SeqRef) Pos() (int, int)      { return sr.Token.Line, sr.Token.Col }
func (sr *SeqRef) stmtNode()            {}
func (sr *SeqRef) seqStmt()             {}
func (sr *SeqRef) parStmt()             {}
