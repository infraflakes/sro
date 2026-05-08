use crate::config::load;
use std::path::PathBuf;
use super::{get_config_path, print_parse_errors};

pub fn run(config_arg: Option<PathBuf>) -> miette::Result<()> {
    let config_path = get_config_path(config_arg);

    match load(&config_path) {
        Ok(cfg) => {
            println!("Config is valid!");
            println!("Shell: {}", cfg.shell);
            println!("Sanctuary: {}", cfg.sanctuary);
            println!("Projects: {}", cfg.projects.len());
            println!("Functions: {}", cfg.functions.len());
            println!("Seqs: {}", cfg.seqs.len());
            println!("Pars: {}", cfg.pars.len());
            Ok(())
        }
        Err(e) => {
            if let crate::config::ConfigError::ParseReports(reports) = e {
                Err(print_parse_errors(reports))
            } else {
                Err(miette::miette!("{}", e))
            }
        }
    }
}
