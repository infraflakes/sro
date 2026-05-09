use super::load_config;
use crate::sync;
use crate::tui::{self, TaskStatus, TuiApp, TuiEvent};
use std::io;
use std::path::PathBuf;

pub fn run(config_arg: Option<PathBuf>, plain: bool) -> miette::Result<()> {
    let config = load_config(config_arg)?;

    if plain {
        let mut stdout = io::stdout();
        sync::sync_all(&config, &mut stdout).map_err(|e| miette::miette!("{}", e))?;
    } else {
        let rt = tokio::runtime::Runtime::new().map_err(|e| miette::miette!("{}", e))?;
        rt.block_on(async {
            let project_names: Vec<String> = config.projects.keys().cloned().collect();
            let mut model = tui::Model::new("sync".to_string(), "all".to_string());

            for proj_name in &project_names {
                model.add_task(proj_name.clone());
            }

            let app = TuiApp::new(model);
            let tx = app.get_sender();

            let tx_clone = tx.clone();
            let sanctuary = config.sanctuary.clone();
            let projects = config.projects.clone();
            tokio::spawn(async move {
                for (i, proj_name) in project_names.iter().enumerate() {
                    tx_clone
                        .send(TuiEvent::UpdateStatus(i, TaskStatus::Running))
                        .ok();

                    if let Some(proj) = projects.get(proj_name) {
                        let tx = tx_clone.clone();
                        let result =
                            sync::sync_project_with_callback(&sanctuary, proj, |line: &str| {
                                tx.send(TuiEvent::AppendOutput(i, line.to_string())).ok();
                            });
                        match result {
                            Ok(_) => {
                                tx_clone
                                    .send(TuiEvent::UpdateStatus(i, TaskStatus::Success))
                                    .ok();
                            }
                            Err(e) => {
                                tx_clone
                                    .send(TuiEvent::AppendOutput(i, format!("Error: {}", e)))
                                    .ok();
                                tx_clone
                                    .send(TuiEvent::UpdateStatus(i, TaskStatus::Error))
                                    .ok();
                            }
                        }
                    } else {
                        tx_clone
                            .send(TuiEvent::UpdateStatus(i, TaskStatus::Error))
                            .ok();
                    }
                }

                if let Err(e) = sync::warn_unknown_repos(&sanctuary, &projects) {
                    tx_clone
                        .send(TuiEvent::AppendOutput(0, format!("Warning: {}", e)))
                        .ok();
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
