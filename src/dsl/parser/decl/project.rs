use super::super::*;
use crate::dsl::token::TokenType;

impl Parser {
    pub(crate) fn parse_project_decl(&mut self) -> Result<Stmt, ParseError> {
        let span = Span::new(self.current_token().line, self.current_token().col);
        self.advance(); // skip 'pr'

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

        let mut fields = Vec::new();

        while self.current_token().ty != TokenType::RBrace {
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
            self.expect(TokenType::Semicolon)?;

            fields.push(ProjectField { key, value });
        }

        self.expect(TokenType::RBrace)?;

        Ok(Stmt::ProjectDecl {
            span,
            name,
            fields,
        })
    }
}
