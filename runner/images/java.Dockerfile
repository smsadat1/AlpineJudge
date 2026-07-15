FROM eclipse-temurin:17-jdk-noble AS jdk17
FROM eclipse-temurin:21-jdk-noble AS jdk21

FROM ubuntu:noble

# Prevent interactive prompts during installation
ENV DEBIAN_FRONTEND=noninteractive

# Install core utilities
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create target directories for the Java installations
RUN mkdir -p /usr/lib/jvm/java-17 /usr/lib/jvm/java-21

COPY --from=jdk17 /opt/java/openjdk /usr/lib/jvm/java-17
COPY --from=jdk21 /opt/java/openjdk /usr/lib/jvm/java-21

WORKDIR /workspace

COPY --chmod=755 ajagent/ajagent /usr/bin/ajagent

ENTRYPOINT [ "/usr/bin/ajagent" ]