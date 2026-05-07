use crate::dsl::ast::Stmt;
use std::collections::HashMap;

#[derive(Debug, Clone)]
pub struct Project {
    pub name: String,
    pub url: String,
    pub dir: String,
    pub sync: String,
    pub use_file: Option<String>,
    pub branch: String,
}

#[derive(Debug, Clone)]
pub struct Config {
    pub shell: String,
    pub sanctuary: String,
    pub projects: HashMap<String, Project>,
    pub functions: HashMap<String, Stmt>,
    pub seqs: HashMap<String, Stmt>,
    pub pars: HashMap<String, Stmt>,
    pub vars: HashMap<String, String>,
}
