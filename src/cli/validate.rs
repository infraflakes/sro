use super::load_config;
use std::path::PathBuf;

pub fn run(config_arg: Option<PathBuf>) -> miette::Result<()> {
    let cfg = load_config(config_arg)?;
    println!("Config is valid!");
    println!("Shell: {}", cfg.shell);
    println!("Sanctuary: {}", cfg.sanctuary);
    println!("Projects: {}", cfg.projects.len());
    println!("Functions: {}", cfg.functions.len());
    println!("Seqs: {}", cfg.seqs.len());
    println!("Pars: {}", cfg.pars.len());
    Ok(())
}
