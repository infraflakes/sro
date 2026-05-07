use crate::token::{Token, TokenType, lookup_ident};

#[derive(Debug)]
pub struct Lexer {
    input: Vec<char>,
    pos: usize,
    read_pos: usize,
    ch: Option<char>,
    line: usize,
    col: usize,
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
        };
        lexer.read_char();
        lexer
    }

    fn read_char(&mut self) {
        self.ch = if self.read_pos < self.input.len() {
            Some(self.input[self.read_pos])
        } else {
            None
        };
        self.pos = self.read_pos;
        self.read_pos += 1;
        
        if self.ch == Some('\n') {
            self.line += 1;
            self.col = 0;
        } else {
            self.col += 1;
        }
    }

    fn peek(&self) -> Option<char> {
        if self.read_pos < self.input.len() {
            Some(self.input[self.read_pos])
        } else {
            None
        }
    }

    fn skip_whitespace(&mut self) {
        while let Some(c) = self.ch {
            if !c.is_whitespace() {
                break;
            }
            self.read_char();
        }
    }

    fn skip_comment(&mut self) {
        while self.ch != Some('\n') && self.ch.is_some() {
            self.read_char();
        }
    }

    fn read_ident(&mut self) -> Token {
        let start_line = self.line;
        let start_col = self.col;
        let start_pos = self.pos;
        
        while let Some(c) = self.ch {
            if c.is_alphanumeric() || c == '_' {
                self.read_char();
            } else {
                break;
            }
        }
        
        let ident: String = self.input[start_pos..self.pos].iter().collect();
        let ty = lookup_ident(&ident);
        Token::new(ty, start_line, start_col)
    }

    fn read_backtick(&mut self) -> Token {
        let start_line = self.line;
        let start_col = self.col;
        let start_pos = self.pos;
        
        self.read_char(); // skip opening backtick
        while let Some(c) = self.ch {
            if c == '`' {
                break;
            }
            self.read_char();
        }
        
        let content: String = self.input[start_pos + 1..self.pos].iter().collect();
        self.read_char(); // skip closing backtick
        
        Token::new(TokenType::Backtick(content), start_line, start_col)
    }

    fn read_path(&mut self) -> Token {
        let start_line = self.line;
        let start_col = self.col;
        let start_pos = self.pos;
        
        self.read_char(); // skip '.'
        self.read_char(); // skip '/'
        
        while let Some(c) = self.ch {
            if !c.is_whitespace() && c != ',' && c != ']' && c != ';' {
                self.read_char();
            } else {
                break;
            }
        }
        
        let path: String = self.input[start_pos..self.pos].iter().collect();
        Token::new(TokenType::PathLit(path), start_line, start_col)
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
        let ch = self.ch;

        match ch {
            None => Token::new(TokenType::EOF, start_line, start_col),
            Some('{') => {
                self.read_char();
                Token::new(TokenType::LBrace, start_line, start_col)
            }
            Some('}') => {
                self.read_char();
                Token::new(TokenType::RBrace, start_line, start_col)
            }
            Some('[') => {
                self.read_char();
                Token::new(TokenType::LBracket, start_line, start_col)
            }
            Some(']') => {
                self.read_char();
                Token::new(TokenType::RBracket, start_line, start_col)
            }
            Some('(') => {
                self.read_char();
                Token::new(TokenType::LParen, start_line, start_col)
            }
            Some(')') => {
                self.read_char();
                Token::new(TokenType::RParen, start_line, start_col)
            }
            Some(',') => {
                self.read_char();
                Token::new(TokenType::Comma, start_line, start_col)
            }
            Some('.') => {
                if self.peek() == Some('/') {
                    self.read_path()
                } else {
                    self.read_char();
                    Token::new(TokenType::Dot, start_line, start_col)
                }
            }
            Some(';') => {
                self.read_char();
                Token::new(TokenType::Semicolon, start_line, start_col)
            }
            Some('$') => {
                self.read_char();
                Token::new(TokenType::Dollar, start_line, start_col)
            }
            Some('=') => {
                self.read_char();
                Token::new(TokenType::Assign, start_line, start_col)
            }
            Some(':') => {
                self.read_char();
                Token::new(TokenType::Illegal("unexpected character: :".to_string()), start_line, start_col)
            }
            Some('`') => self.read_backtick(),
            Some(c) if c.is_alphabetic() || c == '_' => self.read_ident(),
            Some(c) => {
                self.read_char();
                Token::new(TokenType::Illegal(format!("unexpected character: {}", c)), start_line, start_col)
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
        assert_eq!(lexer.next_token().ty, TokenType::Backtick("bash".to_string()));
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
        
        assert_eq!(lexer.next_token().ty, TokenType::PathLit("./path/to/file.sro".to_string()));
    }
}
