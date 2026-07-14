FROM gcc:14.4.0-trixie

WORKDIR /workspace

COPY --chmod=755 ajagent /usr/bin/ajagent

ENTRYPOINT [ "/usr/bin/ajagent" ]