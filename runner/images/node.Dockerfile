FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://nodejs.org/dist/v20.6.1/node-v20.6.1-linux-x64.tar.gz -o /tmp/node20.tar.gz && \
    mkdir -p /opt/node20 && \
    tar -xzf /tmp/node20.tar.gz \
        --strip-components=1 \
        -C /opt/node20 && \
    rm /tmp/node20.tar.gz && \
    curl -fsSL https://nodejs.org/dist/v22.9.0/node-v22.9.0-linux-x64.tar.gz -o /tmp/node22.tar.gz && \
    mkdir -p /opt/node22 && \
    tar -xzf /tmp/node22.tar.gz \
        --strip-components=1 \
        -C /opt/node22 && \
    rm /tmp/node22.tar.gz

RUN ln -s /opt/node20/bin/node /usr/bin/node20 && \
    ln -s /opt/node22/bin/node /usr/bin/node22

WORKDIR /workspace

COPY --chmod=755 ../ajagent/ajagent /usr/bin/ajagent

ENTRYPOINT ["/usr/bin/ajagent"]