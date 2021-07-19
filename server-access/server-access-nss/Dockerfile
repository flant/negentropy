FROM debian/eol:wheezy
# official debian:7 docker is broken, it cant apt-get update because "end of life"

ARG RUST_VERSION=1.53.0

# Installing latest rust version into debian wheezy - we need old libc

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates gcc libc6-dev curl build-essential libsqlite3-dev
RUN curl --proto '=https' -sSf https://sh.rustup.rs > /tmp/rustup
RUN sh /tmp/rustup -y --no-modify-path --profile minimal --default-toolchain $RUST_VERSION

