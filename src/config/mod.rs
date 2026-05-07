pub mod error;
pub mod merge;
pub mod types;
pub mod validation;

pub use error::ConfigError;
pub use types::{Config, Project};

use crate::dsl::ast::{Program, Stmt};
use crate::dsl::lexer::Lexer;
use crate::dsl::parser::Parser;
use std::collections::HashMap;
use std::path::{Path, PathBuf};

pub fn load(entry_path: &Path) -> Result<Config, ConfigError> {
    let abs_path = if entry_path.is_absolute() {
        entry_path.to_path_buf()
    } else {
        std::env::current_dir()
            .map_err(ConfigError::Io)?
            .join(entry_path)
    };

    let mut visited = HashMap::new();
    let programs = parse_recursive(&abs_path, &mut visited)?;

    let mut config = merge::merge(programs)?;
    validation::validate_base(&config)?;

    validation::resolve_use(&mut config, parse_recursive)?;

    Ok(config)
}

fn parse_recursive(
    file_path: &Path,
    visited: &mut HashMap<PathBuf, bool>,
) -> Result<Vec<Program>, ConfigError> {
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

    let data = std::fs::read_to_string(&abs_path).map_err(|e| {
        ConfigError::Io(std::io::Error::new(
            e.kind(),
            format!("Failed to read {}: {}", abs_path.display(), e),
        ))
    })?;

    let lexer = Lexer::new(data);
    let mut parser = Parser::new(lexer);
    let program = parser.parse().map_err(|errors| {
        let error_msgs: Vec<String> = errors.iter().map(|e| format!("{:?}", e)).collect();
        ConfigError::Parse(format!(
            "Parse errors in {}:\n{}",
            abs_path.display(),
            error_msgs.join("\n")
        ))
    })?;

    let mut results = Vec::new();

    let base_dir = abs_path.parent().unwrap_or_else(|| Path::new("."));

    for stmt in &program.stmts {
        if let Stmt::ImportDecl { paths, .. } = stmt {
            for rel_path in paths {
                let import_abs = base_dir.join(rel_path);
                let imported = parse_recursive(&import_abs, visited)?;
                results.extend(imported);
            }
        }
    }

    results.push(program);
    Ok(results)
}
