use crate::config::{Config, Project, ConfigError};
use std::fs;
use std::io::Write;
use std::path::PathBuf;
use std::process::Command;

pub fn sync_all(cfg: &Config, writer: &mut dyn Write) -> Result<(), ConfigError> {
    // Create sanctuary directory
    fs::create_dir_all(&cfg.sanctuary)
        .map_err(|e| ConfigError::Validation(format!("cannot create sanctuary {}: {}", cfg.sanctuary, e)))?;

    // Sync each project
    for proj in cfg.projects.values() {
        sync_project(cfg, proj, writer)?;
    }

    // Warn about unknown repos
    warn_unknown_repos(cfg, writer)?;

    Ok(())
}

pub fn sync_project(cfg: &Config, proj: &Project, writer: &mut dyn Write) -> Result<(), ConfigError> {
    if proj.sync == "ignore" {
        writeln!(writer, "  skip  {} (sync=ignore)", proj.name)
            .map_err(|e| ConfigError::Validation(format!("write error: {}", e)))?;
        return Ok(());
    }

    let target_dir = PathBuf::from(&cfg.sanctuary).join(&proj.dir);
    let git_dir = target_dir.join(".git");

    if git_dir.exists() {
        writeln!(writer, "  exists  {} → {}", proj.name, target_dir.display())
            .map_err(|e| ConfigError::Validation(format!("write error: {}", e)))?;
        return Ok(());
    }

    writeln!(writer, "  clone  {} → {}", proj.name, target_dir.display())
        .map_err(|e| ConfigError::Validation(format!("write error: {}", e)))?;

    let target_dir_str = target_dir.to_string_lossy().to_string();
    let args = if proj.branch.is_empty() {
        vec!["clone", &proj.url, &target_dir_str]
    } else {
        vec!["clone", "-b", &proj.branch, &proj.url, &target_dir_str]
    };

    let output = Command::new("git")
        .args(&args)
        .output()
        .map_err(|e| ConfigError::Validation(format!("failed to clone {}: git command failed: {}", proj.name, e)))?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(ConfigError::Validation(format!("failed to clone {}: {}", proj.name, stderr)));
    }

    // Write clone output
    let stdout = String::from_utf8_lossy(&output.stdout);
    if !stdout.is_empty() {
        for line in stdout.lines() {
            writeln!(writer, "    {}", line)
                .map_err(|e| ConfigError::Validation(format!("write error: {}", e)))?;
        }
    }

    Ok(())
}

fn warn_unknown_repos(cfg: &Config, writer: &mut dyn Write) -> Result<(), ConfigError> {
    let mut known_dirs = std::collections::HashSet::new();
    for proj in cfg.projects.values() {
        known_dirs.insert(&proj.dir);
    }

    let entries = match fs::read_dir(&cfg.sanctuary) {
        Ok(e) => e,
        Err(_) => return Ok(()),
    };

    for entry in entries {
        let entry = entry.map_err(|e| ConfigError::Validation(format!("failed to read sanctuary: {}", e)))?;
        if !entry.path().is_dir() {
            continue;
        }
        let name = entry.file_name();
        let name_str = name.to_string_lossy().to_string();
        if known_dirs.contains(&name_str) {
            continue;
        }
        let git_dir = PathBuf::from(&cfg.sanctuary).join(&name_str).join(".git");
        if git_dir.exists() {
            writeln!(writer, "  warn  {}/{} is a git repo not in your config", cfg.sanctuary, name_str)
                .map_err(|e| ConfigError::Validation(format!("write error: {}", e)))?;
        }
    }

    Ok(())
}
