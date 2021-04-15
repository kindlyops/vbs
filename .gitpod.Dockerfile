FROM gitpod/workspace-full

# Install custom tools, runtimes, etc.
# For example "bastet", a command-line tetris clone:
# RUN brew install bastet
#
# More information: https://www.gitpod.io/docs/config-docker/

# Create bin folder under $HOME for random binaries
RUN mkdir $HOME/bin

# Install bazelisk so that it is available on the command line
RUN npm install -g @bazel/bazelisk
# Install bazel-watcher
RUN npm install -g @bazel/ibazel

# Install buildifier and buildozer
ENV GOPATH=$HOME/go-packages
RUN go get -u -v github.com/bazelbuild/buildtools/buildifier
RUN go get -u -v github.com/bazelbuild/buildtools/buildozer
ENV GOPATH=/workspace/go

# Install starlark LSP
RUN wget -O $HOME/bin/gostarlark https://github.com/stackb/bzl/releases/download/0.9.4/bzl && \
  chmod +x $HOME/bin/gostarlark
  
# Add Bazel command line completion
USER root
COPY scripts/bazel-complete.bash /etc/bash_completion.d/bazel-complete.bash
USER gitpod