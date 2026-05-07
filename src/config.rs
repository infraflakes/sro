use crate::ast::{Program, Expr, Stmt, VarType, SeqStmt, ParStmt};
use crate::lexer::Lexer;
use crate::parser::Parser;
use std::collections::HashMap;
use std::path::{Path, PathBuf};

#[derive(Debug)]
pub enum ConfigError {
    Io(std::io::Error),
    Parse(String),
    CircularImport(String),
    Validation(String),
}

impl std::fmt::Display for ConfigError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            ConfigError::Io(e) => write!(f, "IO error: {}", e),
            ConfigError::Parse(s) => write!(f, "Parse error: {}", s),
            ConfigError::CircularImport(s) => write!(f, "Circular import detected: {}", s),
            ConfigError::Validation(s) => write!(f, "Validation error: {}", s),
        }
    }
}

impl std::error::Error for ConfigError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            ConfigError::Io(e) => Some(e),
            _ => None,
        }
    }
}

impl From<std::io::Error> for ConfigError {
    fn from(e: std::io::Error) -> Self {
        ConfigError::Io(e)
    }
}

#[derive(Debug, Clone)]
pub struct Project {
    pub name: String,
    pub url: String,
    pub dir: String,
    pub sync: String,
    pub use_file: Option<String>,
    pub branch: String,
}

#[derive(Debug, Clone)]
pub struct Config {
    pub shell: String,
    pub sanctuary: String,
    pub projects: HashMap<String, Project>,
    pub functions: HashMap<String, Stmt>,
    pub seqs: HashMap<String, Stmt>,
    pub pars: HashMap<String, Stmt>,
    pub vars: HashMap<String, String>,
}

pub fn load(entry_path: &Path) -> Result<Config, ConfigError> {
    let abs_path = if entry_path.is_absolute() {
        entry_path.to_path_buf()
    } else {
        std::env::current_dir()
            .map_err(ConfigError::Io)?
            .join(entry_path)
    };
    
    
    // Don't canonicalize - just use the absolute path as-is
    // This avoids issues with symlinks or missing directories
    
    let mut visited = HashMap::new();
    let programs = parse_recursive(&abs_path, &mut visited)?;
    
    let mut config = merge(programs)?;
    validate_base(&config)?;
    
    resolve_use(&mut config)?;
    
    Ok(config)
}

fn parse_recursive(file_path: &Path, visited: &mut HashMap<PathBuf, bool>) -> Result<Vec<Program>, ConfigError> {
    // Use the path as-is without canonicalization
    // This avoids issues with symlinks or missing directories
    let abs_path = if file_path.is_absolute() {
        file_path.to_path_buf()
    } else {
        std::env::current_dir()
            .map_err(ConfigError::Io)?
            .join(file_path)
    };
    
    if visited.contains_key(&abs_path) {
        return Err(ConfigError::CircularImport(abs_path.display().to_string()));
    }
    visited.insert(abs_path.clone(), true);
    
    let data = std::fs::read_to_string(&abs_path)
        .map_err(|e| ConfigError::Io(std::io::Error::new(e.kind(), format!("Failed to read {}: {}", abs_path.display(), e))))?;
    
    let lexer = Lexer::new(data);
    let mut parser = Parser::new(lexer);
    let program = parser.parse()
        .map_err(|errors| {
            let error_msgs: Vec<String> = errors.iter()
                .map(|e| format!("{:?}", e))
                .collect();
            ConfigError::Parse(format!("Parse errors in {}:\n{}", 
                abs_path.display(), 
                error_msgs.join("\n")))
        })?;
    
    let mut results = Vec::new();
    
    // Process imports first (depth-first)
    let base_dir = abs_path.parent()
        .unwrap_or_else(|| Path::new("."));
    
    for stmt in &program.stmts {
        if let Stmt::ImportDecl { paths, .. } = stmt {
            for rel_path in paths {
                let import_abs = base_dir.join(rel_path);
                let imported = parse_recursive(&import_abs, visited)?;
                results.extend(imported);
            }
        }
    }
    
    // Then add the current program
    results.push(program);
    Ok(results)
}

fn merge(programs: Vec<Program>) -> Result<Config, ConfigError> {
    let mut shell = String::new();
    let mut sanctuary_expr: Option<Expr> = None;
    let mut projects: HashMap<String, Project> = HashMap::new();
    let mut functions: HashMap<String, Stmt> = HashMap::new();
    let mut seqs: HashMap<String, Stmt> = HashMap::new();
    let mut pars: HashMap<String, Stmt> = HashMap::new();
    let mut vars: HashMap<String, String> = HashMap::new();
    
    // First pass: collect shell declaration
    for program in &programs {
        for stmt in &program.stmts {
            if let Stmt::ShellDecl { value, .. } = stmt
                && shell.is_empty() {
                    shell = value.clone();
                }
        }
    }
    
    // Second pass: collect variables (with shell execution for shell-type vars)
    for program in &programs {
        for stmt in &program.stmts {
            if let Stmt::VarDecl { name, value, var_type, .. } = stmt
                && let Expr::BacktickLit { parts, .. } = value {
                    let resolved: String = parts.iter()
                        .filter(|p| !p.is_var)
                        .map(|p| p.value.as_str())
                        .collect();
                    
                    let final_value = if var_type == &VarType::Shell {
                        if shell.is_empty() {
                            return Err(ConfigError::Validation("shell must be declared before using shell variables".to_string()));
                        }
                        execute_shell(&shell, &resolved)?
                    } else {
                        resolved
                    };
                    
                    vars.insert(name.clone(), final_value);
                }
        }
    }
    
    // Third pass: process other declarations with variable resolution
    for program in programs {
        for stmt in program.stmts {
            match stmt {
                Stmt::ShellDecl { .. } => {
                    // Already handled in first pass
                }
                Stmt::SanctuaryDecl { value, .. } => {
                    if sanctuary_expr.is_none() {
                        sanctuary_expr = Some(value);
                    }
                }
                Stmt::ProjectDecl { name, fields, .. } => {
                    let mut project = Project {
                        name: name.clone(),
                        url: String::new(),
                        dir: String::new(),
                        sync: "clone".to_string(),
                        use_file: None,
                        branch: String::new(),
                    };
                    
                    for field in fields {
                        match field.key.as_str() {
                            "url" => {
                                if let Expr::BacktickLit { parts, .. } = &field.value {
                                    project.url = parts.iter().map(|p| p.value.clone()).collect();
                                }
                            }
                            "dir" => {
                                if let Expr::BacktickLit { parts, .. } = &field.value {
                                    project.dir = parts.iter().map(|p| p.value.clone()).collect();
                                }
                            }
                            "sync" => {
                                if let Expr::BacktickLit { parts, .. } = &field.value {
                                    project.sync = parts.iter().map(|p| p.value.clone()).collect();
                                }
                            }
                            "use" => {
                                if let Expr::BacktickLit { parts, .. } = &field.value {
                                    project.use_file = Some(parts.iter().map(|p| p.value.clone()).collect());
                                }
                            }
                            "branch" => {
                                if let Expr::BacktickLit { parts, .. } = &field.value {
                                    project.branch = parts.iter().map(|p| p.value.clone()).collect();
                                }
                            }
                            _ => {}
                        }
                    }
                    
                    projects.insert(name, project);
                }
                Stmt::FnDecl { ref name, .. } => {
                    functions.insert(name.clone(), stmt);
                }
                Stmt::SeqDecl { ref name, .. } => {
                    seqs.insert(name.clone(), stmt);
                }
                Stmt::ParDecl { ref name, .. } => {
                    pars.insert(name.clone(), stmt);
                }
                _ => {}
            }
        }
    }
    
    // Resolve sanctuary expression with variables
    let sanctuary = if let Some(expr) = sanctuary_expr {
        resolve_expr(&expr, &vars)?
    } else {
        String::new()
    };
    
    Ok(Config {
        shell,
        sanctuary,
        projects,
        functions,
        seqs,
        pars,
        vars,
    })
}

fn resolve_expr(expr: &Expr, vars: &HashMap<String, String>) -> Result<String, ConfigError> {
    match expr {
        Expr::BacktickLit { parts, .. } => {
            let mut result = String::new();
            for part in parts {
                if part.is_var {
                    let var_name = part.value.trim_start_matches('$');
                    if let Some(value) = vars.get(var_name) {
                        result.push_str(value);
                    }
                } else {
                    result.push_str(&part.value);
                }
            }
            Ok(result)
        }
        Expr::VarRef { name, .. } => {
            if let Some(value) = vars.get(name) {
                Ok(value.clone())
            } else {
                Ok(format!("${}", name))
            }
        }
    }
}

fn execute_shell(shell: &str, command: &str) -> Result<String, ConfigError> {
    use std::process::Command;
    
    let output = Command::new(shell)
        .arg("-c")
        .arg(command)
        .output()
        .map_err(|e| ConfigError::Validation(format!("shell execution failed: {}", e)))?;
    
    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(ConfigError::Validation(format!("shell command failed: {}", stderr)));
    }
    
    let result = String::from_utf8_lossy(&output.stdout);
    Ok(result.trim_end().to_string())
}

fn validate_base(config: &Config) -> Result<(), ConfigError> {
    if config.shell.is_empty() {
        return Err(ConfigError::Validation("shell declaration is required".to_string()));
    }
    
    if config.sanctuary.is_empty() {
        return Err(ConfigError::Validation("sanctuary declaration is required".to_string()));
    }
    
    Ok(())
}

fn resolve_use(cfg: &mut Config) -> Result<(), ConfigError> {
    for proj in cfg.projects.values() {
        if proj.use_file.is_none() {
            continue;
        }
        
        let use_file = proj.use_file.as_ref().unwrap();
        
        // Skip projects with sync=ignore — they may not be cloned locally,
        // so their use file won't exist on disk
        if proj.sync == "ignore" {
            continue;
        }
        
        let use_path = PathBuf::from(&cfg.sanctuary).join(&proj.dir).join(use_file);
        
        // Check if use file exists
        if !use_path.exists() {
            return Err(ConfigError::Validation(format!(
                "project {:?}: use file not found: {} (run 'sro sync' first)",
                proj.name,
                use_path.display()
            )));
        }
        
        // Parse the use file with a fresh visited map
        let mut visited = HashMap::new();
        let programs = parse_recursive(&use_path, &mut visited)?;
        
        // Validate use file doesn't contain sanctuary or pr blocks
        for program in &programs {
            for stmt in &program.stmts {
                match stmt {
                    Stmt::SanctuaryDecl { .. } => {
                        return Err(ConfigError::Validation(format!(
                            "project {:?}: use file {} cannot declare sanctuary",
                            proj.name,
                            use_path.display()
                        )));
                    }
                    Stmt::ProjectDecl { .. } => {
                        return Err(ConfigError::Validation(format!(
                            "project {:?}: use file {} cannot declare projects",
                            proj.name,
                            use_path.display()
                        )));
                    }
                    _ => {}
                }
            }
        }
        
        // Merge use file declarations into config
        let use_cfg = merge(programs)?;
        
        // Merge fn/seq/par/vars from use file
        for (name, fn_decl) in use_cfg.functions {
            if cfg.functions.contains_key(&name) {
                return Err(ConfigError::Validation(format!(
                    "project {:?}: duplicate function {:?} from use file {}",
                    proj.name,
                    name,
                    use_path.display()
                )));
            }
            cfg.functions.insert(name, fn_decl);
        }
        for (name, seq_decl) in use_cfg.seqs {
            if cfg.seqs.contains_key(&name) {
                return Err(ConfigError::Validation(format!(
                    "project {:?}: duplicate seq {:?} from use file {}",
                    proj.name,
                    name,
                    use_path.display()
                )));
            }
            cfg.seqs.insert(name, seq_decl);
        }
        for (name, par_decl) in use_cfg.pars {
            if cfg.pars.contains_key(&name) {
                return Err(ConfigError::Validation(format!(
                    "project {:?}: duplicate par {:?} from use file {}",
                    proj.name,
                    name,
                    use_path.display()
                )));
            }
            cfg.pars.insert(name, par_decl);
        }
        for (name, val) in use_cfg.vars {
            if cfg.vars.contains_key(&name) {
                return Err(ConfigError::Validation(format!(
                    "project {:?}: duplicate var {:?} from use file {}",
                    proj.name,
                    name,
                    use_path.display()
                )));
            }
            cfg.vars.insert(name, val);
        }
    }
    
    // Run full validation after all use files are merged
    validate_full(cfg)?;
    
    Ok(())
}

fn validate_full(cfg: &Config) -> Result<(), ConfigError> {
    let mut errs = Vec::new();
    
    // Check seq/par references
    for (name, seq_decl) in &cfg.seqs {
        if let Stmt::SeqDecl { stmts, .. } = seq_decl {
            for stmt in stmts {
                match stmt {
                    SeqStmt::FnCall { fn_name, project_name, .. } => {
                        if !cfg.functions.contains_key(fn_name) {
                            errs.push(format!("seq {:?}: unknown function {:?}", name, fn_name));
                        }
                        if !cfg.projects.contains_key(project_name) {
                            errs.push(format!("seq {:?}: unknown project {:?}", name, project_name));
                        }
                    }
                    SeqStmt::SeqRef { seq_name, .. } => {
                        if !cfg.seqs.contains_key(seq_name) {
                            errs.push(format!("seq {:?}: unknown seq {:?}", name, seq_name));
                        }
                    }
                }
            }
        }
    }
    
    for (name, par_decl) in &cfg.pars {
        if let Stmt::ParDecl { stmts, .. } = par_decl {
            for stmt in stmts {
                match stmt {
                    ParStmt::FnCall { fn_name, project_name, .. } => {
                        if !cfg.functions.contains_key(fn_name) {
                            errs.push(format!("par {:?}: unknown function {:?}", name, fn_name));
                        }
                        if !cfg.projects.contains_key(project_name) {
                            errs.push(format!("par {:?}: unknown project {:?}", name, project_name));
                        }
                    }
                    ParStmt::SeqRef { seq_name, .. } => {
                        if !cfg.seqs.contains_key(seq_name) {
                            errs.push(format!("par {:?}: unknown seq {:?}", name, seq_name));
                        }
                    }
                }
            }
        }
    }
    
    if !errs.is_empty() {
        return Err(ConfigError::Validation(errs.join("\n")));
    }
    
    Ok(())
}
