use super::*;

impl Parser {
    pub(crate) fn parse_block_fn_name(&mut self) -> Result<String, ParseError> {
        let fn_name = match &self.current_token().ty {
            TokenType::Ident(n) => n.clone(),
            _ => {
                return Err(ParseError::new(
                    miette::SourceSpan::new(
                        self.current_token().offset.into(),
                        self.current_token().len,
                    ),
                    format!(
                        "unexpected token in block body: {:?}",
                        self.current_token().ty
                    ),
                ));
            }
        };
        self.advance();
        self.expect(TokenType::Semicolon)?;
        Ok(fn_name)
    }
}
