pub mod error;
pub mod merge;
pub mod types;
pub mod validation;

pub use error::ConfigError;
pub use types::{Config, Project};

use crate::dsl::ast::{Program, Stmt};
use crate::dsl::lexer::Lexer;
use crate::dsl::parser::Parser;
use std::collections::HashSet;
use std::path::{Path, PathBuf};

pub fn load(entry_path: &Path) -> Result<Config, ConfigError> {
    let abs_path = if entry_path.is_absolute() {
        entry_path.to_path_buf()
    } else {
        std::env::current_dir()
            .map_err(ConfigError::Io)?
            .join(entry_path)
    };

    let mut visited = HashSet::new();
    let programs = parse_recursive(&abs_path, &mut visited)?;

    let mut config = merge::merge(programs)?;
    validation::validate_base(&config)?;

    validation::resolve_use(&mut config, parse_recursive)?;

    Ok(config)
}

fn parse_recursive(
    file_path: &Path,
    visited: &mut HashSet<PathBuf>,
) -> Result<Vec<Program>, ConfigError> {
    let abs_path = if file_path.is_absolute() {
        file_path.to_path_buf()
    } else {
        std::env::current_dir()
            .map_err(ConfigError::Io)?
            .join(file_path)
    };

    if !visited.insert(abs_path.clone()) {
        return Err(ConfigError::CircularImport(abs_path.display().to_string()));
    }

    let data = std::fs::read_to_string(&abs_path).map_err(|e| {
        ConfigError::Io(std::io::Error::new(
            e.kind(),
            format!("Failed to read {}: {}", abs_path.display(), e),
        ))
    })?;

    let lexer = Lexer::new(data.clone());
    let mut parser = Parser::new(lexer);
    let program = parser.parse().map_err(|errors| {
        let source = data;
        let source_name = abs_path.display().to_string();
        let reports: Vec<miette::Report> = errors
            .into_iter()
            .map(|error| {
                miette::Report::new(error).with_source_code(miette::NamedSource::new(
                    source_name.clone(),
                    source.clone(),
                ))
            })
            .collect();
        ConfigError::ParseReports(reports)
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

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use std::path::Path;

    fn write_config(dir: &Path, name: &str, content: &str) {
        let path = dir.join(name);
        fs::write(&path, content)
            .unwrap_or_else(|e| panic!("failed to write {}: {}", path.display(), e));
    }

    #[test]
    fn test_load_basic() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp/dev`;\n\
var string a = `hello`;\n\
pr test { url = `http://example.com`; dir = `test`; }\n\
fn greet { log(`hi`); }\
",
        );
        let cfg = load(&dir.path().join("main.sro")).unwrap();
        assert_eq!(cfg.shell, "bash");
        assert_eq!(cfg.sanctuary, "/tmp/dev");
        assert_eq!(cfg.vars.get("a").unwrap(), "hello");
        assert!(cfg.projects.contains_key("test"));
        assert_eq!(cfg.projects["test"].url, "http://example.com");
        assert!(cfg.functions.contains_key("greet"));
    }

    #[test]
    fn test_import_resolution() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(dir.path(), "other.sro", "var string extra = `from-other`;");
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
import ./other.sro;\n\
var string x = $extra;\
",
        );
        let cfg = load(&dir.path().join("main.sro")).unwrap();
        assert_eq!(cfg.vars.get("x").unwrap(), "from-other");
    }

    #[test]
    fn test_circular_import() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "a.sro",
            "shell = `bash`; import ./b.sro; sanctuary = `/tmp`;",
        );
        write_config(
            dir.path(),
            "b.sro",
            "shell = `bash`; import ./a.sro; sanctuary = `/tmp`;",
        );
        let err = load(&dir.path().join("a.sro")).unwrap_err();
        let err_str = err.to_string();
        assert!(
            err_str.contains("circular") || err_str.contains("Circular"),
            "got: {}",
            err_str
        );
    }

    #[test]
    fn test_duplicate_sanctuary() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
sanctuary = `/other`;\
",
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        assert!(
            err.to_string().contains("duplicate sanctuary"),
            "got: {}",
            err
        );
    }

    #[test]
    fn test_duplicate_variable_decl() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
var string x = `a`;\n\
var string x = `b`;\
",
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        assert!(
            err.to_string().contains("duplicate variable"),
            "got: {}",
            err
        );
    }

    #[test]
    fn test_duplicate_project() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
pr p1 { url = `u`; dir = `d1`; }\n\
pr p1 { url = `u2`; dir = `d2`; }\
",
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        assert!(
            err.to_string().contains("duplicate project"),
            "got: {}",
            err
        );
    }

    #[test]
    fn test_variable_chain_resolution() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
var string a = `x`;\n\
var string b = $a;\n\
var string c = $b;\
",
        );
        let cfg = load(&dir.path().join("main.sro")).unwrap();
        assert_eq!(cfg.vars["a"], "x");
        assert_eq!(cfg.vars["b"], "x");
        assert_eq!(cfg.vars["c"], "x");
    }

    #[test]
    fn test_undefined_variable() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
var string x = $missing;\
",
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        assert!(
            err.to_string().contains("undefined variable"),
            "got: {}",
            err
        );
    }

    #[test]
    fn test_missing_shell() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
sanctuary = `/tmp`;\n\
pr test { url = `http://example.com`; dir = `test`; }\
",
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        assert!(err.to_string().contains("shell"), "got: {}", err);
    }

    #[test]
    fn test_missing_sanctuary() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
pr test { url = `http://example.com`; dir = `test`; }\
",
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        assert!(err.to_string().contains("sanctuary"), "got: {}", err);
    }

    #[test]
    fn test_sanctuary_absolute_path() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `relative/path`;\
",
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        assert!(err.to_string().contains("absolute"), "got: {}", err);
    }

    #[test]
    fn test_missing_url() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
pr p { dir = `d`; }\
",
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        assert!(err.to_string().contains("url is required"), "got: {}", err);
    }

    #[test]
    fn test_missing_dir() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
pr p { url = `u`; }\
",
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        assert!(err.to_string().contains("dir is required"), "got: {}", err);
    }

    #[test]
    fn test_duplicate_dir() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
pr a { url = `ua`; dir = `shared`; }\n\
pr b { url = `ub`; dir = `shared`; }\
",
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        assert!(
            err.to_string().contains("duplicate directory"),
            "got: {}",
            err
        );
    }

    #[test]
    fn test_invalid_sync_value() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
pr p { url = `u`; dir = `d`; sync = `invalid`; }\
",
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        assert!(err.to_string().contains("sync"), "got: {}", err);
    }

    #[test]
    fn test_empty_config() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(dir.path(), "main.sro", "");
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        assert!(err.to_string().contains("shell"), "got: {}", err);
    }

    #[test]
    fn test_only_shell_and_sanctuary() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\
",
        );
        let cfg = load(&dir.path().join("main.sro")).unwrap();
        assert_eq!(cfg.shell, "bash");
        assert_eq!(cfg.sanctuary, "/tmp");
    }

    #[test]
    fn test_interpolation_in_backtick() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
var string name = `world`;\n\
var string greeting = `hello ${name}`;\
",
        );
        let cfg = load(&dir.path().join("main.sro")).unwrap();
        assert_eq!(cfg.vars["greeting"], "hello world");
    }

    #[test]
    fn test_project_field_with_var_ref() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
var string myurl = `http://example.com`;\n\
pr x { url = $myurl; dir = `d`; }\
",
        );
        let cfg = load(&dir.path().join("main.sro")).unwrap();
        assert_eq!(cfg.projects["x"].url, "http://example.com");
    }

    #[test]
    fn test_duplicate_fn_seq_par_names() {
        let dir = tempfile::TempDir::new().unwrap();

        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
fn dup { log(`a`); }\n\
fn dup { log(`b`); }\
",
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        assert!(
            err.to_string().contains("duplicate function"),
            "got: {}",
            err
        );

        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
pr test { url = `http://example.com`; dir = `test`; }\n\
seq dup { check(test); }\n\
seq dup { build(test); }\
",
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        assert!(err.to_string().contains("duplicate seq"), "got: {}", err);

        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
pr test { url = `http://example.com`; dir = `test`; }\n\
par dup { check(test); }\n\
par dup { build(test); }\
",
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        assert!(err.to_string().contains("duplicate par"), "got: {}", err);
    }

    #[test]
    fn test_multi_file_parse_order() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(dir.path(), "a.sro", "var string a = `from-a`;");
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
import ./a.sro;\n\
var string b = $a;\
",
        );
        let cfg = load(&dir.path().join("main.sro")).unwrap();
        assert_eq!(cfg.vars["b"], "from-a");
    }

    #[test]
    fn test_undefined_var_in_fn_body() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
pr test { url = `http://example.com`; dir = `test`; }\n\
fn badfn { log($undefined); }\
",
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        assert!(
            err.to_string().contains("undefined variable"),
            "got: {}",
            err
        );
    }

    #[test]
    fn test_seq_par_reference_validation() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
fn real { log(`hi`); }\n\
pr p { url = `u`; dir = `d`; }\n\
seq s { unknown(p); }\n\
par p2 { fake(q); }\
",
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        let err_str = err.to_string();
        assert!(err_str.contains("unknown function"), "got: {}", err_str);
    }

    #[test]
    fn test_seq_cycle_detection() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
pr test { url = `http://example.com`; dir = `test`; }\n\
seq a { seq.b; }\n\
seq b { seq.a; }\
",
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        let err_str = err.to_string();
        assert!(
            err_str.contains("cycle") || err_str.contains("Cycle"),
            "got: {}",
            err_str
        );
    }

    #[test]
    fn test_self_referencing_seq() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
pr test { url = `http://example.com`; dir = `test`; }\n\
seq a { seq.a; }\
",
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        let err_str = err.to_string();
        assert!(
            err_str.contains("cycle") || err_str.contains("Cycle"),
            "got: {}",
            err_str
        );
    }

    #[test]
    fn test_valid_seq_par_references() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
pr test { url = `http://example.com`; dir = `test`; }\n\
fn real { log(`hi`); }\n\
seq s { real(test); }\n\
par p { seq.s; }\
",
        );
        let cfg = load(&dir.path().join("main.sro")).unwrap();
        assert!(cfg.seqs.contains_key("s"));
        assert!(cfg.pars.contains_key("p"));
    }

    #[test]
    fn test_duplicate_var_in_fn_body() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
pr test { url = `http://example.com`; dir = `test`; }\n\
fn bad {\n\
    var string x = `a`;\n\
    var string x = `b`;\n\
}\
",
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        assert!(
            err.to_string().contains("duplicate variable"),
            "got: {}",
            err
        );
    }

    #[test]
    fn test_use_file_resolution() {
        let dir = tempfile::TempDir::new().unwrap();
        let proj_dir = dir.path().join("test");
        std::fs::create_dir_all(&proj_dir).unwrap();
        write_config(
            &proj_dir,
            "use.sro",
            "\
var string usevar = `from-use`;\n\
fn usefn { log(`from-use`); }\n\
seq useseq { usefn(test); }\n\
par usepar { usefn(test); }\
",
        );
        write_config(
            dir.path(),
            "main.sro",
            &format!(
                "\
shell = `bash`;\n\
sanctuary = `{}`;\n\
pr test {{ url = `http://example.com`; dir = `test`; use = `use.sro`; }}\
",
                dir.path().display()
            ),
        );
        let cfg = load(&dir.path().join("main.sro")).unwrap();
        assert_eq!(cfg.vars.get("usevar").unwrap(), "from-use");
        assert!(cfg.functions.contains_key("usefn"));
        assert!(cfg.seqs.contains_key("useseq"));
        assert!(cfg.pars.contains_key("usepar"));
    }

    #[test]
    fn test_use_file_not_found() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            &format!(
                "\
shell = `bash`;\n\
sanctuary = `{}`;\n\
pr test {{ url = `http://example.com`; dir = `test`; use = `nonexistent.sro`; }}\
",
                dir.path().display()
            ),
        );
        let err = load(&dir.path().join("main.sro")).unwrap_err();
        let err_str = err.to_string();
        assert!(
            err_str.contains("use file not found") || err_str.contains("not found"),
            "got: {}",
            err_str
        );
    }

    #[test]
    fn test_use_file_sync_ignore_skips() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            &format!(
                "\
shell = `bash`;\n\
sanctuary = `{}`;\n\
pr test {{ url = `http://example.com`; dir = `test`; sync = `ignore`; use = `use.sro`; }}\
",
                dir.path().display()
            ),
        );
        let cfg = load(&dir.path().join("main.sro")).unwrap();
        assert!(cfg.projects.contains_key("test"));
    }

    #[test]
    fn test_shell_exec_resolution() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            "\
shell = `bash`;\n\
sanctuary = `/tmp`;\n\
var shell test_var = `echo hello`;\
",
        );
        let cfg = load(&dir.path().join("main.sro")).unwrap();
        assert_eq!(cfg.vars["test_var"], "hello");
    }

    #[test]
    fn test_sanctuary_with_var_ref() {
        let dir = tempfile::TempDir::new().unwrap();
        write_config(
            dir.path(),
            "main.sro",
            &format!(
                "\
shell = `bash`;\n\
var shell workdir = `echo {}`;\n\
sanctuary = $workdir;\
",
                dir.path().display()
            ),
        );
        let cfg = load(&dir.path().join("main.sro")).unwrap();
        assert_eq!(cfg.sanctuary, dir.path().to_str().unwrap());
    }
}
