mod fn_seq_par;
mod globals;
mod project;

use super::*;
use crate::dsl::token::TokenType;

impl Parser {
    pub(super) fn parse_stmt(&mut self) -> Result<Stmt, ParseError> {
        match self.current_token().ty {
            TokenType::Shell => self.parse_shell_decl(),
            TokenType::Sanctuary => self.parse_sanctuary_decl(),
            TokenType::Import => self.parse_import_decl(),
            TokenType::Var => self.parse_var_decl(),
            TokenType::Pr => self.parse_project_decl(),
            TokenType::Fn => self.parse_fn_decl(),
            TokenType::Seq => self.parse_seq_decl(),
            TokenType::Par => self.parse_par_decl(),
            _ => Err(ParseError::new(
                miette::SourceSpan::new(
                    self.current_token().offset.into(),
                    self.current_token().len,
                ),
                format!("unexpected token at top level: {:?}", self.current_token().ty),
            )),
        }
    }
}
