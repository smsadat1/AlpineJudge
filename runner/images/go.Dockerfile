FROM alpine:3.22

RUN apk add --no-cache \
    wget \
    tar \
    ca-certificates

RUN wget https://go.dev/dl/go1.24.6.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.24.6.linux-amd64.tar.gz && \
    mv /usr/local/go /usr/local/go1.24 && \
    rm go1.24.6.linux-amd64.tar.gz

RUN wget https://go.dev/dl/go1.26.5.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.26.5.linux-amd64.tar.gz && \
    mv /usr/local/go /usr/local/go1.26 && \
    rm go1.26.5.linux-amd64.tar.gz

WORKDIR /workspace

COPY --chmod=755 ajagent/ajagent /usr/bin/ajagent

ENTRYPOINT ["/usr/bin/ajagent"]