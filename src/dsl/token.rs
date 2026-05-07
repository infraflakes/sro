#[derive(Debug, Clone, PartialEq)]
#[allow(clippy::enum_variant_names)]
#[allow(clippy::upper_case_acronyms)]
#[allow(dead_code)]
pub enum TokenType {
    EOF,
    Illegal(String),
    Ident(String),
    String(String),
    Backtick(String),
    LBrace,
    RBrace,
    LParen,
    RParen,
    LBracket,
    RBracket,
    Semicolon,
    Colon,
    Comma,
    Dot,
    Equal,
    Assign,
    Dollar,
    Import,
    Shell,
    Sanctuary,
    Var,
    StringKw,
    ShellKw,
    Fn,
    Seq,
    Par,
    Pr,
    Use,
    Log,
    Exec,
    Cd,
    Env,
    PathLit(String),
}

#[derive(Debug, Clone, PartialEq)]
pub struct Token {
    pub ty: TokenType,
    pub line: usize,
    pub col: usize,
}

impl Token {
    pub fn new(ty: TokenType, line: usize, col: usize) -> Self {
        Self { ty, line, col }
    }
}

pub fn lookup_ident(ident: &str) -> TokenType {
    match ident {
        "sanctuary" => TokenType::Sanctuary,
        "import" => TokenType::Import,
        "var" => TokenType::Var,
        "string" => TokenType::StringKw,
        "pr" => TokenType::Pr,
        "fn" => TokenType::Fn,
        "seq" => TokenType::Seq,
        "par" => TokenType::Par,
        "env" => TokenType::Env,
        "log" => TokenType::Log,
        "exec" => TokenType::Exec,
        "cd" => TokenType::Cd,
        "shell" => TokenType::Shell,
        _ => TokenType::Ident(ident.to_string()),
    }
}
