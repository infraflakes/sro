package ast

import "github.com/infraflakes/sro/token"

type Node interface {
	TokenLiteral() string
	Pos() (line, col int)
}

type Program struct {
	Statements []Node
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) Pos() (line, col int) {
	if len(p.Statements) > 0 {
		return p.Statements[0].Pos()
	}
	return 0, 0
}

// Expression interface
type Expr interface {
	Node
	exprNode()
}

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

// Statement interface
type Stmt interface {
	Node
	stmtNode()
}

// Statement types
type SanctuaryDecl struct {
	Token token.Token
	Value string
}

func (sd *SanctuaryDecl) TokenLiteral() string { return sd.Token.Literal }
func (sd *SanctuaryDecl) Pos() (int, int)      { return sd.Token.Line, sd.Token.Col }
func (sd *SanctuaryDecl) stmtNode()            {}

type ImportDecl struct {
	Token token.Token
	Paths []string
}

func (id *ImportDecl) TokenLiteral() string { return id.Token.Literal }
func (id *ImportDecl) Pos() (int, int)      { return id.Token.Line, id.Token.Col }
func (id *ImportDecl) stmtNode()            {}

type VarDecl struct {
	Token token.Token
	Name  string
	Value Expr
}

func (vd *VarDecl) TokenLiteral() string { return vd.Token.Literal }
func (vd *VarDecl) Pos() (int, int)      { return vd.Token.Line, vd.Token.Col }
func (vd *VarDecl) stmtNode()            {}
func (vd *VarDecl) fnStmt()              {}

type ProjectDecl struct {
	Token  token.Token
	Name   string
	Fields []ProjectField
}

func (pd *ProjectDecl) TokenLiteral() string { return pd.Token.Literal }
func (pd *ProjectDecl) Pos() (int, int)      { return pd.Token.Line, pd.Token.Col }
func (pd *ProjectDecl) stmtNode()            {}

type ProjectField struct {
	Key   string
	Value string
}

type FnDecl struct {
	Token token.Token
	Name  string
	Body  []FnStmt
}

func (fd *FnDecl) TokenLiteral() string { return fd.Token.Literal }
func (fd *FnDecl) Pos() (int, int)      { return fd.Token.Line, fd.Token.Col }
func (fd *FnDecl) stmtNode()            {}

type SeqDecl struct {
	Token token.Token
	Name  string
	Stmts []SeqStmt
}

func (sd *SeqDecl) TokenLiteral() string { return sd.Token.Literal }
func (sd *SeqDecl) Pos() (int, int)      { return sd.Token.Line, sd.Token.Col }
func (sd *SeqDecl) stmtNode()            {}

type ParDecl struct {
	Token token.Token
	Name  string
	Stmts []ParStmt
}

func (pd *ParDecl) TokenLiteral() string { return pd.Token.Literal }
func (pd *ParDecl) Pos() (int, int)      { return pd.Token.Line, pd.Token.Col }
func (pd *ParDecl) stmtNode()            {}

// FnStmt interface (statements inside fn/env blocks)
type FnStmt interface {
	Stmt
	fnStmt()
}

type LogStmt struct {
	Token token.Token
	Args  []Expr
}

func (ls *LogStmt) TokenLiteral() string { return ls.Token.Literal }
func (ls *LogStmt) Pos() (int, int)      { return ls.Token.Line, ls.Token.Col }
func (ls *LogStmt) stmtNode()            {}
func (ls *LogStmt) fnStmt()              {}

type ExecStmt struct {
	Token token.Token
	Args  []Expr
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
	Value string
}

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
