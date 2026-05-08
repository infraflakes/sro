pub mod context;
pub mod resolver;

use crate::config::{Config, ConfigError};
use crate::dsl::ast::{ParStmt, SeqStmt, Stmt};
pub use context::{ExecContext, OutputCallback};
use std::io::{self, Write};

pub struct Runner {
    cfg: Config,
    writer: Box<dyn Write>,
    suppress_headers: bool,
    output_callback: Option<OutputCallback>,
}

impl Runner {
    pub fn new(cfg: Config) -> Self {
        Runner {
            cfg,
            writer: Box::new(io::stdout()),
            suppress_headers: false,
            output_callback: None,
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

    pub fn with_output_callback(mut self, callback: OutputCallback) -> Self {
        self.output_callback = Some(callback);
        self.writer = Box::new(io::sink()); // Suppress stdout when callback is set
        self
    }

    pub fn execute_fn_call(
        &mut self,
        fn_name: &str,
        project_name: &str,
    ) -> Result<(), ConfigError> {
        let fn_decl = self
            .cfg
            .functions
            .get(fn_name)
            .ok_or_else(|| ConfigError::Validation(format!("unknown function: {}", fn_name)))?
            .clone();

        let project = self
            .cfg
            .projects
            .get(project_name)
            .ok_or_else(|| ConfigError::Validation(format!("unknown project: {}", project_name)))?
            .clone();

        if !self.suppress_headers {
            let line = format!("{}({})", fn_name, project_name);
            if let Some(ref callback) = self.output_callback {
                callback(line);
            } else {
                writeln!(self.writer, "\x1b[38;2;91;156;246m{}\x1b[0m", line)
                    .map_err(|e| ConfigError::Validation(format!("write error: {}", e)))?;
            }
        }

        let mut ctx = ExecContext::new(
            &self.cfg,
            &project,
            &mut *self.writer,
            self.output_callback.as_ref(),
        );
        if let Stmt::FnDecl { body, .. } = &fn_decl {
            ctx.exec_fn_body(body)?;
        }
        Ok(())
    }

    pub fn run_seq(&mut self, seq_name: &str) -> Result<(), ConfigError> {
        let seq_decl = self
            .cfg
            .seqs
            .get(seq_name)
            .ok_or_else(|| ConfigError::Validation(format!("unknown seq: {}", seq_name)))?
            .clone();

        if !self.suppress_headers {
            let line = format!("seq {}", seq_name);
            if let Some(ref callback) = self.output_callback {
                callback(line);
            } else {
                writeln!(self.writer, "{}", line)
                    .map_err(|e| ConfigError::Validation(format!("write error: {}", e)))?;
            }
        }

        self.execute_seq(&seq_decl)
    }

    fn execute_seq(&mut self, seq_decl: &Stmt) -> Result<(), ConfigError> {
        if let Stmt::SeqDecl { stmts, .. } = seq_decl {
            for stmt in stmts {
                match stmt {
                    SeqStmt::FnCall {
                        fn_name,
                        project_name,
                        ..
                    } => {
                        self.execute_fn_call(fn_name, project_name)?;
                    }
                    SeqStmt::SeqRef { seq_name, .. } => {
                        let ref_seq = self
                            .cfg
                            .seqs
                            .get(seq_name)
                            .ok_or_else(|| {
                                ConfigError::Validation(format!("unknown seq: {}", seq_name))
                            })?
                            .clone();
                        self.execute_seq(&ref_seq)?;
                    }
                }
            }
        }
        Ok(())
    }

    pub fn run_par(&mut self, par_name: &str) -> Result<(), ConfigError> {
        let par_decl = self
            .cfg
            .pars
            .get(par_name)
            .ok_or_else(|| ConfigError::Validation(format!("unknown par: {}", par_name)))?
            .clone();

        if !self.suppress_headers {
            writeln!(self.writer, "par {}", par_name)
                .map_err(|e| ConfigError::Validation(format!("write error: {}", e)))?;
        }

        self.execute_par(&par_decl)
    }

    fn execute_par(&mut self, par_decl: &Stmt) -> Result<(), ConfigError> {
        // For now, execute sequentially
        if let Stmt::ParDecl { stmts, .. } = par_decl {
            for stmt in stmts {
                match stmt {
                    ParStmt::FnCall {
                        fn_name,
                        project_name,
                        ..
                    } => {
                        self.execute_fn_call(fn_name, project_name)?;
                    }
                    ParStmt::SeqRef { seq_name, .. } => {
                        let ref_seq = self
                            .cfg
                            .seqs
                            .get(seq_name)
                            .ok_or_else(|| {
                                ConfigError::Validation(format!("unknown seq: {}", seq_name))
                            })?
                            .clone();
                        self.execute_seq(&ref_seq)?;
                    }
                }
            }
        }
        Ok(())
    }
}
