FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    curl \
    xz-utils \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Node 20
RUN curl -fsSL https://nodejs.org/dist/latest-v20.x/node-v20.19.4-linux-x64.tar.xz \
    -o node20.tar.xz && \
    mkdir -p /opt/node20 && \
    tar -xJf node20.tar.xz \
        --strip-components=1 \
        -C /opt/node20 && \
    rm node20.tar.xz

# Node 22
RUN curl -fsSL https://nodejs.org/dist/latest-v22.x/node-v22.18.0-linux-x64.tar.xz \
    -o node22.tar.xz && \
    mkdir -p /opt/node22 && \
    tar -xJf node22.tar.xz \
        --strip-components=1 \
        -C /opt/node22 && \
    rm node22.tar.xz

RUN ln -s /opt/node20/bin/node /usr/bin/node20 && \
    ln -s /opt/node22/bin/node /usr/bin/node22

WORKDIR /workspace

COPY --chmod=755 ajagent/ajagent /usr/bin/ajagent

ENTRYPOINT ["/usr/bin/ajagent"]