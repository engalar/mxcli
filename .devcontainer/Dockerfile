# Mendix Model SDK Go - Development Container
FROM mcr.microsoft.com/devcontainers/go:dev-1.26-bookworm

RUN rm -f /etc/apt/sources.list.d/yarn.list

# Install Adoptium JDK 21 (required by MxBuild) and JDK 17 (required by ANTLR4)
RUN apt-get update && apt-get install -y --no-install-recommends \
    wget apt-transport-https gpg \
    python3-pip \
    sqlite3 \
    postgresql-client \
    kafkacat \
    tmux \
    && wget -qO - https://packages.adoptium.net/artifactory/api/gpg/key/public | gpg --dearmor -o /etc/apt/keyrings/adoptium.gpg \
    && echo "deb [signed-by=/etc/apt/keyrings/adoptium.gpg] https://packages.adoptium.net/artifactory/deb bookworm main" > /etc/apt/sources.list.d/adoptium.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
       openjdk-17-jre-headless \
       temurin-21-jdk \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install ANTLR4 tools
RUN pip3 install --break-system-packages antlr4-tools

# Install bun (required for VS Code extension builds)
# Install to /usr/local so all users (including vscode) can execute it
USER vscode
RUN BUN_INSTALL=/usr/local curl -fsSL https://bun.sh/install | bash

# Configure tmux for vscode user
RUN cat <<'EOF' > /home/vscode/.tmux.conf
set -g mouse on

# Remap prefix from 'C-b' to 'C-a'
unbind C-b
set-option -g prefix C-a
bind-key C-a send-prefix

# Split panes using | and -
bind | split-window -h
bind - split-window -v
unbind '"'
unbind %

# Switch panes using Alt-arrow without prefix
bind -n M-Left select-pane -L
bind -n M-Right select-pane -R
bind -n M-Up select-pane -U
bind -n M-Down select-pane -D

# Reload config
bind r source-file ~/.tmux.conf \; display "Config reloaded!"

# True color support
set -g default-terminal "tmux-256color"
set -ag terminal-overrides ",xterm-256color:RGB"

# Unicode / box-drawing passthrough (fixes Claude Code rendering in tmux)
set -as terminal-features ",*:UTF-8"
set -gq allow-passthrough on
set -g extended-keys on

# Status bar
set -g status-interval 5
set -g status-right-length 60
set -g status-right '#[fg=colour250]#{b:pane_current_path} #[fg=colour214]#(cd #{pane_current_path} && git symbolic-ref --short HEAD 2>/dev/null || echo "no-branch")'

# Colors
set -g status-left-style  "fg=colour255 bg=colour240"
set -g status-right-style "fg=colour215 bg=colour236"
set -g status-style       "fg=colour250 bg=colour236"
set -g message-style      "fg=colour232 bg=colour214"
set -g window-status-current-style  "fg=colour232 bg=colour214"
set -g window-status-activity-style "fg=colour214 bg=colour238"
set -g mode-style         "fg=colour232 bg=colour214"
EOF
RUN chown vscode:vscode /home/vscode/.tmux.conf

# Alias tmux to force UTF-8 mode (fixes Claude Code logo rendering)
RUN echo '\n# Force UTF-8 mode for tmux\nalias tmux="tmux -u"' >> /home/vscode/.bashrc

# Install mdbook (static site generator for documentation)
USER root
RUN MDBOOK_VERSION=v0.5.2 \
    && ARCH="$(uname -m)" \
    && if [ "$ARCH" = "aarch64" ]; then ARCH="aarch64-unknown-linux-musl"; \
       else ARCH="x86_64-unknown-linux-gnu"; fi \
    && wget -qO- "https://github.com/rust-lang/mdBook/releases/download/${MDBOOK_VERSION}/mdbook-${MDBOOK_VERSION}-${ARCH}.tar.gz" \
    | tar -xz -C /usr/local/bin

# Create libs directory for reflection data
RUN mkdir -p /workspaces/ModelSDKGo/libs
