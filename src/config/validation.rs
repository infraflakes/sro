use crate::config::error::ConfigError;
use crate::config::merge::merge;
use crate::config::types::Config;
use crate::dsl::ast::{BlockStmt, Expr, FnStmt, Stmt};
use std::collections::HashSet;
use std::path::{Path, PathBuf};

pub fn validate_base(config: &Config) -> Result<(), ConfigError> {
    let mut errs = Vec::new();

    if config.shell.is_empty() {
        errs.push("shell declaration is required".to_string());
    }

    if config.sanctuary.is_empty() {
        errs.push("sanctuary declaration is required".to_string());
    } else if !Path::new(&config.sanctuary).is_absolute() {
        errs.push(format!(
            "sanctuary path must be absolute: {}",
            config.sanctuary
        ));
    }

    let mut dirs = std::collections::HashSet::new();
    for proj in config.projects.values() {
        if proj.url.is_empty() {
            errs.push(format!("project {:?}: url is required", proj.name));
        }
        if proj.dir.is_empty() {
            errs.push(format!("project {:?}: dir is required", proj.name));
        }
        if !dirs.insert(&proj.dir) {
            errs.push(format!(
                "project {:?}: duplicate directory {:?}",
                proj.name, proj.dir
            ));
        }
        match proj.sync.as_str() {
            "clone" | "ignore" => {}
            other => {
                errs.push(format!(
                    "project {:?}: invalid sync value {:?} (expected 'clone' or 'ignore')",
                    proj.name, other
                ));
            }
        }
    }

    if !errs.is_empty() {
        return Err(ConfigError::Validation(errs.join("\n")));
    }

    Ok(())
}

pub fn resolve_use(
    cfg: &mut Config,
    parse_recursive_fn: impl Fn(
        &Path,
        &mut HashSet<PathBuf>,
    ) -> Result<Vec<crate::dsl::ast::Program>, ConfigError>,
) -> Result<(), ConfigError> {
    for proj in cfg.projects.values() {
        if proj.use_file.is_none() {
            continue;
        }

        let use_file = proj.use_file.as_ref().unwrap();

        if proj.sync == "ignore" {
            continue;
        }

        let use_path = PathBuf::from(&cfg.sanctuary).join(&proj.dir).join(use_file);

        if !use_path.exists() {
            return Err(ConfigError::Validation(format!(
                "project {:?}: use file not found: {} (run 'sro sync' first)",
                proj.name,
                use_path.display()
            )));
        }

        let mut visited = HashSet::new();
        let programs = parse_recursive_fn(&use_path, &mut visited)?;

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

        let use_cfg = merge(programs)?;

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

    validate_full(cfg)?;

    Ok(())
}

pub fn validate_full(cfg: &Config) -> Result<(), ConfigError> {
    let mut errs = Vec::new();

    for (name, seq_decl) in &cfg.seqs {
        if let Stmt::SeqDecl { stmts, .. } = seq_decl {
            for stmt in stmts {
                match stmt {
                    BlockStmt::FnCall {
                        fn_name,
                        project_name,
                        ..
                    } => {
                        if !cfg.functions.contains_key(fn_name) {
                            errs.push(format!("seq {:?}: unknown function {:?}", name, fn_name));
                        }
                        if !cfg.projects.contains_key(project_name) {
                            errs.push(format!(
                                "seq {:?}: unknown project {:?}",
                                name, project_name
                            ));
                        }
                    }
                    BlockStmt::SeqRef { seq_name, .. } => {
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
                    BlockStmt::FnCall {
                        fn_name,
                        project_name,
                        ..
                    } => {
                        if !cfg.functions.contains_key(fn_name) {
                            errs.push(format!("par {:?}: unknown function {:?}", name, fn_name));
                        }
                        if !cfg.projects.contains_key(project_name) {
                            errs.push(format!(
                                "par {:?}: unknown project {:?}",
                                name, project_name
                            ));
                        }
                    }
                    BlockStmt::SeqRef { seq_name, .. } => {
                        if !cfg.seqs.contains_key(seq_name) {
                            errs.push(format!("par {:?}: unknown seq {:?}", name, seq_name));
                        }
                    }
                }
            }
        }
    }

    detect_cycles(cfg, &mut errs);
    validate_fn_vars(cfg, &mut errs);

    if !errs.is_empty() {
        return Err(ConfigError::Validation(errs.join("\n")));
    }

    Ok(())
}

fn detect_cycles(cfg: &Config, errs: &mut Vec<String>) {
    fn visit(
        current: &str,
        cfg: &Config,
        path: &mut Vec<String>,
        visited: &mut std::collections::HashSet<String>,
        errs: &mut Vec<String>,
    ) {
        if path.contains(&current.to_string()) {
            let cycle_path: Vec<&str> = path.iter().map(|s| s.as_str()).collect();
            errs.push(format!(
                "cycle detected: {} -> {}",
                cycle_path.join(" -> "),
                current
            ));
            return;
        }
        if !visited.insert(current.to_string()) {
            return;
        }
        path.push(current.to_string());
        if let Some(Stmt::SeqDecl { stmts, .. }) = cfg.seqs.get(current) {
            for stmt in stmts {
                if let BlockStmt::SeqRef { seq_name, .. } = stmt {
                    visit(seq_name, cfg, path, visited, errs);
                }
            }
        }
        path.pop();
    }

    let mut visited = std::collections::HashSet::new();

    for seq_name in cfg.seqs.keys() {
        let mut path = Vec::new();
        visit(seq_name, cfg, &mut path, &mut visited, errs);
    }

    for par_decl in cfg.pars.values() {
        if let Stmt::ParDecl { stmts, .. } = par_decl {
            for stmt in stmts {
                if let BlockStmt::SeqRef { seq_name, .. } = stmt {
                    let mut path = vec!["(via par)".to_string()];
                    visit(seq_name, cfg, &mut path, &mut visited, errs);
                }
            }
        }
    }
}

fn validate_fn_vars(cfg: &Config, errs: &mut Vec<String>) {
    use std::collections::HashSet;

    fn validate_expr(expr: &Expr, fn_name: &str, scope: &HashSet<String>, errs: &mut Vec<String>) {
        match expr {
            Expr::VarRef { name, .. } => {
                if !scope.contains(name) {
                    errs.push(format!("fn {:?}: undefined variable ${}", fn_name, name));
                }
            }
            Expr::BacktickLit { parts, .. } => {
                for part in parts {
                    if part.is_var {
                        let var_name = part.value.trim_start_matches('$');
                        if !scope.contains(var_name) {
                            errs.push(format!(
                                "fn {:?}: undefined variable ${}",
                                fn_name, var_name
                            ));
                        }
                    }
                }
            }
        }
    }

    fn validate_fn_body(
        fn_name: &str,
        body: &[FnStmt],
        scope: &mut HashSet<String>,
        errs: &mut Vec<String>,
    ) {
        for stmt in body {
            match stmt {
                FnStmt::VarDecl { name, value, .. } => {
                    validate_expr(value, fn_name, scope, errs);
                    if !scope.insert(name.clone()) {
                        errs.push(format!("fn {:?}: duplicate variable {:?}", fn_name, name));
                    }
                }
                FnStmt::Log { value, .. } => validate_expr(value, fn_name, scope, errs),
                FnStmt::Exec { value, .. } => validate_expr(value, fn_name, scope, errs),
                FnStmt::Cd { .. } => {}
                FnStmt::EnvBlock { pairs, body, .. } => {
                    let mut block_scope = scope.clone();
                    for pair in pairs {
                        validate_expr(&pair.value, fn_name, scope, errs);
                        block_scope.insert(pair.key.clone());
                    }
                    validate_fn_body(fn_name, body, &mut block_scope, errs);
                }
            }
        }
    }

    for (fn_name, decl) in &cfg.functions {
        if let Stmt::FnDecl { body, .. } = decl {
            let mut scope: HashSet<String> = cfg.vars.keys().cloned().collect();
            validate_fn_body(fn_name, body, &mut scope, errs);
        }
    }
}
