FROM gcc:13-bookworm

# copy agent executable
COPY --chmod=755 ../ajagent/cmd/ajagent /usr/bin/ajagent

# create the workspace directory inside the base image
RUN mkdir -p /workspace && touch /workspace/main.cpp /workspace/execspec.json /workspace/agent.sock

ENTRYPOINT [ "/usr/bin/ajagent" ]