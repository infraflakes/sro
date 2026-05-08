use miette::Report;
use std::fmt;

#[derive(Debug)]
pub enum ConfigError {
    Io(std::io::Error),
    ParseReports(Vec<Report>),
    CircularImport(String),
    Validation(String),
}

impl fmt::Display for ConfigError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ConfigError::Io(e) => write!(f, "IO error: {}", e),
            ConfigError::ParseReports(reports) => {
                for report in reports {
                    write!(f, "{}", report)?;
                }
                Ok(())
            }
            ConfigError::CircularImport(s) => write!(f, "Circular import detected: {}", s),
            ConfigError::Validation(s) => write!(f, "Validation error: {}", s),
        }
    }
}

impl std::error::Error for ConfigError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            ConfigError::Io(e) => Some(e),
            _ => None,
        }
    }
}

impl From<std::io::Error> for ConfigError {
    fn from(e: std::io::Error) -> Self {
        ConfigError::Io(e)
    }
}
