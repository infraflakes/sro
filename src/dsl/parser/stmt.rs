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
                miette::SourceSpan::new(self.current_token().line.into(), 1),
                format!("unexpected token: {:?}", self.current_token().ty),
            )),
        }
    }

    fn parse_shell_decl(&mut self) -> Result<Stmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'shell'
        self.expect(TokenType::Assign)?;

        let value = match &self.current_token().ty {
            TokenType::Backtick(s) => s.clone(),
            _ => {
                return Err(ParseError::new(
                    miette::SourceSpan::new(self.current_token().line.into(), 1),
                    "expected backtick string".to_string(),
                ));
            }
        };
        self.advance();

        self.expect(TokenType::Semicolon)?;

        Ok(Stmt::ShellDecl { span, value })
    }

    fn parse_sanctuary_decl(&mut self) -> Result<Stmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'sanctuary'
        self.expect(TokenType::Assign)?;

        let value = self.parse_expr()?;

        self.expect(TokenType::Semicolon)?;

        Ok(Stmt::SanctuaryDecl { span, value })
    }

    fn parse_import_decl(&mut self) -> Result<Stmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'import'

        self.expect(TokenType::LBracket)?;

        let mut paths = Vec::new();
        while self.current_token().ty != TokenType::RBracket {
            match &self.current_token().ty {
                TokenType::PathLit(p) => {
                    paths.push(p.clone());
                    self.advance();
                }
                _ => {
                    return Err(ParseError::new(
                        miette::SourceSpan::new(self.current_token().line.into(), 1),
                        "expected path literal".to_string(),
                    ));
                }
            }

            if self.current_token().ty == TokenType::Comma {
                self.advance();
            }
        }

        self.expect(TokenType::RBracket)?;
        self.expect(TokenType::Semicolon)?;

        Ok(Stmt::ImportDecl { span, paths })
    }

    fn parse_var_decl(&mut self) -> Result<Stmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'var'

        let var_type = match self.current_token().ty {
            TokenType::StringKw => VarType::String,
            TokenType::Shell => VarType::Shell,
            _ => {
                return Err(ParseError::new(
                    miette::SourceSpan::new(self.current_token().line.into(), 1),
                    "expected 'string' or 'shell'".to_string(),
                ));
            }
        };
        self.advance();

        let name = match &self.current_token().ty {
            TokenType::Ident(n) => n.clone(),
            _ => {
                return Err(ParseError::new(
                    miette::SourceSpan::new(self.current_token().line.into(), 1),
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

    fn parse_project_decl(&mut self) -> Result<Stmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'pr'

        let name = match &self.current_token().ty {
            TokenType::Ident(n) => n.clone(),
            _ => {
                return Err(ParseError::new(
                    miette::SourceSpan::new(self.current_token().line.into(), 1),
                    "expected identifier".to_string(),
                ));
            }
        };
        self.advance();

        self.expect(TokenType::LBrace)?;

        let mut fields = Vec::new();
        while self.current_token().ty != TokenType::RBrace {
            let key = match &self.current_token().ty {
                TokenType::Ident(k) => k.clone(),
                _ => {
                    return Err(ParseError::new(
                        miette::SourceSpan::new(self.current_token().line.into(), 1),
                        "expected identifier".to_string(),
                    ));
                }
            };
            self.advance();

            self.expect(TokenType::Assign)?;

            let value = self.parse_expr()?;

            fields.push(ProjectField { key, value });

            self.expect(TokenType::Semicolon)?;
        }

        self.expect(TokenType::RBrace)?;

        Ok(Stmt::ProjectDecl { span, name, fields })
    }

    fn parse_fn_decl(&mut self) -> Result<Stmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'fn'

        let name = match &self.current_token().ty {
            TokenType::Ident(n) => n.clone(),
            _ => {
                return Err(ParseError::new(
                    miette::SourceSpan::new(self.current_token().line.into(), 1),
                    "expected identifier".to_string(),
                ));
            }
        };
        self.advance();

        self.expect(TokenType::LBrace)?;

        let mut body = Vec::new();
        while self.current_token().ty != TokenType::RBrace {
            body.push(self.parse_fn_stmt()?);
        }

        self.expect(TokenType::RBrace)?;

        Ok(Stmt::FnDecl { span, name, body })
    }

    fn parse_seq_decl(&mut self) -> Result<Stmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'seq'

        let name = match &self.current_token().ty {
            TokenType::Ident(n) => n.clone(),
            _ => {
                return Err(ParseError::new(
                    miette::SourceSpan::new(self.current_token().line.into(), 1),
                    "expected identifier".to_string(),
                ));
            }
        };
        self.advance();

        self.expect(TokenType::LBrace)?;

        let mut stmts = Vec::new();
        while self.current_token().ty != TokenType::RBrace {
            stmts.push(self.parse_seq_stmt()?);
        }

        self.expect(TokenType::RBrace)?;

        Ok(Stmt::SeqDecl { span, name, stmts })
    }

    fn parse_par_decl(&mut self) -> Result<Stmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'par'

        let name = match &self.current_token().ty {
            TokenType::Ident(n) => n.clone(),
            _ => {
                return Err(ParseError::new(
                    miette::SourceSpan::new(self.current_token().line.into(), 1),
                    "expected identifier".to_string(),
                ));
            }
        };
        self.advance();

        self.expect(TokenType::LBrace)?;

        let mut stmts = Vec::new();
        while self.current_token().ty != TokenType::RBrace {
            stmts.push(self.parse_par_stmt()?);
        }

        self.expect(TokenType::RBrace)?;

        Ok(Stmt::ParDecl { span, name, stmts })
    }

    pub(super) fn parse_fn_stmt(&mut self) -> Result<FnStmt, ParseError> {
        match self.current_token().ty {
            TokenType::Log => self.parse_log_stmt(),
            TokenType::Exec => self.parse_exec_stmt(),
            TokenType::Cd => self.parse_cd_stmt(),
            TokenType::Var => self.parse_fn_var_decl(),
            TokenType::Env => self.parse_env_block(),
            _ => Err(ParseError::new(
                miette::SourceSpan::new(self.current_token().line.into(), 1),
                format!("unexpected token in fn body: {:?}", self.current_token().ty),
            )),
        }
    }

    fn parse_log_stmt(&mut self) -> Result<FnStmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'log'

        self.expect(TokenType::LParen)?;
        let value = self.parse_expr()?;
        self.expect(TokenType::RParen)?;
        self.expect(TokenType::Semicolon)?;

        Ok(FnStmt::Log { span, value })
    }

    fn parse_exec_stmt(&mut self) -> Result<FnStmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'exec'

        self.expect(TokenType::LParen)?;
        let value = self.parse_expr()?;
        self.expect(TokenType::RParen)?;
        self.expect(TokenType::Semicolon)?;

        Ok(FnStmt::Exec { span, value })
    }

    fn parse_cd_stmt(&mut self) -> Result<FnStmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'cd'

        self.expect(TokenType::LParen)?;

        let arg = match &self.current_token().ty {
            TokenType::Backtick(s) => s.clone(),
            _ => {
                return Err(ParseError::new(
                    miette::SourceSpan::new(self.current_token().line.into(), 1),
                    "expected backtick string".to_string(),
                ));
            }
        };
        self.advance();

        self.expect(TokenType::RParen)?;
        self.expect(TokenType::Semicolon)?;

        Ok(FnStmt::Cd { span, arg })
    }

    fn parse_fn_var_decl(&mut self) -> Result<FnStmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'var'

        let var_type = match self.current_token().ty {
            TokenType::StringKw => VarType::String,
            TokenType::Shell => VarType::Shell,
            _ => {
                return Err(ParseError::new(
                    miette::SourceSpan::new(self.current_token().line.into(), 1),
                    "expected 'string' or 'shell'".to_string(),
                ));
            }
        };
        self.advance();

        let name = match &self.current_token().ty {
            TokenType::Ident(n) => n.clone(),
            _ => {
                return Err(ParseError::new(
                    miette::SourceSpan::new(self.current_token().line.into(), 1),
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

    fn parse_env_block(&mut self) -> Result<FnStmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'env'

        self.expect(TokenType::LBracket)?;

        let mut pairs = Vec::new();
        while self.current_token().ty != TokenType::RBracket {
            let key = match &self.current_token().ty {
                TokenType::Ident(k) => k.clone(),
                _ => {
                    return Err(ParseError::new(
                        miette::SourceSpan::new(self.current_token().line.into(), 1),
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

    pub(super) fn parse_seq_stmt(&mut self) -> Result<SeqStmt, ParseError> {
        match &self.current_token().ty {
            TokenType::Ident(fn_name) => {
                let span = Span::new(self.current_token().line, self.current_token().col);
                let fn_name = fn_name.clone();
                self.advance();

                self.expect(TokenType::LParen)?;

                let project_name = match &self.current_token().ty {
                    TokenType::Ident(n) => n.clone(),
                    _ => {
                        return Err(ParseError::new(
                            miette::SourceSpan::new(self.current_token().line.into(), 1),
                            "expected identifier".to_string(),
                        ));
                    }
                };
                self.advance();

                self.expect(TokenType::RParen)?;
                self.expect(TokenType::Semicolon)?;

                Ok(SeqStmt::FnCall {
                    span,
                    fn_name,
                    project_name,
                })
            }
            TokenType::Seq => {
                let span = Span::new(self.current_token().line, self.current_token().col);
                self.advance();
                self.expect(TokenType::Dot)?;

                let seq_name = match &self.current_token().ty {
                    TokenType::Ident(n) => n.clone(),
                    _ => {
                        return Err(ParseError::new(
                            miette::SourceSpan::new(self.current_token().line.into(), 1),
                            "expected identifier".to_string(),
                        ));
                    }
                };
                self.advance();

                self.expect(TokenType::Semicolon)?;

                Ok(SeqStmt::SeqRef { span, seq_name })
            }
            _ => Err(ParseError::new(
                miette::SourceSpan::new(self.current_token().line.into(), 1),
                format!(
                    "unexpected token in seq body: {:?}",
                    self.current_token().ty
                ),
            )),
        }
    }

    pub(super) fn parse_par_stmt(&mut self) -> Result<ParStmt, ParseError> {
        match &self.current_token().ty {
            TokenType::Ident(fn_name) => {
                let span = Span::new(self.current_token().line, self.current_token().col);
                let fn_name = fn_name.clone();
                self.advance();

                self.expect(TokenType::LParen)?;

                let project_name = match &self.current_token().ty {
                    TokenType::Ident(n) => n.clone(),
                    _ => {
                        return Err(ParseError::new(
                            miette::SourceSpan::new(self.current_token().line.into(), 1),
                            "expected identifier".to_string(),
                        ));
                    }
                };
                self.advance();

                self.expect(TokenType::RParen)?;
                self.expect(TokenType::Semicolon)?;

                Ok(ParStmt::FnCall {
                    span,
                    fn_name,
                    project_name,
                })
            }
            TokenType::Seq => {
                let span = Span::new(self.current_token().line, self.current_token().col);
                self.advance();
                self.expect(TokenType::Dot)?;

                let seq_name = match &self.current_token().ty {
                    TokenType::Ident(n) => n.clone(),
                    _ => {
                        return Err(ParseError::new(
                            miette::SourceSpan::new(self.current_token().line.into(), 1),
                            "expected identifier".to_string(),
                        ));
                    }
                };
                self.advance();

                self.expect(TokenType::Semicolon)?;

                Ok(ParStmt::SeqRef { span, seq_name })
            }
            _ => Err(ParseError::new(
                miette::SourceSpan::new(self.current_token().line.into(), 1),
                format!(
                    "unexpected token in par body: {:?}",
                    self.current_token().ty
                ),
            )),
        }
    }
}
