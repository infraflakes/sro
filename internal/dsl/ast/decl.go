package ast

import "github.com/infraflakes/sro/internal/dsl/token"

// Statement types
type SanctuaryDecl struct {
	Token token.Token
	Value Expr
}

func (sd *SanctuaryDecl) TokenLiteral() string { return sd.Token.Literal }
func (sd *SanctuaryDecl) Pos() (int, int)      { return sd.Token.Line, sd.Token.Col }
func (sd *SanctuaryDecl) stmtNode()            {}

type ShellDecl struct {
	Token token.Token
	Value string
}

func (sd *ShellDecl) TokenLiteral() string { return sd.Token.Literal }
func (sd *ShellDecl) Pos() (int, int)      { return sd.Token.Line, sd.Token.Col }
func (sd *ShellDecl) stmtNode()            {}

type ImportDecl struct {
	Token token.Token
	Paths []string
}

func (id *ImportDecl) TokenLiteral() string { return id.Token.Literal }
func (id *ImportDecl) Pos() (int, int)      { return id.Token.Line, id.Token.Col }
func (id *ImportDecl) stmtNode()            {}

type VarDecl struct {
	Token   token.Token
	VarType string // "string" or "shell"
	Name    string
	Value   Expr
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
	Value Expr
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
