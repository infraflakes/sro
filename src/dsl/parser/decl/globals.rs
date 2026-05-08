use super::super::*;
use crate::dsl::token::TokenType;

impl Parser {
    pub(crate) fn parse_shell_decl(&mut self) -> Result<Stmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'shell'
        self.expect(TokenType::Assign)?;

        let value = match &self.current_token().ty {
            TokenType::Backtick(s) => s.clone(),
            _ => {
                return Err(ParseError::new(
                    miette::SourceSpan::new(
                        self.current_token().offset.into(),
                        self.current_token().len,
                    ),
                    "expected backtick string".to_string(),
                ));
            }
        };
        self.advance();
        self.expect(TokenType::Semicolon)?;

        Ok(Stmt::ShellDecl { span, value })
    }

    pub(crate) fn parse_sanctuary_decl(&mut self) -> Result<Stmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'sanctuary'
        self.expect(TokenType::Assign)?;

        let value = self.parse_expr()?;
        self.expect(TokenType::Semicolon)?;

        Ok(Stmt::SanctuaryDecl { span, value })
    }

    pub(crate) fn parse_import_decl(&mut self) -> Result<Stmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'import'

        let path = match &self.current_token().ty {
            TokenType::PathLit(p) => p.clone(),
            _ => {
                return Err(ParseError::new(
                    miette::SourceSpan::new(
                        self.current_token().offset.into(),
                        self.current_token().len,
                    ),
                    "expected path literal (e.g. ./file.sro)".to_string(),
                ));
            }
        };
        self.advance();
        self.expect(TokenType::Semicolon)?;

        Ok(Stmt::ImportDecl { span, paths: vec![path] })
    }

    pub(crate) fn parse_var_decl(&mut self) -> Result<Stmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'var'

        let var_type = match self.current_token().ty {
            TokenType::StringKw => VarType::String,
            TokenType::Shell => VarType::Shell,
            _ => {
                return Err(ParseError::new(
                    miette::SourceSpan::new(
                        self.current_token().offset.into(),
                        self.current_token().len,
                    ),
                    "expected 'string' or 'shell'".to_string(),
                ));
            }
        };
        self.advance();

        let name = match &self.current_token().ty {
            TokenType::Ident(n) => n.clone(),
            _ => {
                return Err(ParseError::new(
                    miette::SourceSpan::new(
                        self.current_token().offset.into(),
                        self.current_token().len,
                    ),
                    "expected identifier".to_string(),
                ));
            }
        };
        self.advance();

        self.expect(TokenType::Assign)?;

        let value = self.parse_expr()?;

        self.expect(TokenType::Semicolon)?;

        Ok(Stmt::VarDecl {
            span,
            var_type,
            name,
            value,
        })
    }
}
