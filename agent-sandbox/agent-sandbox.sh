#!/usr/bin/env bash
# Crea un worktree git y ejecuta un agente sobre él en el contenedor agent-sandbox.
# --agent es obligatorio: codex, claude, opencode o pi. --model tiene un valor por
# defecto según el agente (codex: gpt-5.6-terra, claude: opus). Sin --push no se pushean los cambios
#
# Ejemplos:
#   ./agent-sandbox/agent-sandbox.sh test-branch --agent codex --model gpt-5.6-sol --push "create a hello.txt file"
#   ./agent-sandbox/agent-sandbox.sh test-branch --agent claude --model opus --push "create a hello.txt file"
#   ./agent-sandbox/agent-sandbox.sh test-branch --agent opencode --model deepseek/deepseek-v4-pro --push "create a hello.txt file"
#   ./agent-sandbox/agent-sandbox.sh test-branch --agent pi --model deepseek/deepseek-v4-pro --push "create a hello.txt file"
#
# Eliminar worktree y branch: git worktree remove --force ../test-branch && git branch -D test-branch

set -euo pipefail

image_name=agent-sandbox
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)

declare -A agent_binaries=([codex]=codex [claude]=claude [opencode]=opencode [pi]=pi)
declare -A agent_packages=([codex]="@openai/codex" [claude]="@anthropic-ai/claude-code" [opencode]="opencode-ai" [pi]="@earendil-works/pi-coding-agent")
declare -A agent_build_args=([codex]=CODEX_VERSION [claude]=CLAUDE_CODE_VERSION [opencode]=OPENCODE_VERSION [pi]=PI_VERSION)
declare -A agent_default_models=([codex]="gpt-5.6-terra" [claude]="opus")

usage() {
  printf 'Usage: %s <branch-name> --agent <codex|claude|opencode|pi> [--model <model>] [--push] <prompt...>\n' "${0##*/}" >&2
  exit 2
}

require_agent() {
  [[ -n $1 ]] || {
    printf 'error: --agent is required (codex, claude, opencode or pi)\n' >&2
    usage
  }
}

validate_agent() {
  [[ -n ${agent_binaries[$1]:-} ]] || {
    printf 'error: unknown agent: %s (expected codex, claude, opencode or pi)\n' "$1" >&2
    exit 2
  }
}

build_image() {
  local agent=${1:-} version=${2:-}
  local build_args=()

  if [[ -n $agent && -n $version ]]; then
    build_args+=(--build-arg "${agent_build_args[$agent]}=$version")
  fi

  docker build \
    ${build_args[@]+"${build_args[@]}"} \
    --tag "$image_name" \
    --file "$script_dir/Dockerfile" \
    "$script_dir"
}

installed_version() {
  docker run --rm --entrypoint "${agent_binaries[$1]}" "$image_name" --version \
    | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -n 1
}

latest_version() {
  docker run --rm --entrypoint npm "$image_name" view "${agent_packages[$1]}" version
}

ensure_image_latest() {
  local agent=$1
  local installed latest rebuilt

  if ! docker image inspect "$image_name" >/dev/null 2>&1; then
    printf 'Image %s not found. Building it.\n' "$image_name"
    build_image
  fi

  installed=$(installed_version "$agent")
  latest=$(latest_version "$agent")

  printf '%s in image: %s\n' "$agent" "$installed"
  printf '%s latest:   %s\n' "$agent" "$latest"

  if [[ $installed == "$latest" ]]; then
    printf '%s is up to date.\n' "$agent"
    return
  fi

  printf 'Update available. Rebuilding %s with %s %s.\n' "$image_name" "$agent" "$latest"
  build_image "$agent" "$latest"

  rebuilt=$(installed_version "$agent")
  if [[ $rebuilt != "$latest" ]]; then
    printf 'error: rebuilt %s is %s; expected %s\n' "$agent" "$rebuilt" "$latest" >&2
    return 1
  fi

  printf '%s rebuilt at version %s.\n' "$agent" "$rebuilt"
}

[[ $# -ge 2 ]] || usage
[[ ${1:-} != --* ]] || {
  printf 'error: the branch name must come first, before any option\n' >&2
  usage
}

branch_name=$1
shift
agent=
model=
push=false

while [[ ${1:-} == --* ]]; do
  case $1 in
    --agent)
      [[ $# -ge 2 ]] || usage
      agent=$2
      shift 2
      ;;
    --model)
      [[ $# -ge 2 ]] || usage
      model=$2
      shift 2
      ;;
    --push)
      push=true
      shift
      ;;
    *)
      printf 'error: unknown option: %s\n' "$1" >&2
      usage
      ;;
  esac
done

[[ $# -ge 1 ]] || usage
prompt=$*

require_agent "$agent"
validate_agent "$agent"
model=${model:-${agent_default_models[$agent]:-}}

execution_dir=$(pwd -P)
repo_root=$(git -C "$execution_dir" rev-parse --show-toplevel) || {
  printf 'error: %s is not inside a Git repository\n' "$execution_dir" >&2
  exit 1
}

ensure_image_latest "$agent"

git check-ref-format --branch "$branch_name" >/dev/null || {
  printf 'error: invalid branch name: %s\n' "$branch_name" >&2
  exit 2
}

worktree_dir="$(dirname -- "$execution_dir")/$branch_name"
if [[ -e "$worktree_dir" ]]; then
  printf 'error: worktree path already exists: %s\n' "$worktree_dir" >&2
  exit 1
fi

mkdir -p -- "$(dirname -- "$worktree_dir")"

if git -C "$repo_root" show-ref --verify --quiet "refs/heads/$branch_name"; then
  git -C "$repo_root" worktree add "$worktree_dir" "$branch_name"
else
  git -C "$repo_root" worktree add -b "$branch_name" "$worktree_dir"
fi

full_prompt="$prompt agent-sandbox-yolo: proceed autonomously; this task is explicit authorization to create and modify scoped files without a separate preview."

container_args=(
  --rm
  --user "$(id -u):$(id -g)"
  -v "$worktree_dir:/workspace"
)

case $agent in
  codex)
    codex_home="$HOME/.codex"
    [[ -d "$codex_home" ]] || {
      printf 'error: %s does not exist; run codex on the host first\n' "$codex_home" >&2
      exit 1
    }
    container_args+=(
      -it
      -e CODEX_HOME=/codex-home
      -v "$codex_home:/codex-home"
    )
    agent_args=(
      --dangerously-bypass-approvals-and-sandbox
      --model "$model"
      --config 'model_reasoning_effort="high"'
      exec "$full_prompt"
    )
    ;;
  claude)
    claude_home="$HOME/.claude"
    claude_config="$HOME/.claude.json"
    [[ -d "$claude_home" ]] || {
      printf 'error: %s does not exist; run claude on the host first\n' "$claude_home" >&2
      exit 1
    }
    [[ -f "$claude_config" ]] || {
      printf 'error: %s does not exist; run claude on the host first\n' "$claude_config" >&2
      exit 1
    }
    container_args+=(
      -i
      -e HOME=/claude-home
      -v "$claude_home:/claude-home/.claude"
      -v "$claude_config:/claude-home/.claude.json"
    )
    agent_args=(
      --print
      --permission-mode bypassPermissions
      --effort high
      --model "$model"
      "$full_prompt"
    )
    ;;
  opencode)
    opencode_config="$HOME/.config/opencode"
    opencode_data="$HOME/.local/share/opencode"
    opencode_state="$HOME/.local/state/opencode"
    for dir in "$opencode_config" "$opencode_data" "$opencode_state"; do
      [[ -d "$dir" ]] || {
        printf 'error: %s does not exist; run opencode on the host first\n' "$dir" >&2
        exit 1
      }
    done
    # opencode escribe en $HOME (~/.cache) y Docker crea los directorios padre del
    # mount como root. Un tmpfs le da un HOME escribible que vive solo en RAM.
    container_args+=(
      -e HOME=/opencode-home
      -e OPENCODE_DB=/tmp/opencode.db
      --tmpfs /opencode-home:rw,exec,mode=1777
      -v "$opencode_config:/opencode-home/.config/opencode"
      -v "$opencode_data:/opencode-home/.local/share/opencode"
      -v "$opencode_state:/opencode-home/.local/state/opencode"
    )
    agent_args=(run --auto --variant high --print-logs)
    if [[ -n $model ]]; then
      agent_args+=(--model "$model")
    fi
    agent_args+=("$full_prompt")
    ;;
  pi)
    pi_agent_dir="$HOME/.pi/agent"
    [[ -d "$pi_agent_dir" ]] || {
      printf 'error: %s does not exist; run pi on the host first\n' "$pi_agent_dir" >&2
      exit 1
    }
    # HOME es un tmpfs: vive solo en RAM y mantiene todo lo que ejecute el agente
    # (npm, cachés) fuera de un HOME que Docker dejaría sin permisos de escritura.
    #
    # PI_OFFLINE=1 saltea el chequeo de versión al arrancar y la actualización de
    # paquetes, que necesitarían git (no está en la imagen) y escribirían en el mount.
    container_args+=(
      -e HOME=/pi-home
      -e PI_CODING_AGENT_DIR=/pi-agent
      -e PI_OFFLINE=1
      --tmpfs /pi-home:rw,exec,mode=1777
      -v "$pi_agent_dir:/pi-agent"
    )
    agent_args=(-p --thinking high)
    if [[ -n $model ]]; then
      agent_args+=(--model "$model")
    fi
    agent_args+=("$full_prompt")
    ;;
esac

docker run "${container_args[@]}" \
  --entrypoint "${agent_binaries[$agent]}" \
  "$image_name" \
  "${agent_args[@]}"

if [[ $push == false ]]; then
  printf 'Skipping commit and push. Changes left in %s\n' "$worktree_dir"
  exit 0
fi

git -C "$worktree_dir" add -A

if git -C "$worktree_dir" diff --cached --quiet; then
  printf 'Nothing to commit in %s\n' "$worktree_dir"
  exit 0
fi

git -C "$worktree_dir" commit -m "$prompt"
git -C "$worktree_dir" push --set-upstream origin "$branch_name"
