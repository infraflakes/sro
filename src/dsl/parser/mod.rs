use crate::dsl::ast::*;
use crate::dsl::lexer::Lexer;
use crate::dsl::token::{Token, TokenType};
use miette::{Diagnostic, SourceSpan};
use thiserror::Error;

mod expr;
mod decl;
mod body;

#[derive(Debug, Error, Diagnostic)]
#[error("{msg}")]
pub struct ParseError {
    #[label("{msg}")]
    span: SourceSpan,
    msg: String,
}

impl ParseError {
    fn new(span: SourceSpan, msg: String) -> Self {
        Self { span, msg }
    }
}

fn format_token_type(ty: &TokenType) -> String {
    match ty {
        TokenType::LBrace => "`{`".to_string(),
        TokenType::RBrace => "`}`".to_string(),
        TokenType::LParen => "`(`".to_string(),
        TokenType::RParen => "`)`".to_string(),
        TokenType::LBracket => "`[`".to_string(),
        TokenType::RBracket => "`]`".to_string(),
        TokenType::Semicolon => "`;`".to_string(),
        TokenType::Comma => "`,`".to_string(),
        TokenType::Assign => "`=`".to_string(),
        TokenType::Dollar => "`$`".to_string(),
        TokenType::Dot => "`.`".to_string(),
        TokenType::Shell => "`shell`".to_string(),
        TokenType::Sanctuary => "`sanctuary`".to_string(),
        TokenType::Import => "`import`".to_string(),
        TokenType::Var => "`var`".to_string(),
        TokenType::StringKw => "`string`".to_string(),
        TokenType::ShellKw => "`shell`".to_string(),
        TokenType::Pr => "`pr`".to_string(),
        TokenType::Fn => "`fn`".to_string(),
        TokenType::Seq => "`seq`".to_string(),
        TokenType::Par => "`par`".to_string(),
        TokenType::Env => "`env`".to_string(),
        TokenType::Log => "`log`".to_string(),
        TokenType::Exec => "`exec`".to_string(),
        TokenType::Cd => "`cd`".to_string(),
        TokenType::Use => "`use`".to_string(),
        TokenType::Ident(_) => "identifier".to_string(),
        TokenType::Backtick(_) => "backtick string".to_string(),
        TokenType::String(_) => "string literal".to_string(),
        TokenType::PathLit(_) => "path literal".to_string(),
        TokenType::Illegal(_) => "illegal token".to_string(),
        TokenType::EOF => "end of file".to_string(),
        TokenType::Colon => "`:`".to_string(),
        TokenType::Equal => "`==`".to_string(),
    }
}

fn format_token(token: &Token) -> String {
    match &token.ty {
        TokenType::Ident(s) => format!("`{}`", s),
        TokenType::Backtick(s) => format!("`{}`", s),
        TokenType::String(s) => format!("\"{}\"", s),
        TokenType::PathLit(s) => format!("`{}`", s),
        TokenType::Illegal(s) => format!("`{}`", s),
        _ => format_token_type(&token.ty),
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
            let expected = format_token_type(&ty);
            let found = format_token(&token);
            Err(ParseError::new(
                SourceSpan::new(token.offset.into(), token.len),
                format!("expected {}, found {}", expected, found),
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
                    self.skip_to_stmt_boundary();
                }
            }
        }

        if errors.is_empty() {
            Ok(program)
        } else {
            Err(errors)
        }
    }

    fn skip_to_stmt_boundary(&mut self) {
        use TokenType::*;
        loop {
            match &self.current_token().ty {
                EOF => break,
                Semicolon | RBrace => {
                    self.advance();
                    // Continue to skip more boundary tokens
                }
                Shell | Sanctuary | Import | Var | Pr | Fn | Seq | Par => break,
                _ => self.advance(),
            }
        }
    }

}
