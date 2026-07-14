FROM ubuntu:24.04
ENV DEBIAN_FRONTEND=noninteractive

# Install core prerequisites and add the Deadsnakes PPA
RUN apt-get update && apt-get install -y --no-install-recommends \
    software-properties-common \
    && add-apt-repository ppa:deadsnakes/ppa \
    && apt-get update

# Install specific Python versions and their development headers
RUN apt-get install -y --no-install-recommends \
    python3.10 python3.10-venv \
    python3.12 python3.12-venv \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

RUN python3.10 --version && python3.12 --version

WORKDIR /workspace

COPY --chmod=755 ajagent /usr/bin/ajagent

ENTRYPOINT [ "/usr/bin/ajagent" ]