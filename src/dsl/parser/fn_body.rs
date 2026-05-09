use super::*;
use crate::dsl::token::TokenType;

impl Parser {
    pub(crate) fn parse_log_stmt(&mut self) -> Result<FnStmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'log'

        self.expect(TokenType::LParen)?;
        let value = self.parse_expr()?;
        self.expect(TokenType::RParen)?;
        self.expect(TokenType::Semicolon)?;

        Ok(FnStmt::Log { span, value })
    }

    pub(crate) fn parse_exec_stmt(&mut self) -> Result<FnStmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'exec'

        self.expect(TokenType::LParen)?;
        let value = self.parse_expr()?;
        self.expect(TokenType::RParen)?;
        self.expect(TokenType::Semicolon)?;

        Ok(FnStmt::Exec { span, value })
    }

    pub(crate) fn parse_cd_stmt(&mut self) -> Result<FnStmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'cd'

        self.expect(TokenType::LParen)?;

        let arg = match &self.current_token().ty {
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

        self.expect(TokenType::RParen)?;
        self.expect(TokenType::Semicolon)?;

        Ok(FnStmt::Cd { span, arg })
    }

    pub(crate) fn parse_fn_var_decl(&mut self) -> Result<FnStmt, ParseError> {
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

        Ok(FnStmt::VarDecl {
            span,
            var_type,
            name,
            value,
        })
    }

    pub(crate) fn parse_env_block(&mut self) -> Result<FnStmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'env'

        self.expect(TokenType::LBracket)?;

        let mut pairs = Vec::new();
        while self.current_token().ty != TokenType::RBracket {
            let key = match &self.current_token().ty {
                TokenType::Ident(k) => k.clone(),
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

            pairs.push(EnvPair { key, value });

            if self.current_token().ty == TokenType::Comma {
                self.advance();
            }
        }

        self.expect(TokenType::RBracket)?;
        self.expect(TokenType::LBrace)?;

        let mut body = Vec::new();
        while self.current_token().ty != TokenType::RBrace {
            body.push(self.parse_fn_stmt()?);
        }

        self.expect(TokenType::RBrace)?;
        self.expect(TokenType::Semicolon)?;

        Ok(FnStmt::EnvBlock { span, pairs, body })
    }
}
