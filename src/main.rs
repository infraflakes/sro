mod ast;
mod cli;
mod config;
mod lexer;
mod parser;
mod runner;
mod sync;
mod tui;
mod token;

fn main() {
    cli::run();
}
