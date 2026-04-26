package token

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Col     int
}

const (
	EOF     TokenType = "EOF"
	ILLEGAL TokenType = "ILLEGAL"

	IDENT      TokenType = "IDENT"
	STRING_LIT TokenType = "STRING"
	PATH_LIT   TokenType = "PATH"
	SHELL_LIT  TokenType = "SHELL_LIT"

	ASSIGN    TokenType = "="
	DECLARE   TokenType = ":="
	LBRACE    TokenType = "{"
	RBRACE    TokenType = "}"
	LBRACKET  TokenType = "["
	RBRACKET  TokenType = "]"
	LPAREN    TokenType = "("
	RPAREN    TokenType = ")"
	COMMA     TokenType = ","
	DOT       TokenType = "."
	SEMICOLON TokenType = ";"
	DOLLAR    TokenType = "$"

	SANCTUARY TokenType = "SANCTUARY"
	IMPORT    TokenType = "IMPORT"
	VAR       TokenType = "VAR"
	PR        TokenType = "PR"
	FN        TokenType = "FN"
	SEQ       TokenType = "SEQ"
	PAR       TokenType = "PAR"
	ENV       TokenType = "ENV"
	LOG       TokenType = "LOG"
	EXEC      TokenType = "EXEC"
	CD        TokenType = "CD"
	SHELL     TokenType = "SHELL"
)

var keywords = map[string]TokenType{
	"sanctuary": SANCTUARY,
	"import":    IMPORT,
	"var":       VAR,
	"pr":        PR,
	"fn":        FN,
	"seq":       SEQ,
	"par":       PAR,
	"env":       ENV,
	"log":       LOG,
	"exec":      EXEC,
	"cd":        CD,
	"shell":     SHELL,
}

func LookupIdent(ident string) TokenType {
	if t, ok := keywords[ident]; ok {
		return t
	}
	return IDENT
}
