#!/usr/bin/env bash
# Create a sibling worktree and open it in the existing codex-cli container.
# Example: ./docker/codex/worktree-codex.sh feature/add-query-log --model gpt-5.6-sol "Add a query-log view."

set -euo pipefail

usage() {
  printf 'Usage: %s [--check-codex-update] | <branch-name> [--model <model>] <prompt...>\n' "${0##*/}" >&2
  exit 2
}

ensure_codex_image_latest() {
  local repo_root=$1
  local installed latest rebuilt

  installed=$(docker run --rm --entrypoint codex codex-cli --version | awk '{print $NF}')
  latest=$(docker run --rm --entrypoint npm codex-cli view @openai/codex version)

  printf 'Codex image version: %s\n' "$installed"
  printf 'Latest version:      %s\n' "$latest"

  if [[ $installed == "$latest" ]]; then
    printf 'Codex image is up to date.\n'
    return
  fi

  printf 'Update available. Rebuilding the Codex image.\n'
  docker build \
    --build-arg "CODEX_VERSION=$latest" \
    --tag codex-cli \
    --file "$repo_root/docker/codex/Dockerfile" \
    "$repo_root"

  rebuilt=$(docker run --rm --entrypoint codex codex-cli --version | awk '{print $NF}')
  if [[ $rebuilt != "$latest" ]]; then
    printf 'error: rebuilt Codex image is %s; expected %s\n' "$rebuilt" "$latest" >&2
    return 1
  fi

  printf 'Codex image rebuilt at version %s.\n' "$rebuilt"
}

if [[ ${1:-} == --check-codex-update ]]; then
  [[ $# -eq 1 ]] || usage
  execution_dir=$(pwd -P)
  repo_root=$(git -C "$execution_dir" rev-parse --show-toplevel) || {
    printf 'error: %s is not inside a Git repository\n' "$execution_dir" >&2
    exit 1
  }
  ensure_codex_image_latest "$repo_root"
  exit 0
fi

[[ $# -ge 2 ]] || usage

branch_name=$1
shift
model=gpt-5.6-terra
if [[ ${1:-} == --model ]]; then
  [[ $# -ge 3 ]] || usage
  model=$2
  shift 2
fi

[[ $# -ge 1 ]] || usage
prompt=$*

execution_dir=$(pwd -P)
repo_root=$(git -C "$execution_dir" rev-parse --show-toplevel) || {
  printf 'error: %s is not inside a Git repository\n' "$execution_dir" >&2
  exit 1
}

ensure_codex_image_latest "$repo_root"

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

codex_home="$HOME/.codex"
mkdir -p -- "$codex_home"

codex_container_args=(
  --rm
  --user "$(id -u):$(id -g)"
  -e CODEX_HOME=/codex-home
  -v "$worktree_dir:/workspace"
  -v "$codex_home:/codex-home"
)

docker run -it "${codex_container_args[@]}" \
  codex-cli \
  --dangerously-bypass-approvals-and-sandbox \
  --model "$model" \
  --config 'model_reasoning_effort="high"' \
  exec "$prompt do not use superpowerer or any spec skills, you decide all"

git -C "$worktree_dir" add -A

git -C "$worktree_dir" commit -m "$prompt"
git -C "$worktree_dir" push --set-upstream origin "$branch_name"
