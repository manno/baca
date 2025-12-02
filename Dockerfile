FROM catthehacker/ubuntu:act-latest
LABEL maintainer="mm/hackweek"

# Upgrade Node.js from v18 to v20
# The base image has Node in /opt/acttoolcache, we'll install v20 via NodeSource and update PATH
RUN curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | \
    gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg && \
    echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_20.x nodistro main" | \
    tee /etc/apt/sources.list.d/nodesource.list && \
    apt-get update && \
    apt-get install -y nodejs && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* && \
    rm -rf /opt/acttoolcache/node/18.20.8

# needs npm, gh, fleet-cli, linuxbrew
RUN echo "Installing custom CLIs: Gemini CLI and GitHub Copilot CLI..." && \
    npm install -g @google/gemini-cli && \
    npm install -g @github/copilot && \
    echo "Cleaning up npm cache..." && \
    npm cache clean --force

ARG TARGETARCH

# Install Fleet CLI
RUN echo "Installing Fleet CLI for ${TARGETARCH}..." && \
    curl -L https://github.com/rancher/fleet/releases/download/v0.14.0/fleet-linux-${TARGETARCH} -o /usr/local/bin/fleet && \
    chmod +x /usr/local/bin/fleet && \
    echo "Fleet CLI installed successfully" && \
    /usr/local/bin/fleet --version || echo "Fleet version check failed"

# Copy pre-built bca binary (must be built for correct architecture before docker build)
COPY dist/bca-linux-$TARGETARCH /usr/local/bin/bca

# Set working directory for job execution
WORKDIR /workspace

# Set bca as the default command (can be overridden)
CMD ["/usr/local/bin/bca"]
