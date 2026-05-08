use super::super::*;
use crate::dsl::token::TokenType;

impl Parser {
    pub(crate) fn parse_fn_decl(&mut self) -> Result<Stmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'fn'

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

        self.expect(TokenType::LBrace)?;

        let mut body = Vec::new();
        while self.current_token().ty != TokenType::RBrace {
            body.push(self.parse_fn_stmt()?);
        }

        self.expect(TokenType::RBrace)?;

        Ok(Stmt::FnDecl { span, name, body })
    }

    pub(crate) fn parse_seq_decl(&mut self) -> Result<Stmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'seq'

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

        self.expect(TokenType::LBrace)?;

        let mut stmts = Vec::new();
        while self.current_token().ty != TokenType::RBrace {
            stmts.push(self.parse_seq_stmt()?);
        }

        self.expect(TokenType::RBrace)?;

        Ok(Stmt::SeqDecl { span, name, stmts })
    }

    pub(crate) fn parse_par_decl(&mut self) -> Result<Stmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'par'

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

        self.expect(TokenType::LBrace)?;

        let mut stmts = Vec::new();
        while self.current_token().ty != TokenType::RBrace {
            stmts.push(self.parse_par_stmt()?);
        }

        self.expect(TokenType::RBrace)?;

        Ok(Stmt::ParDecl { span, name, stmts })
    }
}
