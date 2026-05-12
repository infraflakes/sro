use super::*;

impl Parser {
    pub(crate) fn parse_log_stmt(&mut self) -> Result<FnStmt, ParseError> {
        self.advance();

        self.expect(TokenType::LParen)?;
        let value = self.parse_expr()?;
        self.expect(TokenType::RParen)?;
        self.expect(TokenType::Semicolon)?;

        Ok(FnStmt::Log { value })
    }

    pub(crate) fn parse_exec_stmt(&mut self) -> Result<FnStmt, ParseError> {
        self.advance();

        self.expect(TokenType::LParen)?;
        let value = self.parse_expr()?;
        self.expect(TokenType::RParen)?;
        self.expect(TokenType::Semicolon)?;

        Ok(FnStmt::Exec { value })
    }

    pub(crate) fn parse_cd_stmt(&mut self) -> Result<FnStmt, ParseError> {
        self.advance();

        self.expect(TokenType::LParen)?;
        let arg = self.parse_simple_backtick()?;
        self.expect(TokenType::RParen)?;
        self.expect(TokenType::Semicolon)?;

        Ok(FnStmt::Cd { arg })
    }

    pub(crate) fn parse_fn_var_decl(&mut self) -> Result<FnStmt, ParseError> {
        self.advance();

        let var_type = match &self.current_token().ty {
            TokenType::StringKw => {
                self.advance();
                VarType::String
            }
            TokenType::Shell => {
                self.advance();
                VarType::Shell
            }
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
            var_type,
            name,
            value,
        })
    }

    pub(crate) fn parse_env_block(&mut self) -> Result<FnStmt, ParseError> {
        self.advance();

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
                        "expected identifier in env pair".to_string(),
                    ));
                }
            };
            self.advance();

            self.expect(TokenType::Assign)?;

            let value = self.parse_expr()?;
            pairs.push(EnvPair { key, value });

            match &self.current_token().ty {
                TokenType::Comma => {
                    self.advance();
                }
                TokenType::RBracket => break,
                _ => {
                    return Err(ParseError::new(
                        miette::SourceSpan::new(
                            self.current_token().offset.into(),
                            self.current_token().len,
                        ),
                        "expected `,` or `]`".to_string(),
                    ));
                }
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

        Ok(FnStmt::EnvBlock { pairs, body })
    }
}
