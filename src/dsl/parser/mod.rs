use crate::dsl::ast::*;
use crate::dsl::lexer::Lexer;
use crate::dsl::token::{Token, TokenType};
use miette::{Diagnostic, SourceSpan};
use thiserror::Error;

mod expr;
mod stmt;

#[derive(Debug, Error, Diagnostic)]
#[error("Parse error")]
#[diagnostic(code(sro::parse_error))]
pub struct ParseError {
    #[label]
    span: SourceSpan,
    msg: String,
}

impl ParseError {
    fn new(span: SourceSpan, msg: String) -> Self {
        Self { span, msg }
    }
}

pub struct Parser {
    lexer: Lexer,
    current: Token,
}

impl Parser {
    pub fn new(mut lexer: Lexer) -> Self {
        let current = lexer.next_token();
        Parser { lexer, current }
    }

    fn current_token(&self) -> &Token {
        &self.current
    }

    #[allow(dead_code)]
    fn peek_token(&self) -> &Token {
        &self.current
    }

    fn advance(&mut self) {
        self.current = self.lexer.next_token();
    }

    fn expect(&mut self, ty: TokenType) -> Result<Token, ParseError> {
        let token = self.current_token().clone();
        if std::mem::discriminant(&token.ty) == std::mem::discriminant(&ty) {
            self.advance();
            Ok(token)
        } else {
            Err(ParseError::new(
                SourceSpan::new(token.line.into(), 1),
                format!("expected {:?}, found {:?}", ty, token.ty),
            ))
        }
    }

    pub fn parse(&mut self) -> Result<Program, Vec<ParseError>> {
        let mut program = Program::new();
        let mut errors = Vec::new();

        while self.current_token().ty != TokenType::EOF {
            match self.parse_stmt() {
                Ok(stmt) => program.stmts.push(stmt),
                Err(e) => {
                    errors.push(e);
                    self.advance();
                }
            }
        }

        if errors.is_empty() {
            Ok(program)
        } else {
            Err(errors)
        }
    }
}
