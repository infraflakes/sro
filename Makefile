.PHONY: build fmt test clean

init:
	dagger develop
	cargo check

build:
	cargo build --release --target x86_64-unknown-linux-musl

test:
	cargo clippy
	cargo fmt --check

fmt:
	cargo clippy --fix --allow-dirty
	cargo fmt

clean:
	rm -rf target
