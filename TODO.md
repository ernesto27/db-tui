# TODO

## Project-local Delve setup

Install a Delve version compatible with this project's Go toolchain without
changing the globally installed Delve version:

```sh
mkdir -p .tools/bin
GOBIN="$PWD/.tools/bin" go install github.com/go-delve/delve/cmd/dlv@v1.27.0
./.tools/bin/dlv version
```

Configure VS Code in `.vscode/settings.json` to use the project-local binary:

```json
{
  "go.alternateTools": {
    "dlv": "/absolute/path/to/db-tui/.tools/bin/dlv"
  }
}
```

Reload VS Code after changing the setting. `.tools` is a project convention,
not an official Go directory convention.



# Run Codex in Docker

The Codex CLI can run in Docker. A practical setup is:

```dockerfile
FROM golang:1.26-bookworm AS go

FROM node:22-bookworm-slim

COPY --from=go /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"

RUN apt-get update \
    && apt-get install -y --no-install-recommends git ripgrep ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN npm install --global @openai/codex

WORKDIR /workspace
ENTRYPOINT ["codex"]
```

Build it:

```bash
docker build -t codex-cli .
docker volume create codex-home
```

Authenticate from the container using device authentication:

```bash
docker run --rm -it \
  -v "$PWD:/workspace" \
  -v codex-home:/root/.codex \
  codex-cli login --device-auth
```

Then run Codex:

```bash
docker run --rm -it \
  -v "$PWD:/workspace" \
  -v codex-home:/root/.codex \
  codex-cli
```

For non-interactive automation:

```bash
docker run --rm \
  -v "$PWD:/workspace" \
  -e CODEX_API_KEY="$OPENAI_API_KEY" \
  codex-cli exec "Run the tests and explain any failures"
```

Notes:

- The named volume persists authentication and configuration.
- Only mount directories Codex should access.
- Avoid mounting the Docker socket, your entire home directory, or SSH credentials.
- If Codex's nested Linux sandbox causes container errors, run it with `--sandbox danger-full-access`. This disables Codex's internal sandbox, but Docker still limits it to the container and mounted paths.
- Files may be created as `root`; configure a matching container user if host ownership matters.

Official documentation: [authentication](https://learn.chatgpt.com/docs/auth), [environment variables](https://learn.chatgpt.com/docs/config-file/environment-variables), and [sandboxing](https://learn.chatgpt.com/docs/sandboxing).
