use crate::config::load;
use crate::runner::Runner;
use crate::sync;
use crate::tui::{self, TuiApp, TuiEvent, TaskStatus};
use clap::{Parser, Subcommand};
use std::io;
use std::path::PathBuf;

#[derive(Parser)]
#[command(name = "sro")]
#[command(about = "SRO - Sanctuary Repository Orchestrator", long_about = None)]
struct Cli {
    /// Path to config file
    #[arg(short, long, global = true)]
    config: Option<PathBuf>,
    
    /// Use plain text output instead of TUI
    #[arg(short, long, global = true)]
    plain: bool,
    
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Parse and validate the configuration file
    Validate,
    /// Clone/sync project repositories
    Sync,
    /// Run a sequential execution block
    Seq {
        /// Name of the sequential block to run
        name: String,
    },
    /// Run a parallel execution block
    Par {
        /// Name of the parallel block to run
        name: String,
    },
    /// Print the version number
    Version,
}

pub fn run() {
    let cli = Cli::parse();
    
    match cli.command {
        Commands::Validate => run_validate(cli.config),
        Commands::Sync => run_sync(cli.config, cli.plain),
        Commands::Seq { name } => run_seq(cli.config, name, cli.plain),
        Commands::Par { name } => run_par(cli.config, name, cli.plain),
        Commands::Version => run_version(),
    }
}

fn get_config_path(config_arg: Option<PathBuf>) -> PathBuf {
    if let Some(path) = config_arg {
        return path;
    }
    
    // Default to ~/.config/sro/config.sro
    if let Some(config_dir) = dirs::config_dir() {
        return config_dir.join("sro").join("config.sro");
    }
    
    PathBuf::from("config.sro")
}

fn run_validate(config_arg: Option<PathBuf>) {
    let config_path = get_config_path(config_arg);
    println!("Validating config: {:?}", config_path);
    
    match load(&config_path) {
        Ok(cfg) => {
            println!("Config is valid!");
            println!("Shell: {}", cfg.shell);
            println!("Sanctuary: {}", cfg.sanctuary);
            println!("Projects: {}", cfg.projects.len());
            println!("Functions: {}", cfg.functions.len());
            println!("Seqs: {}", cfg.seqs.len());
            println!("Pars: {}", cfg.pars.len());
        }
        Err(e) => {
            eprintln!("Error: {}", e);
            std::process::exit(1);
        }
    }
}

fn run_sync(config_arg: Option<PathBuf>, plain: bool) {
    let config_path = get_config_path(config_arg);
    let config = load(&config_path).unwrap_or_else(|e| {
        eprintln!("Error: {}", e);
        std::process::exit(1);
    });
    
    if plain {
        let mut stdout = io::stdout();
        if let Err(e) = sync::sync_all(&config, &mut stdout) {
            eprintln!("Error: {}", e);
            std::process::exit(1);
        }
    } else {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let project_names: Vec<String> = config.projects.keys().cloned().collect();
            let mut model = tui::Model::new("sync".to_string(), "all".to_string());
            
            // Pre-add tasks to model so they show up immediately
            for proj_name in &project_names {
                model.add_task(proj_name.clone());
            }
            
            let app = TuiApp::new(model);
            let tx = app.get_sender();
            
            // Spawn sync task
            let tx_clone = tx.clone();
            let sanctuary = config.sanctuary.clone();
            let projects = config.projects.clone();
            tokio::spawn(async move {
                // Update status for each project
                for (i, proj_name) in project_names.iter().enumerate() {
                    tx_clone.send(TuiEvent::UpdateStatus(i, TaskStatus::Running)).ok();
                    
                    // Run actual sync
                    if let Some(proj) = projects.get(proj_name) {
                        let result = sync::sync_project(&sanctuary, proj);
                        match result {
                            Ok(_) => {
                                tx_clone.send(TuiEvent::UpdateStatus(i, TaskStatus::Success)).ok();
                            }
                            Err(e) => {
                                tx_clone.send(TuiEvent::AppendOutput(i, format!("Error: {}", e))).ok();
                                tx_clone.send(TuiEvent::UpdateStatus(i, TaskStatus::Error)).ok();
                            }
                        }
                    } else {
                        tx_clone.send(TuiEvent::UpdateStatus(i, TaskStatus::Error)).ok();
                    }
                }
                
                // Warn about unknown repos
                if let Err(e) = sync::warn_unknown_repos(&sanctuary, &projects) {
                    tx_clone.send(TuiEvent::AppendOutput(0, format!("Warning: {}", e))).ok();
                }
                
                // Don't send Quit - let user press 'q' to exit
            });
            
            if let Err(e) = app.run().await {
                eprintln!("TUI error: {}", e);
                std::process::exit(1);
            }
        });
    }
}

fn run_seq(config_arg: Option<PathBuf>, name: String, plain: bool) {
    let config_path = get_config_path(config_arg);
    let config = load(&config_path).unwrap_or_else(|e| {
        eprintln!("Error: {}", e);
        std::process::exit(1);
    });
    
    if plain {
        let mut runner = Runner::new(config);
        if let Err(e) = runner.run_seq(&name) {
            eprintln!("Error: {}", e);
            std::process::exit(1);
        }
    } else {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let seq_decl = config.seqs.get(&name)
                .ok_or_else(|| format!("unknown seq: {}", name))
                .unwrap();
            
            // Extract function calls from the sequence with their indices
            let mut tasks: Vec<(usize, String, crate::ast::SeqStmt)> = Vec::new();
            if let crate::ast::Stmt::SeqDecl { stmts, .. } = seq_decl {
                for (i, stmt) in stmts.iter().enumerate() {
                    match stmt {
                        crate::ast::SeqStmt::FnCall { fn_name, project_name, .. } => {
                            tasks.push((i, format!("{}({})", fn_name, project_name), stmt.clone()));
                        }
                        crate::ast::SeqStmt::SeqRef { seq_name, .. } => {
                            tasks.push((i, format!("seq:{}", seq_name), stmt.clone()));
                        }
                    }
                }
            }
            
            let mut model = tui::Model::new("seq".to_string(), name.clone());
            
            // Pre-add tasks to model so they show up immediately
            for (_, task_name, _) in &tasks {
                model.add_task(task_name.clone());
            }
            
            let app = TuiApp::new(model);
            let tx = app.get_sender();
            
            // Spawn seq task
            let config_clone = config.clone();
            let tx_clone = tx.clone();
            tokio::spawn(async move {
                // Execute each function call sequentially with individual task updates
                for (task_idx, _task_name, stmt) in tasks {
                    tx_clone.send(TuiEvent::UpdateStatus(task_idx, TaskStatus::Running)).ok();
                    
                    let tx_clone_for_callback = tx_clone.clone();
                    let task_idx_for_callback = task_idx;
                    
                    let callback: crate::runner::OutputCallback = Box::new(move |line| {
                        tx_clone_for_callback.send(TuiEvent::AppendOutput(task_idx_for_callback, line)).ok();
                    });
                    
                    match &stmt {
                        crate::ast::SeqStmt::FnCall { fn_name, project_name, .. } => {
                            let mut runner = Runner::new(config_clone.clone()).with_output_callback(callback);
                            match runner.execute_fn_call(fn_name, project_name) {
                                Ok(_) => {
                                    tx_clone.send(TuiEvent::UpdateStatus(task_idx, TaskStatus::Success)).ok();
                                }
                                Err(e) => {
                                    tx_clone.send(TuiEvent::AppendOutput(task_idx, format!("Error: {}", e))).ok();
                                    tx_clone.send(TuiEvent::UpdateStatus(task_idx, TaskStatus::Error)).ok();
                                }
                            }
                        }
                        crate::ast::SeqStmt::SeqRef { seq_name, .. } => {
                            let mut runner = Runner::new(config_clone.clone()).with_output_callback(callback);
                            match runner.run_seq(seq_name) {
                                Ok(_) => {
                                    tx_clone.send(TuiEvent::UpdateStatus(task_idx, TaskStatus::Success)).ok();
                                }
                                Err(e) => {
                                    tx_clone.send(TuiEvent::AppendOutput(task_idx, format!("Error: {}", e))).ok();
                                    tx_clone.send(TuiEvent::UpdateStatus(task_idx, TaskStatus::Error)).ok();
                                }
                            }
                        }
                    }
                }
                
                // Don't send Quit - let user press 'q' to exit
            });
            
            if let Err(e) = app.run().await {
                eprintln!("TUI error: {}", e);
                std::process::exit(1);
            }
        });
    }
}

fn run_par(config_arg: Option<PathBuf>, name: String, plain: bool) {
    let config_path = get_config_path(config_arg);
    let config = load(&config_path).unwrap_or_else(|e| {
        eprintln!("Error: {}", e);
        std::process::exit(1);
    });
    
    if plain {
        let mut runner = Runner::new(config);
        if let Err(e) = runner.run_par(&name) {
            eprintln!("Error: {}", e);
            std::process::exit(1);
        }
    } else {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let par_decl = config.pars.get(&name)
                .ok_or_else(|| format!("unknown par: {}", name))
                .unwrap();
            
            // Extract function calls from the parallel with their indices
            let mut tasks: Vec<(usize, String, crate::ast::ParStmt)> = Vec::new();
            if let crate::ast::Stmt::ParDecl { stmts, .. } = par_decl {
                for (i, stmt) in stmts.iter().enumerate() {
                    match stmt {
                        crate::ast::ParStmt::FnCall { fn_name, project_name, .. } => {
                            tasks.push((i, format!("{}({})", fn_name, project_name), stmt.clone()));
                        }
                        crate::ast::ParStmt::SeqRef { seq_name, .. } => {
                            tasks.push((i, format!("seq:{}", seq_name), stmt.clone()));
                        }
                    }
                }
            }
            
            let mut model = tui::Model::new("par".to_string(), name.clone());
            
            // Pre-add tasks to model so they show up immediately
            for (_, task_name, _) in &tasks {
                model.add_task(task_name.clone());
            }
            
            let app = TuiApp::new(model);
            let tx = app.get_sender();
            
            // Spawn par task
            let config_clone = config.clone();
            let tx_clone = tx.clone();
            tokio::spawn(async move {
                // Execute each function call in parallel with individual task updates
                let mut join_handles = Vec::new();
                
                for (task_idx, _task_name, stmt) in tasks {
                    let tx_clone = tx_clone.clone();
                    let config_clone = config_clone.clone();
                    
                    let handle = tokio::spawn(async move {
                        tx_clone.send(TuiEvent::UpdateStatus(task_idx, TaskStatus::Running)).ok();
                        
                        let tx_clone_for_callback = tx_clone.clone();
                        let task_idx_for_callback = task_idx;
                        
                        let callback: crate::runner::OutputCallback = Box::new(move |line| {
                            tx_clone_for_callback.send(TuiEvent::AppendOutput(task_idx_for_callback, line)).ok();
                        });
                        
                        match &stmt {
                            crate::ast::ParStmt::FnCall { fn_name, project_name, .. } => {
                                let mut runner = Runner::new(config_clone).with_output_callback(callback);
                                match runner.execute_fn_call(fn_name, project_name) {
                                    Ok(_) => {
                                        tx_clone.send(TuiEvent::UpdateStatus(task_idx, TaskStatus::Success)).ok();
                                    }
                                    Err(e) => {
                                        tx_clone.send(TuiEvent::AppendOutput(task_idx, format!("Error: {}", e))).ok();
                                        tx_clone.send(TuiEvent::UpdateStatus(task_idx, TaskStatus::Error)).ok();
                                    }
                                }
                            }
                            crate::ast::ParStmt::SeqRef { seq_name, .. } => {
                                let mut runner = Runner::new(config_clone).with_output_callback(callback);
                                match runner.run_seq(seq_name) {
                                    Ok(_) => {
                                        tx_clone.send(TuiEvent::UpdateStatus(task_idx, TaskStatus::Success)).ok();
                                    }
                                    Err(e) => {
                                        tx_clone.send(TuiEvent::AppendOutput(task_idx, format!("Error: {}", e))).ok();
                                        tx_clone.send(TuiEvent::UpdateStatus(task_idx, TaskStatus::Error)).ok();
                                    }
                                }
                            }
                        }
                    });
                    
                    join_handles.push(handle);
                }
                
                // Wait for all parallel tasks to complete
                for handle in join_handles {
                    handle.await.ok();
                }
                
                // Don't send Quit - let user press 'q' to exit
            });
            
            if let Err(e) = app.run().await {
                eprintln!("TUI error: {}", e);
                std::process::exit(1);
            }
        });
    }
}

fn run_version() {
    println!("sro 0.0.1");
}
