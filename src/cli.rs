use crate::config::load;
use crate::runner::Runner;
use crate::sync;
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
        Commands::Sync => run_sync(cli.config),
        Commands::Seq { name } => run_seq(cli.config, name),
        Commands::Par { name } => run_par(cli.config, name),
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

fn run_sync(config_arg: Option<PathBuf>) {
    let config_path = get_config_path(config_arg);
    let config = load(&config_path).unwrap_or_else(|e| {
        eprintln!("Error: {}", e);
        std::process::exit(1);
    });
    
    let mut stdout = io::stdout();
    if let Err(e) = sync::sync_all(&config, &mut stdout) {
        eprintln!("Error: {}", e);
        std::process::exit(1);
    }
}

fn run_seq(config_arg: Option<PathBuf>, name: String) {
    let config_path = get_config_path(config_arg);
    let config = load(&config_path).unwrap_or_else(|e| {
        eprintln!("Error: {}", e);
        std::process::exit(1);
    });
    let mut runner = Runner::new(config);
    if let Err(e) = runner.run_seq(&name) {
        eprintln!("Error: {}", e);
        std::process::exit(1);
    }
}

fn run_par(config_arg: Option<PathBuf>, name: String) {
    let config_path = get_config_path(config_arg);
    let config = load(&config_path).unwrap_or_else(|e| {
        eprintln!("Error: {}", e);
        std::process::exit(1);
    });
    let mut runner = Runner::new(config);
    if let Err(e) = runner.run_par(&name) {
        eprintln!("Error: {}", e);
        std::process::exit(1);
    }
}

fn run_version() {
    println!("sro 0.0.1");
}
