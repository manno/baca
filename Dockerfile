FROM catthehacker/ubuntu:act-latest

LABEL maintainer="mm/hackweek"

# needs npm, gh, fleet-cli, linuxbrew
RUN echo "Installing custom CLIs: Gemini CLI and GitHub Copilot CLI..." && \
    npm install -g @google/gemini-cli && \
    npm install -g @github/copilot && \
    echo "Cleaning up npm cache..." && \
    npm cache clean --force
