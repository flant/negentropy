.PHONY: build install test

# Makefile directory
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

EXECUTABLES = docker
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH")))

all	: build install

build:
	docker build -f ${ROOT_DIR}/Dockerfile -t rust:debian-jessie ${ROOT_DIR}
	mkdir -p /tmp/flantauthbuild
	docker run --rm -v ${ROOT_DIR}:/app -v /tmp/flantauthbuild:/root/.cargo/registry -w /app rust:debian-jessie /bin/bash -c \
		"/root/.cargo/bin/cargo build --lib --release --features dynamic_paths && \
		strip -s target/release/libnss_flantauth.so"
	cp ${ROOT_DIR}/target/release/libnss_flantauth.so ${ROOT_DIR}/lib/libnss_flantauth.so.2

install: build
	cp ${ROOT_DIR}/lib/libnss_flantauth.so.2 /lib/x86_64-linux-gnu/

test:
	./example/run.sh