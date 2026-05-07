#[derive(Debug, Clone)]
pub struct Span {
    pub line: usize,
    pub col: usize,
}

impl Span {
    pub fn new(line: usize, col: usize) -> Self {
        Self { line, col }
    }
}

#[derive(Debug, Clone)]
pub enum Expr {
    BacktickLit {
        span: Span,
        parts: Vec<TemplatePart>,
    },
    VarRef {
        span: Span,
        name: String,
    },
}

#[derive(Debug, Clone)]
pub struct TemplatePart {
    pub is_var: bool,
    pub value: String,
}

#[derive(Debug, Clone)]
pub enum Stmt {
    ShellDecl {
        span: Span,
        value: String,
    },
    SanctuaryDecl {
        span: Span,
        value: Expr,
    },
    ImportDecl {
        span: Span,
        paths: Vec<String>,
    },
    VarDecl {
        span: Span,
        var_type: VarType,
        name: String,
        value: Expr,
    },
    ProjectDecl {
        span: Span,
        name: String,
        fields: Vec<ProjectField>,
    },
    FnDecl {
        span: Span,
        name: String,
        body: Vec<FnStmt>,
    },
    SeqDecl {
        span: Span,
        name: String,
        stmts: Vec<SeqStmt>,
    },
    ParDecl {
        span: Span,
        name: String,
        stmts: Vec<ParStmt>,
    },
}

#[derive(Debug, Clone, PartialEq)]
pub enum VarType {
    String,
    Shell,
}

#[derive(Debug, Clone)]
pub struct ProjectField {
    pub key: String,
    pub value: Expr,
}

#[derive(Debug, Clone)]
pub enum FnStmt {
    Log {
        span: Span,
        value: Expr,
    },
    Exec {
        span: Span,
        value: Expr,
    },
    Cd {
        span: Span,
        arg: String,
    },
    VarDecl {
        span: Span,
        var_type: VarType,
        name: String,
        value: Expr,
    },
    EnvBlock {
        span: Span,
        pairs: Vec<EnvPair>,
        body: Vec<FnStmt>,
    },
}

#[derive(Debug, Clone)]
pub struct EnvPair {
    pub key: String,
    pub value: Expr,
}

#[derive(Debug, Clone)]
pub enum SeqStmt {
    FnCall {
        span: Span,
        fn_name: String,
        project_name: String,
    },
    SeqRef {
        span: Span,
        seq_name: String,
    },
}

#[derive(Debug, Clone)]
pub enum ParStmt {
    FnCall {
        span: Span,
        fn_name: String,
        project_name: String,
    },
    SeqRef {
        span: Span,
        seq_name: String,
    },
}

#[derive(Debug, Clone)]
pub struct Program {
    pub stmts: Vec<Stmt>,
}

impl Program {
    pub fn new() -> Self {
        Self { stmts: Vec::new() }
    }
}
