package ast

import "github.com/infraflakes/sro/internal/dsl/token"

// FnStmt interface (statements inside fn/env blocks)
type FnStmt interface {
	Stmt
	fnStmt()
}

type LogStmt struct {
	Token token.Token
	Value Expr
}

func (ls *LogStmt) TokenLiteral() string { return ls.Token.Literal }
func (ls *LogStmt) Pos() (int, int)      { return ls.Token.Line, ls.Token.Col }
func (ls *LogStmt) stmtNode()            {}
func (ls *LogStmt) fnStmt()              {}

type ExecStmt struct {
	Token token.Token
	Value Expr
}

func (es *ExecStmt) TokenLiteral() string { return es.Token.Literal }
func (es *ExecStmt) Pos() (int, int)      { return es.Token.Line, es.Token.Col }
func (es *ExecStmt) stmtNode()            {}
func (es *ExecStmt) fnStmt()              {}

type CdStmt struct {
	Token token.Token
	Arg   string
}

func (cs *CdStmt) TokenLiteral() string { return cs.Token.Literal }
func (cs *CdStmt) Pos() (int, int)      { return cs.Token.Line, cs.Token.Col }
func (cs *CdStmt) stmtNode()            {}
func (cs *CdStmt) fnStmt()              {}

type EnvBlock struct {
	Token token.Token
	Pairs []EnvPair
	Body  []FnStmt
}

func (eb *EnvBlock) TokenLiteral() string { return eb.Token.Literal }
func (eb *EnvBlock) Pos() (int, int)      { return eb.Token.Line, eb.Token.Col }
func (eb *EnvBlock) stmtNode()            {}
func (eb *EnvBlock) fnStmt()              {}

type EnvPair struct {
	Key   string
	Value Expr
}
