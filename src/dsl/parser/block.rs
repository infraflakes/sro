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
                        "expected function name, found {}",
                        format_token(self.current_token())
                    ),
                ));
            }
        };
        self.advance();
        self.expect(TokenType::Semicolon)?;
        Ok(fn_name)
    }
}
