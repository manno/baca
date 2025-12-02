FROM catthehacker/ubuntu:act-latest
LABEL maintainer="mm/hackweek"

# needs npm, gh, fleet-cli, linuxbrew
RUN echo "Installing custom CLIs: Gemini CLI and GitHub Copilot CLI..." && \
    npm install -g @google/gemini-cli && \
    npm install -g @github/copilot && \
    echo "Cleaning up npm cache..." && \
    npm cache clean --force

# Install Fleet CLI
RUN echo "Installing Fleet CLI..." && \
    curl -L https://github.com/rancher/fleet/releases/download/v0.14.0/fleet-linux-${TARGETARCH} -o /usr/local/bin/fleet && \
    chmod +x /usr/local/bin/fleet

# Copy pre-built bca binary (must be built for correct architecture before docker build)
# For local builds, copy bca-release which should match TARGETARCH
# For multi-arch builds with buildx, use dist/bca-linux-${TARGETARCH}
ONBUILD ARG TARGETARCH
ONBUILD COPY dist/bca-linux-$TARGETARCH /usr/local/bin/bca

# Set working directory for job execution
WORKDIR /workspace

# Set bca as the default command (can be overridden)
CMD ["/usr/local/bin/bca"]
