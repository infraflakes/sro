use crate::ast::{FnStmt, SeqStmt, ParStmt, Expr};
use crate::config::{Config, Project, ConfigError};
use std::collections::HashMap;
use std::io::{self, Write};
use std::path::PathBuf;
use std::process::Command;

pub struct Runner {
    cfg: Config,
    writer: Box<dyn Write>,
    suppress_headers: bool,
}

impl Runner {
    pub fn new(cfg: Config) -> Self {
        Runner {
            cfg,
            writer: Box::new(io::stdout()),
            suppress_headers: false,
        }
    }

    #[allow(dead_code)]
    pub fn with_writer(mut self, writer: Box<dyn Write>) -> Self {
        self.writer = writer;
        self
    }

    #[allow(dead_code)]
    pub fn suppress_headers(mut self, suppress: bool) -> Self {
        self.suppress_headers = suppress;
        self
    }

    pub fn execute_fn_call(&mut self, fn_name: &str, project_name: &str) -> Result<(), ConfigError> {
        let fn_decl = self.cfg.functions.get(fn_name)
            .ok_or_else(|| ConfigError::Validation(format!("unknown function: {}", fn_name)))?
            .clone();
        
        let project = self.cfg.projects.get(project_name)
            .ok_or_else(|| ConfigError::Validation(format!("unknown project: {}", project_name)))?
            .clone();

        if !self.suppress_headers {
            writeln!(self.writer, "\x1b[38;2;91;156;246m{}\x1b[0m({})", fn_name, project_name)
                .map_err(|e| ConfigError::Validation(format!("write error: {}", e)))?;
        }

        let mut ctx = ExecContext::new(&self.cfg, &project, &mut *self.writer);
        if let crate::ast::Stmt::FnDecl { body, .. } = &fn_decl {
            ctx.exec_fn_body(body)?;
        }
        Ok(())
    }

    pub fn run_seq(&mut self, seq_name: &str) -> Result<(), ConfigError> {
        let seq_decl = self.cfg.seqs.get(seq_name)
            .ok_or_else(|| ConfigError::Validation(format!("unknown seq: {}", seq_name)))?
            .clone();

        if !self.suppress_headers {
            writeln!(self.writer, "seq {}", seq_name)
                .map_err(|e| ConfigError::Validation(format!("write error: {}", e)))?;
        }

        self.execute_seq(&seq_decl)
    }

    fn execute_seq(&mut self, seq_decl: &crate::ast::Stmt) -> Result<(), ConfigError> {
        if let crate::ast::Stmt::SeqDecl { stmts, .. } = seq_decl {
            for stmt in stmts {
                match stmt {
                    SeqStmt::FnCall { fn_name, project_name, .. } => {
                        self.execute_fn_call(fn_name, project_name)?;
                    }
                    SeqStmt::SeqRef { seq_name, .. } => {
                        let ref_seq = self.cfg.seqs.get(seq_name)
                            .ok_or_else(|| ConfigError::Validation(format!("unknown seq: {}", seq_name)))?
                            .clone();
                        self.execute_seq(&ref_seq)?;
                    }
                }
            }
        }
        Ok(())
    }

    pub fn run_par(&mut self, par_name: &str) -> Result<(), ConfigError> {
        let par_decl = self.cfg.pars.get(par_name)
            .ok_or_else(|| ConfigError::Validation(format!("unknown par: {}", par_name)))?
            .clone();

        if !self.suppress_headers {
            writeln!(self.writer, "par {}", par_name)
                .map_err(|e| ConfigError::Validation(format!("write error: {}", e)))?;
        }

        self.execute_par(&par_decl)
    }

    fn execute_par(&mut self, par_decl: &crate::ast::Stmt) -> Result<(), ConfigError> {
        // For now, execute sequentially - will need async for true parallel execution
        if let crate::ast::Stmt::ParDecl { stmts, .. } = par_decl {
            for stmt in stmts {
                match stmt {
                    ParStmt::FnCall { fn_name, project_name, .. } => {
                        self.execute_fn_call(fn_name, project_name)?;
                    }
                    ParStmt::SeqRef { seq_name, .. } => {
                        let ref_seq = self.cfg.seqs.get(seq_name)
                            .ok_or_else(|| ConfigError::Validation(format!("unknown seq: {}", seq_name)))?
                            .clone();
                        self.execute_seq(&ref_seq)?;
                    }
                }
            }
        }
        Ok(())
    }
}

struct ExecContext<'a> {
    cfg: &'a Config,
    project: &'a Project,
    vars: HashMap<String, String>,
    env_stack: Vec<HashMap<String, String>>,
    work_dir: PathBuf,
    writer: &'a mut dyn Write,
}

impl<'a> ExecContext<'a> {
    fn new(cfg: &'a Config, project: &'a Project, writer: &'a mut dyn Write) -> Self {
        let mut vars = HashMap::new();
        for (k, v) in &cfg.vars {
            vars.insert(k.clone(), v.clone());
        }

        let base_dir = PathBuf::from(&cfg.sanctuary).join(&project.dir);

        ExecContext {
            cfg,
            project,
            vars,
            env_stack: Vec::new(),
            work_dir: base_dir,
            writer,
        }
    }

    fn exec_fn_body(&mut self, body: &[FnStmt]) -> Result<(), ConfigError> {
        for stmt in body {
            match stmt {
                FnStmt::Log { value, .. } => self.exec_log(value)?,
                FnStmt::Exec { value, .. } => self.exec_exec(value)?,
                FnStmt::Cd { arg, .. } => self.exec_cd(arg)?,
                FnStmt::VarDecl { name, value, var_type, .. } => self.exec_var_decl(name, value, var_type)?,
                FnStmt::EnvBlock { pairs, body: block_body, .. } => self.exec_env_block(pairs, block_body)?,
            }
        }
        Ok(())
    }

    fn exec_log(&mut self, value: &Expr) -> Result<(), ConfigError> {
        let msg = self.resolve_expr(value)?;
        let indent = "  ".repeat(self.env_stack.len());
        writeln!(self.writer, "{}\x1b[38;2;255;203;107mlog  {}\x1b[0m", indent, msg)
            .map_err(|e| ConfigError::Validation(format!("write error: {}", e)))?;
        Ok(())
    }

    fn exec_exec(&mut self, value: &Expr) -> Result<(), ConfigError> {
        let cmd_str = self.resolve_expr(value)?;
        let indent = "  ".repeat(self.env_stack.len());
        writeln!(self.writer, "{}\x1b[38;2;91;156;246mexec {}\x1b[0m", indent, cmd_str)
            .map_err(|e| ConfigError::Validation(format!("write error: {}", e)))?;

        let output = Command::new(&self.cfg.shell)
            .arg("-c")
            .arg(&cmd_str)
            .current_dir(&self.work_dir)
            .envs(self.build_env())
            .output()
            .map_err(|e| ConfigError::Validation(format!("exec failed: {}: {}", cmd_str, e)))?;

        if !output.status.success() {
            return Err(ConfigError::Validation(format!(
                "exec failed with exit code: {}",
                output.status.code().unwrap_or(-1)
            )));
        }

        let stdout = String::from_utf8_lossy(&output.stdout);
        let stderr = String::from_utf8_lossy(&output.stderr);
        
        if !stdout.is_empty() {
            let stdout_indent = "  ".repeat(self.env_stack.len() + 1);
            for line in stdout.lines() {
                writeln!(self.writer, "{}{}", stdout_indent, line)
                    .map_err(|e| ConfigError::Validation(format!("write error: {}", e)))?;
            }
        }
        
        if !stderr.is_empty() {
            let stderr_indent = "  ".repeat(self.env_stack.len() + 1);
            for line in stderr.lines() {
                writeln!(self.writer, "{}{}", stderr_indent, line)
                    .map_err(|e| ConfigError::Validation(format!("write error: {}", e)))?;
            }
        }

        Ok(())
    }

    fn exec_cd(&mut self, arg: &str) -> Result<(), ConfigError> {
        let base_dir = PathBuf::from(&self.cfg.sanctuary).join(&self.project.dir);
        self.work_dir = if arg == "." {
            base_dir
        } else {
            base_dir.join(arg)
        };

        if !self.work_dir.exists() {
            return Err(ConfigError::Validation(format!("cd {}: directory does not exist", arg)));
        }

        let indent = "  ".repeat(self.env_stack.len());
        writeln!(self.writer, "{}\x1b[38;2;255;203;107mcd   {}\x1b[0m", indent, arg)
            .map_err(|e| ConfigError::Validation(format!("write error: {}", e)))?;
        Ok(())
    }

    fn exec_var_decl(&mut self, name: &str, value: &Expr, var_type: &crate::ast::VarType) -> Result<(), ConfigError> {
        let val = self.resolve_expr(value)?;
        
        if var_type == &crate::ast::VarType::Shell {
            let output = Command::new(&self.cfg.shell)
                .arg("-c")
                .arg(&val)
                .current_dir(&self.work_dir)
                .envs(self.build_env())
                .output()
                .map_err(|e| ConfigError::Validation(format!("shell execution failed for var {}: {}", name, e)))?;

            if !output.status.success() {
                return Err(ConfigError::Validation(format!("shell execution failed for var {}", name)));
            }

            let result = String::from_utf8_lossy(&output.stdout).trim_end().to_string();
            self.vars.insert(name.to_string(), result);
        } else {
            self.vars.insert(name.to_string(), val);
        }
        Ok(())
    }

    fn exec_env_block(&mut self, pairs: &[crate::ast::EnvPair], body: &[FnStmt]) -> Result<(), ConfigError> {
        let mut layer = HashMap::new();
        for pair in pairs {
            let val = self.resolve_expr(&pair.value)?;
            layer.insert(pair.key.clone(), val);
        }

        let keys: Vec<&str> = pairs.iter().map(|p| p.key.as_str()).collect();
        let indent = "  ".repeat(self.env_stack.len());
        writeln!(self.writer, "{}\x1b[38;2;199;146;234menv  {}\x1b[0m", indent, keys.join(", "))
            .map_err(|e| ConfigError::Validation(format!("write error: {}", e)))?;

        self.env_stack.push(layer);

        let saved_vars = self.vars.clone();
        let result = self.exec_fn_body(body);

        self.vars = saved_vars;
        self.env_stack.pop();

        result
    }

    fn resolve_expr(&self, expr: &Expr) -> Result<String, ConfigError> {
        match expr {
            Expr::BacktickLit { parts, .. } => {
                let mut result = String::new();
                for part in parts {
                    if part.is_var {
                        let var_name = part.value.trim_start_matches('$');
                        if let Some(value) = self.vars.get(var_name) {
                            result.push_str(value);
                        } else {
                            return Err(ConfigError::Validation(format!("undefined variable: ${}", var_name)));
                        }
                    } else {
                        result.push_str(&part.value);
                    }
                }
                Ok(result)
            }
            Expr::VarRef { name, .. } => {
                if let Some(value) = self.vars.get(name) {
                    Ok(value.clone())
                } else {
                    Err(ConfigError::Validation(format!("undefined variable: ${}", name)))
                }
            }
        }
    }

    fn build_env(&self) -> Vec<(String, String)> {
        let mut env: HashMap<String, String> = HashMap::new();
        
        // Copy current environment
        for (key, value) in std::env::vars() {
            env.insert(key, value);
        }

        // Apply env stack layers
        for layer in &self.env_stack {
            for (key, value) in layer {
                env.insert(key.clone(), value.clone());
            }
        }

        env.into_iter().collect()
    }
}
