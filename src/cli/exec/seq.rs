use super::super::load_config;
use crate::dsl::ast::{BlockStmt, Stmt};
use crate::runner::{OutputCallback, Runner};
use crate::tui::{Model, TaskStatus, TuiApp, TuiEvent};
use std::path::PathBuf;
use std::sync::Arc;

pub fn run(config_arg: Option<PathBuf>, name: String, plain: bool) -> miette::Result<()> {
    let config = load_config(config_arg)?;

    if plain {
        let mut runner = Runner::new(config);
        runner
            .run_seq(&name)
            .map_err(|e| miette::miette!("{}", e))?;
    } else {
        let rt = tokio::runtime::Runtime::new().map_err(|e| miette::miette!("{}", e))?;
        let result = rt.block_on(async {
            let config = Arc::new(config);

            let seq_decl = match config.seqs.get(&name) {
                Some(decl) => decl,
                None => {
                    return Err(miette::miette!("unknown seq: {}", name));
                }
            };

            let mut tasks: Vec<(usize, String, crate::dsl::ast::BlockStmt)> = Vec::new();
            if let Stmt::SeqDecl { stmts, .. } = seq_decl {
                for (i, stmt) in stmts.iter().enumerate() {
                    match stmt {
                        BlockStmt::FnCall {
                            fn_name,
                            project_name,
                            ..
                        } => {
                            tasks.push((i, format!("{}({})", fn_name, project_name), stmt.clone()));
                        }
                        BlockStmt::SeqRef { seq_name, .. } => {
                            tasks.push((i, format!("seq:{}", seq_name), stmt.clone()));
                        }
                    }
                }
            }

            let mut model = Model::new("seq".to_string(), name.clone());

            for (_, task_name, _) in &tasks {
                model.add_task(task_name.clone());
            }

            let app = TuiApp::new(model);
            let tx = app.get_sender();

            let config_arc = Arc::clone(&config);
            let tx_clone = tx.clone();
            tokio::spawn(async move {
                for (task_idx, _task_name, stmt) in tasks {
                    tx_clone
                        .send(TuiEvent::UpdateStatus(task_idx, TaskStatus::Running))
                        .ok();

                    let tx_clone_for_callback = tx_clone.clone();
                    let task_idx_for_callback = task_idx;

                    let callback: OutputCallback = Box::new(move |line| {
                        tx_clone_for_callback
                            .send(TuiEvent::AppendOutput(task_idx_for_callback, line))
                            .ok();
                    });

                    match &stmt {
                        BlockStmt::FnCall {
                            fn_name,
                            project_name,
                            ..
                        } => {
                            let mut runner = Runner::from_arc(Arc::clone(&config_arc))
                                .with_output_callback(callback);
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
                        BlockStmt::SeqRef { seq_name, .. } => {
                            let mut runner = Runner::from_arc(Arc::clone(&config_arc))
                                .with_output_callback(callback);
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

            Ok(())
        });

        result?;
    }
    Ok(())
}
