use super::*;
use crate::dsl::token::TokenType;

impl Parser {
    pub(super) fn parse_expr(&mut self) -> Result<Expr, ParseError> {
        match &self.current_token().ty {
            TokenType::Backtick(s) => {
                let span = Span::new(self.current_token().line, self.current_token().col);
                let parts = self.parse_template_parts(s);
                self.advance();
                Ok(Expr::BacktickLit { span, parts })
            }
            TokenType::Dollar => {
                let span = Span::new(self.current_token().line, self.current_token().col);
                self.advance();

                let name = match &self.current_token().ty {
                    TokenType::Ident(n) => n.clone(),
                    _ => {
                        return Err(ParseError::new(
                            miette::SourceSpan::new(self.current_token().line.into(), 1),
                            "expected identifier after $".to_string(),
                        ));
                    }
                };
                self.advance();

                Ok(Expr::VarRef { span, name })
            }
            _ => Err(ParseError::new(
                miette::SourceSpan::new(self.current_token().line.into(), 1),
                "expected expression".to_string(),
            )),
        }
    }

    pub(super) fn parse_template_parts(&self, s: &str) -> Vec<TemplatePart> {
        let mut parts = Vec::new();
        let mut chars = s.chars().peekable();

        while let Some(c) = chars.next() {
            if c == '$' && chars.peek() == Some(&'{') {
                chars.next(); // skip '{'
                let mut var_name = String::new();

                while let Some(&c) = chars.peek() {
                    if c == '}' {
                        chars.next();
                        break;
                    }
                    var_name.push(chars.next().unwrap());
                }

                parts.push(TemplatePart {
                    is_var: true,
                    value: var_name,
                });
            } else {
                parts.push(TemplatePart {
                    is_var: false,
                    value: c.to_string(),
                });
            }
        }

        parts
    }
}
