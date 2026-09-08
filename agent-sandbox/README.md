# agent-sandbox

Ejecuta un agente de código (**Codex**, **Claude Code**, **opencode** o **pi**) dentro de un
contenedor Docker, sobre un *worktree* de Git aislado. El agente trabaja en un branch propio,
en un directorio separado, sin tocar tu copia de trabajo actual.

## Contenido

| Archivo | Descripción |
| --- | --- |
| `Dockerfile` | Imagen `agent-sandbox` basada en `node:22-bookworm-slim` con `codex`, `claude`, `opencode` y `pi` instalados globalmente. |
| `agent-sandbox.sh` | Crea el worktree, levanta el contenedor y ejecuta el agente con tu prompt. |

## Requisitos

- Docker
- Git (ejecutar el script desde dentro de un repositorio)
- Configuración del agente **ya existente en el host**:
  - Codex → directorio `~/.codex`
  - Claude → directorio `~/.claude` **y** archivo `~/.claude.json`
  - opencode → directorios `~/.config/opencode`, `~/.local/share/opencode` y
    `~/.local/state/opencode`
  - pi → directorio `~/.pi/agent`

Si falta alguna de esas rutas el script se detiene con un error. No las crea: eso significa
que primero debes ejecutar `codex`, `claude`, `opencode` o `pi` en el host para autenticarte.

La autenticación **siempre** sale de esos directorios montados: el contenedor no recibe
ninguna variable de entorno con credenciales. Si tu agente se autentica por API key en el
host y no por login, primero tenés que loguearte para que quede en el config.

## Modo de uso

```
./agent-sandbox/agent-sandbox.sh <branch> --agent <codex|claude|opencode|pi> [--model <modelo>] [--push] <prompt...>
```

**El nombre del branch va siempre primero**, antes de cualquier opción.

### Parámetros

| Parámetro | Obligatorio | Descripción |
| --- | --- | --- |
| `<branch>` | Sí | Nombre del branch. El worktree se crea en `../<branch>` (directorio hermano del actual). |
| `--agent <codex\|claude\|opencode\|pi>` | Sí | Agente a ejecutar. |
| `--model <modelo>` | No | Modelo a usar. Por defecto: `gpt-5.6-terra` (codex) y `opus` (claude). **opencode y pi no tienen default**: si lo omitís, cada uno resuelve el modelo por su cuenta (pi lee `defaultModel` de `~/.pi/agent/settings.json`). |
| `--push` | No | Al terminar: `git add -A`, commit usando el prompt como mensaje y `git push --set-upstream`. Sin este flag los cambios quedan sin commitear en el worktree. |
| `<prompt...>` | Sí | Instrucción para el agente. Entre comillas |

### Modelos

**Claude** acepta un alias o el ID completo:

| Forma | Ejemplo | Comportamiento |
| --- | --- | --- |
| Alias | `opus`, `sonnet`, `haiku`, `fable` | Siempre el modelo más nuevo de ese nivel. |
| ID actual | `claude-sonnet-5`, `claude-opus-5` | Modelo fijo. **Sin sufijo de fecha** — agregarlo da error 404. |
| ID con fecha (modelos antiguos) | `claude-sonnet-4-5-20250929` | Snapshot congelado. |

**Codex** usa nombres de modelos de OpenAI: `gpt-5.6-terra`, `gpt-5.6-sol`, etc.

**opencode** usa el formato `proveedor/modelo`, con el proveedor tal como figura en tu
`~/.local/share/opencode/auth.json`:

| Ejemplo | Proveedor |
| --- | --- |
| `deepseek/deepseek-v4-pro` | deepseek |
| `anthropic/claude-sonnet-5` | anthropic |
| `openai/gpt-5.5` | openai |
| `openrouter/z-ai/glm-5.2` | openrouter |

Para ver los modelos disponibles: `opencode models` en el host.

**pi** también usa el formato `proveedor/modelo` (por ejemplo `deepseek/deepseek-v4-pro`). Si
omitís `--model`, pi resuelve el modelo con el `defaultModel` de `~/.pi/agent/settings.json`.


### Error branch worktree mismo nombre


| Situación | Error |
| --- | --- |
| El directorio `../<branch>` ya existe | `error: worktree path already exists: ...` (exit 1) |
| El branch ya está checkeado en otro worktree (o es el branch actual) | `fatal: '<branch>' is already used by worktree at ...` |

Solo funciona cuando:

- el branch **no existe** → se crea con `git worktree add -b`, o
- el branch **existe pero no está checkeado en ningún worktree** → se reutiliza.

En la práctica, si ya corriste el script con ese nombre tenés que limpiar el worktree y el
branch antes de volver a usarlo (ver "Limpiar un worktree"), o simplemente elegir otro nombre.

## Ejemplos

### Claude

Con el modelo por defecto (`opus`), dejando los cambios sin commitear:

```bash
./agent-sandbox/agent-sandbox.sh test-branch --agent claude "create a hello.txt file"
```

Con Sonnet:

```bash
./agent-sandbox/agent-sandbox.sh test-branch --agent claude --model sonnet "create a hello.txt file"
```

Fijando el modelo exacto y haciendo commit + push al terminar:

```bash
./agent-sandbox/agent-sandbox.sh test-branch --agent claude --model claude-sonnet-5 --push "create a hello.txt file"
```

### Codex

Con el modelo por defecto (`gpt-5.6-terra`), dejando los cambios sin commitear:

```bash
./agent-sandbox/agent-sandbox.sh test-branch --agent codex "create a hello.txt file"
```

Con un modelo específico:

```bash
./agent-sandbox/agent-sandbox.sh test-branch --agent codex --model gpt-5.6-sol "create a hello.txt file"
```

Con modelo específico y commit + push al terminar:

```bash
./agent-sandbox/agent-sandbox.sh test-branch --agent codex --model gpt-5.6-sol --push "create a hello.txt file"
```

Prompt más largo, con el branch y el push:

```bash
./agent-sandbox/agent-sandbox.sh add-query-log --agent codex --push "add a query-log view with filters by date and user"
```

### opencode

Con un modelo específico (formato `proveedor/modelo`):

```bash
./agent-sandbox/agent-sandbox.sh test-branch --agent opencode --model deepseek/deepseek-v4-pro "create a hello.txt file"
```

Sin `--model`, dejando que opencode resuelva el modelo:

```bash
./agent-sandbox/agent-sandbox.sh test-branch --agent opencode "create a hello.txt file"
```

Con commit + push al terminar:

```bash
./agent-sandbox/agent-sandbox.sh test-branch --agent opencode --model deepseek/deepseek-v4-pro --push "create a hello.txt file"
```

### pi

Con un modelo específico (formato `proveedor/modelo`):

```bash
./agent-sandbox/agent-sandbox.sh test-branch --agent pi --model deepseek/deepseek-v4-pro "create a hello.txt file"
```

Sin `--model`, usando el `defaultModel` de `~/.pi/agent/settings.json`:

```bash
./agent-sandbox/agent-sandbox.sh test-branch --agent pi "create a hello.txt file"
```

Con commit + push al terminar:

```bash
./agent-sandbox/agent-sandbox.sh test-branch --agent pi --model deepseek/deepseek-v4-pro --push "create a hello.txt file"
```


## Eliminar un worktree

```bash
git worktree remove --force ../test-branch && git branch -D test-branch
```

- `--force` descarta cambios sin commitear del worktree.
- `-D` borra el branch aunque no esté mergeado.
- Si borraste el directorio a mano: `git worktree prune`.
- Para ver los worktrees activos: `git worktree list`.

