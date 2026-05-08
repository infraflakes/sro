use super::super::*;
use crate::dsl::token::TokenType;

impl Parser {
    pub(crate) fn parse_seq_stmt(&mut self) -> Result<SeqStmt, ParseError> {
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
                            miette::SourceSpan::new(
                                self.current_token().offset.into(),
                                self.current_token().len,
                            ),
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
                            miette::SourceSpan::new(
                                self.current_token().offset.into(),
                                self.current_token().len,
                            ),
                            "expected identifier".to_string(),
                        ));
                    }
                };
                self.advance();

                self.expect(TokenType::Semicolon)?;

                Ok(SeqStmt::SeqRef { span, seq_name })
            }
            _ => Err(ParseError::new(
                miette::SourceSpan::new(
                    self.current_token().offset.into(),
                    self.current_token().len,
                ),
                format!(
                    "unexpected token in seq body: {:?}",
                    self.current_token().ty
                ),
            )),
        }
    }

    pub(crate) fn parse_par_stmt(&mut self) -> Result<ParStmt, ParseError> {
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
                            miette::SourceSpan::new(
                                self.current_token().offset.into(),
                                self.current_token().len,
                            ),
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
                            miette::SourceSpan::new(
                                self.current_token().offset.into(),
                                self.current_token().len,
                            ),
                            "expected identifier".to_string(),
                        ));
                    }
                };
                self.advance();

                self.expect(TokenType::Semicolon)?;

                Ok(ParStmt::SeqRef { span, seq_name })
            }
            _ => Err(ParseError::new(
                miette::SourceSpan::new(
                    self.current_token().offset.into(),
                    self.current_token().len,
                ),
                format!(
                    "unexpected token in par body: {:?}",
                    self.current_token().ty
                ),
            )),
        }
    }
}
