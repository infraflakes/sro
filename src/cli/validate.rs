use super::load_config;
use std::path::PathBuf;

pub fn run(config_arg: Option<PathBuf>) -> miette::Result<()> {
    let cfg = load_config(config_arg)?;
    println!("Config is valid!");
    println!("Shell: {}", cfg.shell);
    println!("Sanctuary: {}", cfg.sanctuary);
    println!("Projects: {}", cfg.projects.len());
    for (name, proj) in &cfg.projects {
        println!(
            "  {}: {} fns, {} seqs, {} pars",
            name,
            proj.functions.len(),
            proj.seqs.len(),
            proj.pars.len()
        );
    }
    Ok(())
}
