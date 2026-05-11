use super::*;

impl Parser {
    pub(crate) fn parse_fn_decl(&mut self) -> Result<Stmt, ParseError> {
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

        self.expect(TokenType::LBrace)?;

        let mut body = Vec::new();
        while self.current_token().ty != TokenType::RBrace {
            body.push(self.parse_fn_stmt()?);
        }

        self.expect(TokenType::RBrace)?;

        Ok(Stmt::FnDecl { name, body })
    }

    pub(crate) fn parse_seq_decl(&mut self) -> Result<Stmt, ParseError> {
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

        self.expect(TokenType::LBrace)?;

        let mut fns = Vec::new();
        while self.current_token().ty != TokenType::RBrace {
            fns.push(self.parse_block_fn_name()?);
        }

        self.expect(TokenType::RBrace)?;

        Ok(Stmt::SeqDecl { name, fns })
    }

    pub(crate) fn parse_par_decl(&mut self) -> Result<Stmt, ParseError> {
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

        self.expect(TokenType::LBrace)?;

        let mut fns = Vec::new();
        while self.current_token().ty != TokenType::RBrace {
            fns.push(self.parse_block_fn_name()?);
        }

        self.expect(TokenType::RBrace)?;

        Ok(Stmt::ParDecl { name, fns })
    }
}
