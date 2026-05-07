use crate::config::error::ConfigError;
use crate::config::merge::merge;
use crate::config::types::Config;
use crate::dsl::ast::{ParStmt, SeqStmt, Stmt};
use std::collections::HashMap;
use std::path::{Path, PathBuf};

pub fn validate_base(config: &Config) -> Result<(), ConfigError> {
    if config.shell.is_empty() {
        return Err(ConfigError::Validation(
            "shell declaration is required".to_string(),
        ));
    }

    if config.sanctuary.is_empty() {
        return Err(ConfigError::Validation(
            "sanctuary declaration is required".to_string(),
        ));
    }

    Ok(())
}

pub fn resolve_use(
    cfg: &mut Config,
    parse_recursive_fn: impl Fn(
        &Path,
        &mut HashMap<PathBuf, bool>,
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

        let mut visited = HashMap::new();
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
                    SeqStmt::FnCall {
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
                    ParStmt::FnCall {
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
