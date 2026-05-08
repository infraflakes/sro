mod exec;
mod sync;
mod validate;

use clap::{Parser, Subcommand};
use std::path::PathBuf;

fn print_parse_errors(reports: Vec<miette::Report>) -> miette::Report {
    let count = reports.len();
    eprintln!();
    for (i, report) in reports.into_iter().enumerate() {
        if i > 0 {
            eprintln!();
        }
        eprintln!("{:?}", report);
    }
    miette::miette!("{} parse error(s) found", count)
}

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

pub fn run() -> miette::Result<()> {
    let cli = Cli::parse();

    match cli.command {
        Commands::Validate => validate::run(cli.config),
        Commands::Sync => sync::run(cli.config, cli.plain),
        Commands::Seq { name } => exec::run_seq(cli.config, name, cli.plain),
        Commands::Par { name } => exec::run_par(cli.config, name, cli.plain),
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

fn run_version() -> miette::Result<()> {
    println!("sro 0.0.1");
    Ok(())
}
