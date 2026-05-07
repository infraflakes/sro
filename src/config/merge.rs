use crate::config::error::ConfigError;
use crate::config::types::{Config, Project};
use crate::dsl::ast::{Expr, Program, Stmt, VarType};
use std::collections::HashMap;

pub fn merge(programs: Vec<Program>) -> Result<Config, ConfigError> {
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
                && shell.is_empty()
            {
                shell = value.clone();
            }
        }
    }

    // Second pass: collect variables (with shell execution for shell-type vars)
    for program in &programs {
        for stmt in &program.stmts {
            if let Stmt::VarDecl {
                name,
                value,
                var_type,
                ..
            } = stmt
                && let Expr::BacktickLit { parts, .. } = value
            {
                let resolved: String = parts
                    .iter()
                    .filter(|p| !p.is_var)
                    .map(|p| p.value.as_str())
                    .collect();

                let final_value = if var_type == &VarType::Shell {
                    if shell.is_empty() {
                        return Err(ConfigError::Validation(
                            "shell must be declared before using shell variables".to_string(),
                        ));
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
                                    project.use_file =
                                        Some(parts.iter().map(|p| p.value.clone()).collect());
                                }
                            }
                            "branch" => {
                                if let Expr::BacktickLit { parts, .. } = &field.value {
                                    project.branch =
                                        parts.iter().map(|p| p.value.clone()).collect();
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

pub fn resolve_expr(expr: &Expr, vars: &HashMap<String, String>) -> Result<String, ConfigError> {
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

pub fn execute_shell(shell: &str, command: &str) -> Result<String, ConfigError> {
    use std::process::Command;

    let output = Command::new(shell)
        .arg("-c")
        .arg(command)
        .output()
        .map_err(|e| ConfigError::Validation(format!("shell execution failed: {}", e)))?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(ConfigError::Validation(format!(
            "shell command failed: {}",
            stderr
        )));
    }

    let result = String::from_utf8_lossy(&output.stdout);
    Ok(result.trim_end().to_string())
}
