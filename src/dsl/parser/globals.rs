use super::*;

impl Parser {
    pub(crate) fn parse_shell_decl(&mut self) -> Result<Stmt, ParseError> {
        self.advance();

        self.expect(TokenType::Assign)?;

        let value = self.parse_simple_backtick()?;
        self.expect(TokenType::Semicolon)?;

        Ok(Stmt::ShellDecl { value })
    }

    pub(crate) fn parse_sanctuary_decl(&mut self) -> Result<Stmt, ParseError> {
        self.advance();

        self.expect(TokenType::Assign)?;

        let value = self.parse_expr()?;
        self.expect(TokenType::Semicolon)?;

        Ok(Stmt::SanctuaryDecl { value })
    }

    pub(crate) fn parse_import_decl(&mut self) -> Result<Stmt, ParseError> {
        self.advance();

        let path = match &self.current_token().ty {
            TokenType::PathLit(p) => p.clone(),
            _ => {
                return Err(ParseError::new(
                    miette::SourceSpan::new(
                        self.current_token().offset.into(),
                        self.current_token().len,
                    ),
                    "expected import path".to_string(),
                ));
            }
        };
        self.advance();

        self.expect(TokenType::Semicolon)?;

        Ok(Stmt::ImportDecl { paths: vec![path] })
    }

    pub(crate) fn parse_var_decl(&mut self) -> Result<Stmt, ParseError> {
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

        Ok(Stmt::VarDecl {
            var_type,
            name,
            value,
        })
    }
}
