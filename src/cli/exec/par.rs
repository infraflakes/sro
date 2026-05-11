use super::super::load_config;
use crate::runner::{OutputCallback, Runner};
use crate::tui::{self, Model, TaskStatus, TuiApp, TuiEvent};
use std::path::PathBuf;
use std::sync::Arc;

pub fn run(
    config_arg: Option<PathBuf>,
    name: String,
    project: String,
    plain: bool,
) -> miette::Result<()> {
    let config = load_config(config_arg)?;

    if !config.projects.contains_key(&project) {
        return Err(miette::miette!("unknown project: {}", project));
    }

    let project_entry = &config.projects[&project];
    let fns = match project_entry.pars.get(&name) {
        Some(f) => f.clone(),
        None => {
            return Err(miette::miette!(
                "unknown par '{}' in project '{}'",
                name,
                project
            ));
        }
    };

    if plain {
        let mut runner = Runner::new(config);
        runner
            .run_par(&name, &project)
            .map_err(|e| miette::miette!("{}", e))?;
    } else {
        let rt = tokio::runtime::Runtime::new().map_err(|e| miette::miette!("{}", e))?;
        let result: miette::Result<()> = rt.block_on(async {
            let config = Arc::new(config);

            let mut model = Model::new("par".to_string(), format!("{} ({})", name, project));

            for fn_name in &fns {
                model.add_task(format!("{}({})", fn_name, project));
            }

            for task in &mut model.tasks {
                task.status = TaskStatus::Running;
            }

            let app = TuiApp::new(model);
            let tx = app.get_sender();

            let config_arc = Arc::clone(&config);
            let tx_clone = tx.clone();
            let project_clone = project.clone();
            tokio::spawn(async move {
                let mut join_handles = Vec::new();

                for (task_idx, fn_name) in fns.iter().enumerate() {
                    let tx_clone = tx_clone.clone();
                    let config_clone = Arc::clone(&config_arc);
                    let fn_name = fn_name.clone();
                    let project_name = project_clone.clone();

                    let handle = tokio::spawn(async move {
                        let tx_clone_for_callback = tx_clone.clone();
                        let task_idx_for_callback = task_idx;

                        let callback: OutputCallback = Box::new(move |line| {
                            tui::send_event(
                                &tx_clone_for_callback,
                                TuiEvent::AppendOutput(task_idx_for_callback, line),
                            );
                        });

                        let mut runner =
                            Runner::from_arc(config_clone).with_output_callback(callback);
                        match runner.execute_fn_call(&fn_name, &project_name) {
                            Ok(_) => {
                                tui::send_event(
                                    &tx_clone,
                                    TuiEvent::UpdateStatus(task_idx, TaskStatus::Success),
                                );
                            }
                            Err(e) => {
                                tui::send_event(
                                    &tx_clone,
                                    TuiEvent::AppendOutput(task_idx, format!("Error: {}", e)),
                                );
                                tui::send_event(
                                    &tx_clone,
                                    TuiEvent::UpdateStatus(task_idx, TaskStatus::Error),
                                );
                            }
                        }
                    });

                    join_handles.push(handle);
                }

                for handle in join_handles {
                    handle.await.ok();
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
