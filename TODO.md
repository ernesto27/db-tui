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
