#[derive(Debug, Clone)]
#[allow(dead_code)]
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
        #[allow(dead_code)]
        span: Span,
        parts: Vec<TemplatePart>,
    },
    VarRef {
        #[allow(dead_code)]
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
#[allow(clippy::enum_variant_names)]
pub enum Stmt {
    ShellDecl {
        #[allow(dead_code)]
        span: Span,
        value: String,
    },
    SanctuaryDecl {
        #[allow(dead_code)]
        span: Span,
        value: Expr,
    },
    ImportDecl {
        #[allow(dead_code)]
        span: Span,
        paths: Vec<String>,
    },
    VarDecl {
        #[allow(dead_code)]
        span: Span,
        var_type: VarType,
        name: String,
        value: Expr,
    },
    ProjectDecl {
        #[allow(dead_code)]
        span: Span,
        name: String,
        fields: Vec<ProjectField>,
    },
    FnDecl {
        #[allow(dead_code)]
        span: Span,
        name: String,
        body: Vec<FnStmt>,
    },
    SeqDecl {
        #[allow(dead_code)]
        span: Span,
        name: String,
        stmts: Vec<BlockStmt>,
    },
    ParDecl {
        #[allow(dead_code)]
        span: Span,
        name: String,
        stmts: Vec<BlockStmt>,
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
        #[allow(dead_code)]
        span: Span,
        value: Expr,
    },
    Exec {
        #[allow(dead_code)]
        span: Span,
        value: Expr,
    },
    Cd {
        #[allow(dead_code)]
        span: Span,
        arg: String,
    },
    VarDecl {
        #[allow(dead_code)]
        span: Span,
        var_type: VarType,
        name: String,
        value: Expr,
    },
    EnvBlock {
        #[allow(dead_code)]
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
pub enum BlockStmt {
    FnCall {
        #[allow(dead_code)]
        span: Span,
        fn_name: String,
        project_name: String,
    },
    SeqRef {
        #[allow(dead_code)]
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
