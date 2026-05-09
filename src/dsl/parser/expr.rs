use super::*;
use crate::dsl::token::TokenType;

impl Parser {
    pub(super) fn parse_expr(&mut self) -> Result<Expr, ParseError> {
        match &self.current_token().ty {
            TokenType::Backtick(s) => {
                let span = Span::new(self.current_token().line, self.current_token().col);
                let err_span = miette::SourceSpan::new(
                    self.current_token().offset.into(),
                    self.current_token().len,
                );
                let parts = self.parse_template_parts(s, err_span)?;
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
                            miette::SourceSpan::new(
                                self.current_token().offset.into(),
                                self.current_token().len,
                            ),
                            "expected identifier after $".to_string(),
                        ));
                    }
                };
                self.advance();

                Ok(Expr::VarRef { span, name })
            }
            _ => Err(ParseError::new(
                miette::SourceSpan::new(
                    self.current_token().offset.into(),
                    self.current_token().len,
                ),
                "expected expression".to_string(),
            )),
        }
    }

    pub(super) fn parse_template_parts(
        &self,
        s: &str,
        span: miette::SourceSpan,
    ) -> Result<Vec<TemplatePart>, ParseError> {
        let mut parts = Vec::new();
        let mut chars = s.chars().peekable();

        while let Some(c) = chars.next() {
            if c == '$' && chars.peek() == Some(&'{') {
                chars.next(); // skip '{'

                let mut var_name = String::new();
                let mut found_close = false;
                while let Some(&c) = chars.peek() {
                    if c == '}' {
                        chars.next();
                        found_close = true;
                        break;
                    }
                    var_name.push(chars.next().unwrap());
                }

                if !found_close {
                    return Err(ParseError::new(
                        span,
                        "unclosed variable interpolation".to_string(),
                    ));
                }

                if var_name.is_empty() {
                    return Err(ParseError::new(
                        span,
                        "empty variable name in template".to_string(),
                    ));
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

        Ok(parts)
    }
}
