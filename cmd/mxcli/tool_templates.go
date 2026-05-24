// SPDX-License-Identifier: Apache-2.0

// tool_templates.go - Configuration templates for Claude Code integration
package main

import (
	"fmt"
)

func generateClaudeSettings(projectName, mprPath string) string {
	return settingsJSON
}

func generateDevcontainerJSON(projectName, mprPath, containerRuntime string) string {
	feature := `"ghcr.io/devcontainers/features/docker-in-docker:2": {}`
	containerEnv := `"PLAYWRIGHT_CLI_SESSION": "mendix-app"`
	if containerRuntime == "podman" {
		feature = `"ghcr.io/devcontainers/features/podman-in-podman:1": {}`
		containerEnv = `"PLAYWRIGHT_CLI_SESSION": "mendix-app",
    "MXCLI_CONTAINER_CLI": "podman"`
	}

	return fmt.Sprintf(`{
  "name": "%s",
  "build": {
    "dockerfile": "Dockerfile"
  },
  "features": {
    %s
  },
  "forwardPorts": [8080, 8090, 5432],
  "portsAttributes": {
    "8080-8099": { "onAutoForward": "silent" },
    "5432-5499": { "onAutoForward": "silent" }
  },
  "containerEnv": {
    %s
  },
  "postCreateCommand": "curl -fsSL https://claude.ai/install.sh | bash && if [ -f ./mxcli ] && file ./mxcli | grep -q Linux; then echo 'mxcli binary OK'; else ./mxcli setup mxcli --output ./mxcli 2>/dev/null || { ARCH=$(uname -m); [ \"$ARCH\" = x86_64 ] && ARCH=amd64; [ \"$ARCH\" = aarch64 ] && ARCH=arm64; curl -fsSL https://github.com/mendixlabs/mxcli/releases/latest/download/mxcli-linux-${ARCH} -o ./mxcli && chmod +x ./mxcli; }; fi",
  "customizations": {
    "vscode": {
      "extensions": [
        "anthropic.claude-code"
      ],
      "settings": {
        "mdl.mxcliPath": "./mxcli"
      }
    }
  },
  "remoteUser": "vscode"
}
`, projectName, feature, containerEnv)
}

func generateDockerfile(projectName, mprPath string) string {
	return `FROM mcr.microsoft.com/devcontainers/base:bookworm

# Install Adoptium JDK 21 (required by MxBuild), Node.js 22, and utility tools
RUN apt-get update && apt-get install -y --no-install-recommends wget apt-transport-https gpg ca-certificates curl && \
    wget -qO - https://packages.adoptium.net/artifactory/api/gpg/key/public | gpg --dearmor -o /etc/apt/keyrings/adoptium.gpg && \
    echo "deb [signed-by=/etc/apt/keyrings/adoptium.gpg] https://packages.adoptium.net/artifactory/deb bookworm main" > /etc/apt/sources.list.d/adoptium.list && \
    curl -fsSL https://deb.nodesource.com/setup_22.x | bash - && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
       temurin-21-jdk \
       nodejs \
       postgresql-client \
       kafkacat \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install playwright-cli and Chromium with all system dependencies (must run as root)
RUN npm install -g @playwright/cli@latest && \
    npx playwright install --with-deps chromium
`
}

func generatePlaywrightConfig() string {
	return `{
  "browser": {
    "browserName": "chromium",
    "isolated": true,
    "launchOptions": {
      "headless": true
    }
  },
  "timeouts": {
    "action": 10000,
    "navigation": 30000
  },
  "network": {
    "allowedOrigins": [
      "http://localhost:8079",
      "http://localhost:8080",
      "http://localhost:8081",
      "http://localhost:8082",
      "http://localhost:8083",
      "http://localhost:8084",
      "http://localhost:8085"
    ]
  }
}
`
}
