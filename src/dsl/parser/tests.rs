use super::*;
use crate::dsl::lexer::Lexer;

fn parse_program(input: &str) -> Result<Program, Vec<ParseError>> {
    let lexer = Lexer::new(input.to_string());
    let mut parser = Parser::new(lexer);
    parser.parse()
}

fn count_stmt_types(program: &Program) -> Vec<&'static str> {
    program
        .stmts
        .iter()
        .map(|s| match s {
            Stmt::ShellDecl { .. } => "shell",
            Stmt::SanctuaryDecl { .. } => "sanctuary",
            Stmt::ImportDecl { .. } => "import",
            Stmt::VarDecl { .. } => "var",
            Stmt::ProjectDecl { .. } => "pr",
            Stmt::FnDecl { .. } => "fn",
            Stmt::SeqDecl { .. } => "seq",
            Stmt::ParDecl { .. } => "par",
        })
        .collect()
}

#[test]
fn test_shell_decl() {
    let prog = parse_program("shell = `bash`;").unwrap();
    assert_eq!(count_stmt_types(&prog), vec!["shell"]);
    match &prog.stmts[0] {
        Stmt::ShellDecl { value, .. } => assert_eq!(value, "bash"),
        _ => panic!("expected ShellDecl"),
    }
}

#[test]
fn test_sanctuary_decl() {
    let prog = parse_program("sanctuary = `/tmp/dev`;").unwrap();
    assert_eq!(count_stmt_types(&prog), vec!["sanctuary"]);
    match &prog.stmts[0] {
        Stmt::SanctuaryDecl { value, .. } => match value {
            Expr::BacktickLit { parts, .. } => {
                // Each literal character is a separate part
                let concat: String = parts.iter().map(|p| p.value.as_str()).collect();
                assert_eq!(concat, "/tmp/dev");
            }
            _ => panic!("expected BacktickLit"),
        },
        _ => panic!("expected SanctuaryDecl"),
    }
}

#[test]
fn test_sanctuary_with_var_ref() {
    let prog = parse_program("sanctuary = $workdir;").unwrap();
    match &prog.stmts[0] {
        Stmt::SanctuaryDecl { value, .. } => match value {
            Expr::VarRef { name, .. } => assert_eq!(name, "workdir"),
            _ => panic!("expected VarRef"),
        },
        _ => panic!("expected SanctuaryDecl"),
    }
}

#[test]
fn test_import_decl() {
    let prog = parse_program("import ./other.sro;").unwrap();
    assert_eq!(count_stmt_types(&prog), vec!["import"]);
    match &prog.stmts[0] {
        Stmt::ImportDecl { paths, .. } => {
            assert_eq!(paths, &vec!["./other.sro".to_string()]);
        }
        _ => panic!("expected ImportDecl"),
    }
}

#[test]
fn test_var_string_decl() {
    let prog = parse_program("var string x = `hello`;").unwrap();
    match &prog.stmts[0] {
        Stmt::VarDecl {
            var_type,
            name,
            value,
            ..
        } => {
            assert_eq!(*var_type, VarType::String);
            assert_eq!(name, "x");
            match value {
                Expr::BacktickLit { parts, .. } => {
                    // Each literal character is a separate part
                    let concat: String = parts.iter().map(|p| p.value.as_str()).collect();
                    assert_eq!(concat, "hello");
                }
                _ => panic!("expected BacktickLit"),
            }
        }
        _ => panic!("expected VarDecl"),
    }
}

#[test]
fn test_var_shell_decl() {
    let prog = parse_program("shell = `bash`;\nvar shell x = `echo hello`;").unwrap();
    match &prog.stmts[1] {
        Stmt::VarDecl { var_type, name, .. } => {
            assert_eq!(*var_type, VarType::Shell);
            assert_eq!(name, "x");
        }
        _ => panic!("expected VarDecl"),
    }
}

#[test]
fn test_var_missing_type_annotation() {
    let result = parse_program("var x = `hello`;");
    assert!(result.is_err());
    let errs = result.unwrap_err();
    assert!(
        errs.iter()
            .any(|e| e.to_string().contains("expected 'string' or 'shell'"))
    );
}

#[test]
fn test_var_invalid_type() {
    let result = parse_program("var number x = `5`;");
    assert!(result.is_err());
    let errs = result.unwrap_err();
    assert!(
        errs.iter()
            .any(|e| e.to_string().contains("expected 'string' or 'shell'"))
    );
}

#[test]
fn test_project_decl() {
    let input = "\npr todo {\n    url = `git@github.com:user/repo.git`;\n    dir = `todo`;\n    sync = `clone`;\n    use = `./main.sro`;\n    branch = `main`;\n}";
    let prog = parse_program(input).unwrap();
    assert_eq!(count_stmt_types(&prog), vec!["pr"]);
    match &prog.stmts[0] {
        Stmt::ProjectDecl { name, fields, .. } => {
            assert_eq!(name, "todo");
            assert_eq!(fields.len(), 5);
            let keys: Vec<&str> = fields.iter().map(|f| f.key.as_str()).collect();
            assert_eq!(keys, vec!["url", "dir", "sync", "use", "branch"]);
        }
        _ => panic!("expected ProjectDecl"),
    }
}

#[test]
fn test_project_duplicate_fields() {
    let input = "\npr x {\n    url = `a`;\n    url = `b`;\n    dir = `d`;\n}";
    let prog = parse_program(input).unwrap();
    match &prog.stmts[0] {
        Stmt::ProjectDecl { fields, .. } => {
            // Parser accepts duplicates (validation rejects them later)
            assert_eq!(fields.len(), 3);
        }
        _ => panic!("expected ProjectDecl"),
    }
}

#[test]
fn test_fn_decl_with_body() {
    let input = "\nfn init {\n    log(`a`);\n    var string x = `b`;\n    env [KEY = `val`] {\n        log(`c`);\n    };\n}";
    let prog = parse_program(input).unwrap();
    assert_eq!(count_stmt_types(&prog), vec!["fn"]);
    match &prog.stmts[0] {
        Stmt::FnDecl { name, body, .. } => {
            assert_eq!(name, "init");
            assert_eq!(body.len(), 3);
            assert!(matches!(body[0], FnStmt::Log { .. }));
            assert!(matches!(body[1], FnStmt::VarDecl { .. }));
            assert!(matches!(body[2], FnStmt::EnvBlock { .. }));
            if let FnStmt::EnvBlock {
                pairs,
                body: env_body,
                ..
            } = &body[2]
            {
                assert_eq!(pairs.len(), 1);
                assert_eq!(pairs[0].key, "KEY");
                assert_eq!(env_body.len(), 1);
                assert!(matches!(env_body[0], FnStmt::Log { .. }));
            }
        }
        _ => panic!("expected FnDecl"),
    }
}

#[test]
fn test_env_with_trailing_comma() {
    let input = "\nfn init {\n    env [X = `1`,] {\n        log(`x`);\n    };\n}";
    let prog = parse_program(input).unwrap();
    match &prog.stmts[0] {
        Stmt::FnDecl { body, .. } => match &body[0] {
            FnStmt::EnvBlock { pairs, .. } => {
                assert_eq!(pairs.len(), 1);
                assert_eq!(pairs[0].key, "X");
            }
            _ => panic!("expected EnvBlock"),
        },
        _ => panic!("expected FnDecl"),
    }
}

#[test]
fn test_empty_fn_body() {
    let prog = parse_program("\nfn empty {}").unwrap();
    match &prog.stmts[0] {
        Stmt::FnDecl { name, body, .. } => {
            assert_eq!(name, "empty");
            assert!(body.is_empty());
        }
        _ => panic!("expected FnDecl"),
    }
}

#[test]
fn test_empty_seq_body() {
    let prog = parse_program("\nseq empty {}").unwrap();
    match &prog.stmts[0] {
        Stmt::SeqDecl { name, stmts, .. } => {
            assert_eq!(name, "empty");
            assert!(stmts.is_empty());
        }
        _ => panic!("expected SeqDecl"),
    }
}

#[test]
fn test_empty_par_body() {
    let prog = parse_program("\npar empty {}").unwrap();
    match &prog.stmts[0] {
        Stmt::ParDecl { name, stmts, .. } => {
            assert_eq!(name, "empty");
            assert!(stmts.is_empty());
        }
        _ => panic!("expected ParDecl"),
    }
}

#[test]
fn test_seq_with_fn_calls() {
    let input = "\nseq build {\n    check(todo);\n    build(todo);\n}";
    let prog = parse_program(input).unwrap();
    match &prog.stmts[0] {
        Stmt::SeqDecl { name, stmts, .. } => {
            assert_eq!(name, "build");
            assert_eq!(stmts.len(), 2);
            match &stmts[0] {
                BlockStmt::FnCall {
                    fn_name,
                    project_name,
                    ..
                } => {
                    assert_eq!(fn_name, "check");
                    assert_eq!(project_name, "todo");
                }
                _ => panic!("expected FnCall"),
            }
            match &stmts[1] {
                BlockStmt::FnCall {
                    fn_name,
                    project_name,
                    ..
                } => {
                    assert_eq!(fn_name, "build");
                    assert_eq!(project_name, "todo");
                }
                _ => panic!("expected FnCall"),
            }
        }
        _ => panic!("expected SeqDecl"),
    }
}

#[test]
fn test_par_with_seq_ref() {
    let input = "\npar test {\n    check(todo);\n    seq.init;\n}";
    let prog = parse_program(input).unwrap();
    match &prog.stmts[0] {
        Stmt::ParDecl { stmts, .. } => {
            assert_eq!(stmts.len(), 2);
            match &stmts[0] {
                BlockStmt::FnCall {
                    fn_name,
                    project_name,
                    ..
                } => {
                    assert_eq!(fn_name, "check");
                    assert_eq!(project_name, "todo");
                }
                _ => panic!("expected FnCall"),
            }
            match &stmts[1] {
                BlockStmt::SeqRef { seq_name, .. } => {
                    assert_eq!(seq_name, "init");
                }
                _ => panic!("expected SeqRef"),
            }
        }
        _ => panic!("expected ParDecl"),
    }
}

#[test]
fn test_exec_with_interpolation() {
    let input = "\nfn init {\n    exec(`hello ${name}`);\n}";
    let prog = parse_program(input).unwrap();
    match &prog.stmts[0] {
        Stmt::FnDecl { body, .. } => {
            match &body[0] {
                FnStmt::Exec { value, .. } => {
                    match value {
                        Expr::BacktickLit { parts, .. } => {
                            // Each literal char is a separate part + one var part
                            let literal_parts: Vec<&TemplatePart> =
                                parts.iter().filter(|p| !p.is_var).collect();
                            let var_parts: Vec<&TemplatePart> =
                                parts.iter().filter(|p| p.is_var).collect();
                            assert_eq!(literal_parts.len(), 6); // "hello " = 6 chars
                            assert_eq!(var_parts.len(), 1);
                            assert_eq!(var_parts[0].value, "name");
                        }
                        _ => panic!("expected BacktickLit"),
                    }
                }
                _ => panic!("expected ExecStmt"),
            }
        }
        _ => panic!("expected FnDecl"),
    }
}

#[test]
fn test_log_with_var_ref() {
    let input = "\nfn init {\n    log($myvar);\n}";
    let prog = parse_program(input).unwrap();
    match &prog.stmts[0] {
        Stmt::FnDecl { body, .. } => match &body[0] {
            FnStmt::Log { value, .. } => match value {
                Expr::VarRef { name, .. } => {
                    assert_eq!(name, "myvar");
                }
                _ => panic!("expected VarRef"),
            },
            _ => panic!("expected LogStmt"),
        },
        _ => panic!("expected FnDecl"),
    }
}

#[test]
fn test_cd_stmt() {
    let input = "\nfn go {\n    cd(`/tmp`);\n}";
    let prog = parse_program(input).unwrap();
    match &prog.stmts[0] {
        Stmt::FnDecl { body, .. } => match &body[0] {
            FnStmt::Cd { arg, .. } => {
                assert_eq!(arg, "/tmp");
            }
            _ => panic!("expected CdStmt"),
        },
        _ => panic!("expected FnDecl"),
    }
}

#[test]
fn test_nested_env_blocks() {
    let input = "\nfn init {\n    env [OUTER = `1`] {\n        env [INNER = `2`] {\n            exec(`go build`);\n        };\n    };\n}";
    let prog = parse_program(input).unwrap();
    match &prog.stmts[0] {
        Stmt::FnDecl { body, .. } => match &body[0] {
            FnStmt::EnvBlock {
                pairs,
                body: outer_body,
                ..
            } => {
                assert_eq!(pairs[0].key, "OUTER");
                match &outer_body[0] {
                    FnStmt::EnvBlock {
                        pairs: inner_pairs,
                        body: inner_body,
                        ..
                    } => {
                        assert_eq!(inner_pairs[0].key, "INNER");
                        assert!(matches!(inner_body[0], FnStmt::Exec { .. }));
                    }
                    _ => panic!("expected nested EnvBlock"),
                }
            }
            _ => panic!("expected EnvBlock"),
        },
        _ => panic!("expected FnDecl"),
    }
}

// --- Error recovery tests ---

#[test]
fn test_missing_semicolon() {
    let result = parse_program("sanctuary = `$HOME`");
    assert!(result.is_err());
    let errs = result.unwrap_err();
    assert!(errs.iter().any(|e| e.to_string().contains("expected")));
}

#[test]
fn test_missing_opening_brace_after_fn() {
    let result = parse_program("fn bad");
    assert!(result.is_err());
}

#[test]
fn test_missing_opening_brace_after_seq() {
    let result = parse_program("seq bad");
    assert!(result.is_err());
}

#[test]
fn test_missing_opening_brace_after_par() {
    let result = parse_program("par bad");
    assert!(result.is_err());
}

#[test]
fn test_unexpected_token_in_fn_body() {
    let result = parse_program("fn bad { unknown }");
    assert!(result.is_err());
    let errs = result.unwrap_err();
    assert!(
        errs.iter()
            .any(|e| e.to_string().contains("unexpected token in fn body"))
    );
}

#[test]
fn test_unexpected_token_at_top_level() {
    let result = parse_program("fooobar = `bar`;");
    assert!(result.is_err());
    let errs = result.unwrap_err();
    assert!(
        errs.iter()
            .any(|e| e.to_string().contains("unexpected token"))
    );
}

#[test]
fn test_unclosed_fn_brace() {
    let result = parse_program("fn bad { log(`hi`);");
    assert!(result.is_err());
}

#[test]
fn test_unclosed_seq_brace() {
    let result = parse_program("seq s { check(p);");
    assert!(result.is_err());
}

#[test]
fn test_empty_env_pairs() {
    // The parser allows empty env pairs (validation may reject them later)
    let prog = parse_program("\nfn init {\n    env [] { log(`x`); };\n}").unwrap();
    match &prog.stmts[0] {
        Stmt::FnDecl { body, .. } => match &body[0] {
            FnStmt::EnvBlock {
                pairs,
                body: env_body,
                ..
            } => {
                assert!(pairs.is_empty());
                assert_eq!(env_body.len(), 1);
            }
            _ => panic!("expected EnvBlock"),
        },
        _ => panic!("expected FnDecl"),
    }
}

#[test]
fn test_par_cannot_reference_par() {
    // The parser doesn't validate par references (validation is later)
    // But the parse_block_stmt function does NOT handle `par.X` - only `seq.X`
    let result = parse_program("par p { par.x; }");
    assert!(result.is_err());
    let errs = result.unwrap_err();
    assert!(
        errs.iter()
            .any(|e| e.to_string().contains("unexpected token"))
    );
}

#[test]
fn test_empty_template_variable() {
    let result = parse_program("fn f { log(`hello ${}`); }");
    assert!(result.is_err());
    let errs = result.unwrap_err();
    assert!(
        errs.iter()
            .any(|e| e.to_string().contains("empty variable name"))
    );
}

#[test]
fn test_unclosed_template_interpolation() {
    let result = parse_program("fn f { log(`hello ${name`); }");
    assert!(result.is_err());
    let errs = result.unwrap_err();
    assert!(
        errs.iter()
            .any(|e| e.to_string().contains("unclosed variable interpolation"))
    );
}

#[test]
fn test_var_with_var_ref_value() {
    let input = "var string x = `a`; var string y = `${x}`;";
    let prog = parse_program(input).unwrap();
    assert_eq!(count_stmt_types(&prog), vec!["var", "var"]);
}

#[test]
fn test_multiple_top_level_statements() {
    let input = "shell = `bash`;\n\
                 sanctuary = `/tmp`;\n\
                 import ./other.sro;\n\
                 var string x = `hello`;\n\
                 pr p { url = `u`; dir = `d`; }\n\
                 fn f { log(`hi`); }\n\
                 seq s { f(p); }\n\
                 par p2 { seq.s; }";
    let prog = parse_program(input).unwrap();
    assert_eq!(
        count_stmt_types(&prog),
        vec![
            "shell",
            "sanctuary",
            "import",
            "var",
            "pr",
            "fn",
            "seq",
            "par"
        ]
    );
}

#[test]
fn test_error_recovery_skips_bad_stmt() {
    // Parser should skip the bad statement and continue to parse valid ones
    let result = parse_program("shell = `bash`;\nfn bad { unknown }\nsanctuary = `/tmp`;");
    // With error recovery, we might still get errors but shouldn't panic
    // The parse returns Err with Vec<ParseError> but we should have parsed valid stmts too
    match result {
        Ok(prog) => {
            assert_eq!(prog.stmts.len(), 3); // shell, fn, sanctuary
        }
        Err(errs) => {
            // Error recovery may produce errors but shouldn't crash
            assert!(
                errs.iter()
                    .any(|e| e.to_string().contains("unexpected token"))
            );
        }
    }
}

#[test]
fn test_import_path_types() {
    // Test that various path formats are accepted by the parser
    let inputs = vec![
        "import ./foo.sro;",
        "import ../foo.sro;",
        "import ../../dir/foo.sro;",
    ];
    for input in inputs {
        let result = parse_program(input);
        assert!(result.is_ok(), "expected success for: {}", input);
    }
}
