mod fn_body;
mod seq_par_body;

use super::*;
use crate::dsl::token::TokenType;

impl Parser {
    pub(crate) fn parse_fn_stmt(&mut self) -> Result<FnStmt, ParseError> {
        match self.current_token().ty {
            TokenType::Log => self.parse_log_stmt(),
            TokenType::Exec => self.parse_exec_stmt(),
            TokenType::Cd => self.parse_cd_stmt(),
            TokenType::Var => self.parse_fn_var_decl(),
            TokenType::Env => self.parse_env_block(),
            _ => Err(ParseError::new(
                miette::SourceSpan::new(
                    self.current_token().offset.into(),
                    self.current_token().len,
                ),
                format!("unexpected token in fn body: {:?}", self.current_token().ty),
            )),
        }
    }
}
