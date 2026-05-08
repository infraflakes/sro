use crate::config::load;
use crate::runner::Runner;
use crate::tui::{TaskStatus, TuiApp, TuiEvent};
use std::path::PathBuf;
use super::super::{get_config_path, print_parse_errors};

pub fn run(config_arg: Option<PathBuf>, name: String, plain: bool) -> miette::Result<()> {
    let config_path = get_config_path(config_arg);
    let config = load(&config_path).map_err(|e| {
        if let crate::config::ConfigError::ParseReports(reports) = e {
            print_parse_errors(reports)
        } else {
            miette::miette!("{}", e)
        }
    })?;

    if plain {
        let mut runner = Runner::new(config);
        runner
            .run_seq(&name)
            .map_err(|e| miette::miette!("{}", e))?;
    } else {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let seq_decl = config
                .seqs
                .get(&name)
                .ok_or_else(|| format!("unknown seq: {}", name))
                .unwrap();

            let mut tasks: Vec<(usize, String, crate::dsl::ast::SeqStmt)> = Vec::new();
            if let crate::dsl::ast::Stmt::SeqDecl { stmts, .. } = seq_decl {
                for (i, stmt) in stmts.iter().enumerate() {
                    match stmt {
                        crate::dsl::ast::SeqStmt::FnCall {
                            fn_name,
                            project_name,
                            ..
                        } => {
                            tasks.push((i, format!("{}({})", fn_name, project_name), stmt.clone()));
                        }
                        crate::dsl::ast::SeqStmt::SeqRef { seq_name, .. } => {
                            tasks.push((i, format!("seq:{}", seq_name), stmt.clone()));
                        }
                    }
                }
            }

            let mut model = crate::tui::Model::new("seq".to_string(), name.clone());

            for (_, task_name, _) in &tasks {
                model.add_task(task_name.clone());
            }

            let app = TuiApp::new(model);
            let tx = app.get_sender();

            let config_clone = config.clone();
            let tx_clone = tx.clone();
            tokio::spawn(async move {
                for (task_idx, _task_name, stmt) in tasks {
                    tx_clone
                        .send(TuiEvent::UpdateStatus(task_idx, TaskStatus::Running))
                        .ok();

                    let tx_clone_for_callback = tx_clone.clone();
                    let task_idx_for_callback = task_idx;

                    let callback: crate::runner::OutputCallback = Box::new(move |line| {
                        tx_clone_for_callback
                            .send(TuiEvent::AppendOutput(task_idx_for_callback, line))
                            .ok();
                    });

                    match &stmt {
                        crate::dsl::ast::SeqStmt::FnCall {
                            fn_name,
                            project_name,
                            ..
                        } => {
                            let mut runner =
                                Runner::new(config_clone.clone()).with_output_callback(callback);
                            match runner.execute_fn_call(fn_name, project_name) {
                                Ok(_) => {
                                    tx_clone
                                        .send(TuiEvent::UpdateStatus(task_idx, TaskStatus::Success))
                                        .ok();
                                }
                                Err(e) => {
                                    tx_clone
                                        .send(TuiEvent::AppendOutput(
                                            task_idx,
                                            format!("Error: {}", e),
                                        ))
                                        .ok();
                                    tx_clone
                                        .send(TuiEvent::UpdateStatus(task_idx, TaskStatus::Error))
                                        .ok();
                                }
                            }
                        }
                        crate::dsl::ast::SeqStmt::SeqRef { seq_name, .. } => {
                            let mut runner =
                                Runner::new(config_clone.clone()).with_output_callback(callback);
                            match runner.run_seq(seq_name) {
                                Ok(_) => {
                                    tx_clone
                                        .send(TuiEvent::UpdateStatus(task_idx, TaskStatus::Success))
                                        .ok();
                                }
                                Err(e) => {
                                    tx_clone
                                        .send(TuiEvent::AppendOutput(
                                            task_idx,
                                            format!("Error: {}", e),
                                        ))
                                        .ok();
                                    tx_clone
                                        .send(TuiEvent::UpdateStatus(task_idx, TaskStatus::Error))
                                        .ok();
                                }
                            }
                        }
                    }
                }
            });

            if let Err(e) = app.run().await {
                eprintln!("TUI error: {}", e);
                std::process::exit(1);
            }
        });
    }
    Ok(())
}
