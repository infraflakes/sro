package ast

type Node interface {
	TokenLiteral() string
	Pos() (line, col int)
}

type Program struct {
	Statements []Stmt
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

// Statement interface
type Stmt interface {
	Node
	stmtNode()
}
