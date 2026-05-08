use crate::dsl::token::{Token, TokenType};

mod reader;

#[derive(Debug)]
pub struct Lexer {
    pub(super) input: Vec<char>,
    pub(super) pos: usize,
    pub(super) read_pos: usize,
    pub(super) ch: Option<char>,
    pub(super) line: usize,
    pub(super) col: usize,
    pub(super) byte_offset: usize,
}

impl Lexer {
    pub fn new(input: String) -> Self {
        let mut lexer = Self {
            input: input.chars().collect(),
            pos: 0,
            read_pos: 0,
            ch: None,
            line: 1,
            col: 0,
            byte_offset: 0,
        };
        lexer.read_char();
        lexer
    }

    pub fn next_token(&mut self) -> Token {
        loop {
            self.skip_whitespace();
            if self.ch != Some('#') {
                break;
            }
            self.skip_comment();
        }

        let start_line = self.line;
        let start_col = self.col;
        let start_byte_offset = self.byte_offset;
        let ch = self.ch;

        match ch {
            None => Token::new(TokenType::EOF, start_line, start_col, start_byte_offset, 0),
            Some('{') => {
                self.read_char();
                Token::new(
                    TokenType::LBrace,
                    start_line,
                    start_col,
                    start_byte_offset,
                    self.byte_offset - start_byte_offset,
                )
            }
            Some('}') => {
                self.read_char();
                Token::new(
                    TokenType::RBrace,
                    start_line,
                    start_col,
                    start_byte_offset,
                    self.byte_offset - start_byte_offset,
                )
            }
            Some('[') => {
                self.read_char();
                Token::new(
                    TokenType::LBracket,
                    start_line,
                    start_col,
                    start_byte_offset,
                    self.byte_offset - start_byte_offset,
                )
            }
            Some(']') => {
                self.read_char();
                Token::new(
                    TokenType::RBracket,
                    start_line,
                    start_col,
                    start_byte_offset,
                    self.byte_offset - start_byte_offset,
                )
            }
            Some('(') => {
                self.read_char();
                Token::new(
                    TokenType::LParen,
                    start_line,
                    start_col,
                    start_byte_offset,
                    self.byte_offset - start_byte_offset,
                )
            }
            Some(')') => {
                self.read_char();
                Token::new(
                    TokenType::RParen,
                    start_line,
                    start_col,
                    start_byte_offset,
                    self.byte_offset - start_byte_offset,
                )
            }
            Some(',') => {
                self.read_char();
                Token::new(
                    TokenType::Comma,
                    start_line,
                    start_col,
                    start_byte_offset,
                    self.byte_offset - start_byte_offset,
                )
            }
            Some('.') => {
                if self.peek() == Some('/') {
                    self.read_path()
                } else {
                    self.read_char();
                    Token::new(
                        TokenType::Dot,
                        start_line,
                        start_col,
                        start_byte_offset,
                        self.byte_offset - start_byte_offset,
                    )
                }
            }
            Some(';') => {
                self.read_char();
                Token::new(
                    TokenType::Semicolon,
                    start_line,
                    start_col,
                    start_byte_offset,
                    self.byte_offset - start_byte_offset,
                )
            }
            Some('$') => {
                self.read_char();
                Token::new(
                    TokenType::Dollar,
                    start_line,
                    start_col,
                    start_byte_offset,
                    self.byte_offset - start_byte_offset,
                )
            }
            Some('=') => {
                self.read_char();
                Token::new(
                    TokenType::Assign,
                    start_line,
                    start_col,
                    start_byte_offset,
                    self.byte_offset - start_byte_offset,
                )
            }
            Some(':') => {
                self.read_char();
                Token::new(
                    TokenType::Illegal("unexpected character: :".to_string()),
                    start_line,
                    start_col,
                    start_byte_offset,
                    self.byte_offset - start_byte_offset,
                )
            }
            Some('`') => self.read_backtick(),
            Some(c) if c.is_alphabetic() || c == '_' => self.read_ident(),
            Some(c) => {
                self.read_char();
                Token::new(
                    TokenType::Illegal(format!("unexpected character: {}", c)),
                    start_line,
                    start_col,
                    start_byte_offset,
                    self.byte_offset - start_byte_offset,
                )
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_basic_tokens() {
        let input = "shell = `bash`;";
        let mut lexer = Lexer::new(input.to_string());

        assert_eq!(lexer.next_token().ty, TokenType::Shell);
        assert_eq!(lexer.next_token().ty, TokenType::Assign);
        assert_eq!(
            lexer.next_token().ty,
            TokenType::Backtick("bash".to_string())
        );
        assert_eq!(lexer.next_token().ty, TokenType::Semicolon);
        assert_eq!(lexer.next_token().ty, TokenType::EOF);
    }

    #[test]
    fn test_keywords() {
        let input = "sanctuary import var string pr fn seq par env log exec cd shell";
        let mut lexer = Lexer::new(input.to_string());

        assert_eq!(lexer.next_token().ty, TokenType::Sanctuary);
        assert_eq!(lexer.next_token().ty, TokenType::Import);
        assert_eq!(lexer.next_token().ty, TokenType::Var);
        assert_eq!(lexer.next_token().ty, TokenType::StringKw);
        assert_eq!(lexer.next_token().ty, TokenType::Pr);
        assert_eq!(lexer.next_token().ty, TokenType::Fn);
        assert_eq!(lexer.next_token().ty, TokenType::Seq);
        assert_eq!(lexer.next_token().ty, TokenType::Par);
        assert_eq!(lexer.next_token().ty, TokenType::Env);
        assert_eq!(lexer.next_token().ty, TokenType::Log);
        assert_eq!(lexer.next_token().ty, TokenType::Exec);
        assert_eq!(lexer.next_token().ty, TokenType::Cd);
        assert_eq!(lexer.next_token().ty, TokenType::Shell);
    }

    #[test]
    fn test_comments() {
        let input = "# comment\nshell = `bash`;";
        let mut lexer = Lexer::new(input.to_string());

        assert_eq!(lexer.next_token().ty, TokenType::Shell);
    }

    #[test]
    fn test_path_literal() {
        let input = "./path/to/file.sro";
        let mut lexer = Lexer::new(input.to_string());

        assert_eq!(
            lexer.next_token().ty,
            TokenType::PathLit("./path/to/file.sro".to_string())
        );
    }
}
