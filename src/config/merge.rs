use crate::config::error::ConfigError;
use crate::config::types::{Config, Project};
use crate::dsl::ast::{Expr, Program, Stmt, VarType};
use std::collections::HashMap;
use std::process::Command;

pub(crate) fn merge(programs: Vec<Program>) -> Result<Config, ConfigError> {
    let mut shell = String::new();
    let mut sanctuary_expr: Option<Expr> = None;
    let mut projects: HashMap<String, Project> = HashMap::new();
    let mut global_vars: HashMap<String, String> = HashMap::new();

    for program in &programs {
        for stmt in &program.stmts {
            if let Stmt::ShellDecl { value } = stmt {
                if !shell.is_empty() {
                    return Err(ConfigError::Validation(
                        "duplicate shell declaration".to_string(),
                    ));
                }
                shell = value.clone();
            }
        }
    }

    for program in &programs {
        for stmt in &program.stmts {
            if let Stmt::VarDecl {
                name,
                value,
                var_type,
            } = stmt
            {
                if global_vars.contains_key(name) {
                    return Err(ConfigError::Validation(format!(
                        "duplicate variable: {}",
                        name
                    )));
                }

                let resolved = value
                    .resolve(&global_vars)
                    .map_err(ConfigError::Validation)?;

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

                global_vars.insert(name.clone(), final_value);
            }
        }
    }

    for program in programs {
        for stmt in program.stmts {
            match stmt {
                Stmt::ShellDecl { .. } => {}
                Stmt::SanctuaryDecl { value } => {
                    if sanctuary_expr.is_some() {
                        return Err(ConfigError::Validation(
                            "duplicate sanctuary declaration".to_string(),
                        ));
                    }
                    sanctuary_expr = Some(value);
                }
                Stmt::ProjectDecl {
                    name, fields, body, ..
                } => {
                    if projects.contains_key(&name) {
                        return Err(ConfigError::Validation(format!(
                            "duplicate project: {}",
                            name
                        )));
                    }

                    let mut project = Project {
                        name: name.clone(),
                        url: String::new(),
                        dir: String::new(),
                        sync: "clone".to_string(),
                        use_file: None,
                        branch: String::new(),
                        vars: HashMap::new(),
                        functions: HashMap::new(),
                        seqs: HashMap::new(),
                        pars: HashMap::new(),
                    };

                    for field in &fields {
                        let value = field
                            .value
                            .resolve(&global_vars)
                            .map_err(ConfigError::Validation)?;
                        match field.key.as_str() {
                            "url" => project.url = value,
                            "dir" => project.dir = value,
                            "sync" => project.sync = value,
                            "use" => {
                                if !value.is_empty() {
                                    project.use_file = Some(value);
                                }
                            }
                            "branch" => project.branch = value,
                            _ => {
                                return Err(ConfigError::Validation(format!(
                                    "unknown project field: {}",
                                    field.key
                                )));
                            }
                        }
                    }

                    for body_stmt in body {
                        merge_project_body_stmt(&mut project, body_stmt, &shell)?;
                    }

                    projects.insert(name, project);
                }
                _ => {}
            }
        }
    }

    let sanctuary = match sanctuary_expr {
        Some(ref expr) => expr
            .resolve(&global_vars)
            .map_err(ConfigError::Validation)?,
        None => String::new(),
    };

    Ok(Config {
        shell,
        sanctuary,
        projects,
        vars: global_vars,
    })
}

pub(crate) fn merge_project_body_stmt(
    project: &mut Project,
    stmt: Stmt,
    shell: &str,
) -> Result<(), ConfigError> {
    match stmt {
        Stmt::VarDecl {
            name,
            value,
            var_type,
        } => {
            if project.vars.contains_key(&name) {
                return Err(ConfigError::Validation(format!(
                    "duplicate variable in project '{}': {}",
                    project.name, name
                )));
            }

            let resolved = value
                .resolve(&project.vars)
                .map_err(ConfigError::Validation)?;

            let final_value = if var_type == VarType::Shell {
                if shell.is_empty() {
                    return Err(ConfigError::Validation(
                        "shell must be declared before using shell variables".to_string(),
                    ));
                }
                execute_shell(shell, &resolved)?
            } else {
                resolved
            };

            project.vars.insert(name, final_value);
        }
        Stmt::FnDecl { name, body, .. } => {
            if project.functions.contains_key(&name) {
                return Err(ConfigError::Validation(format!(
                    "duplicate function in project '{}': {}",
                    project.name, name
                )));
            }
            project.functions.insert(name, body);
        }
        Stmt::SeqDecl { name, fns, .. } => {
            if project.seqs.contains_key(&name) {
                return Err(ConfigError::Validation(format!(
                    "duplicate seq in project '{}': {}",
                    project.name, name
                )));
            }
            project.seqs.insert(name, fns);
        }
        Stmt::ParDecl { name, fns, .. } => {
            if project.pars.contains_key(&name) {
                return Err(ConfigError::Validation(format!(
                    "duplicate par in project '{}': {}",
                    project.name, name
                )));
            }
            project.pars.insert(name, fns);
        }
        _ => {}
    }
    Ok(())
}

pub(crate) fn execute_shell(shell: &str, command: &str) -> Result<String, ConfigError> {
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

    let stdout = String::from_utf8_lossy(&output.stdout);
    Ok(stdout.trim_end().to_string())
}
