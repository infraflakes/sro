.PHONY: build fmt test clean
DOCKER_BIN := $(shell command -v docker 2> /dev/null || command -v podman 2> /dev/null)

init:
	cargo check

build: clean
ifeq ($(DOCKER_BIN),)
	$(error "Neither docker nor podman found in PATH! (＃￣ω￣)")
endif
	$(DOCKER_BIN) build --output type=local,dest=bin .

test:
	cargo test
	cargo clippy
	cargo fmt --check

fmt:
	cargo clippy --fix --allow-dirty
	cargo fmt

clean:
	rm -rf bin
	rm -rf result
