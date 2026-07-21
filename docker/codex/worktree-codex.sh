#!/usr/bin/env bash
# Create a sibling worktree and open it in the existing codex-cli container.

set -euo pipefail

usage() {
  printf 'Usage: %s <branch-name> [--model <model>] <prompt...>\n' "${0##*/}" >&2
  exit 2
}

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
  exec "$prompt"



git -C "$worktree_dir" commit -m "$prompt"
git -C "$worktree_dir" push --set-upstream origin "$branch_name"
