ARG ALPINE_VERSION=3.21

# Minimal runtimes for languages
FROM golang:1.24-alpine${ALPINE_VERSION} AS go_src
FROM node:22-alpine${ALPINE_VERSION} AS node_src
FROM python:3.12-alpine${ALPINE_VERSION} AS python_src
FROM eclipse-temurin:21-jdk-alpine AS java_src

# Base system
FROM alpine:${ALPINE_VERSION}

# C/C++ compiler + STL/stdlib (for competitive programming)
RUN apk add --no-cache gcc g++ musl-dev libstdc++

# Go 1.24 (Minimal toolchain for compilation/execution)
COPY --from=go_src /usr/local/go /usr/local/go
ENV GOROOT=/usr/local/go
ENV PATH=$GOROOT/bin:$PATH
ENV CGO_ENABLED=0

# Java 21 (Copy from the exact Temurin path)
COPY --from=java_src /opt/java/openjdk /opt/java/openjdk
ENV JAVA_HOME=/opt/java/openjdk
ENV PATH=$JAVA_HOME/bin:$PATH

# Node 22 (Raw interpreter binary)
COPY --from=node_src /usr/local /usr/local

# Python 3.12 (Interpreter, stdlib, & shared objects)
COPY --from=python_src /usr/local /usr/local
RUN ln -sf /usr/local/bin/python3 /usr/local/bin/python

# Copy agent executable
COPY --chmod=755 ../ajagent/cmd/ajagent /usr/bin/ajagent

# Create workspace directory structure
RUN mkdir -p /workspace && touch /workspace/execspec.json /workspace/agent.sock

WORKDIR /workspace

ENTRYPOINT [ "/usr/bin/ajagent" ]