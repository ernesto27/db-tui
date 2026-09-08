#!/usr/bin/env bash
# Smoke test de agent-sandbox.sh. No recibe argumentos: ejecuta todos los agentes
# (codex, claude, opencode, pi) con su modelo por defecto.
#
#
# Uso: ./agent-sandbox/test-agent-sandbox.sh

set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
sandbox_script="$script_dir/agent-sandbox.sh"
prompt="create a file named hello.txt at the repository root containing the text hello world"

agents=(codex claude opencode pi)

repo_root=$(git -C "$script_dir" rev-parse --show-toplevel)
cd -- "$repo_root"


cur_branch=
cur_worktree=

cleanup_current() {
  [[ -n $cur_branch ]] || return 0
  if [[ -e "$cur_worktree" ]]; then
    git -C "$repo_root" worktree remove --force -- "$cur_worktree" >/dev/null 2>&1 \
      || rm -rf -- "$cur_worktree"
  fi
  git -C "$repo_root" worktree prune
  if git -C "$repo_root" show-ref --verify --quiet "refs/heads/$cur_branch"; then
    git -C "$repo_root" branch -D "$cur_branch" >/dev/null
  fi
  cur_branch=
  cur_worktree=
}

trap cleanup_current EXIT


missing_config() {
  local dir
  case $1 in
    codex)
      [[ -d "$HOME/.codex" ]] || { printf '~/.codex is missing'; return; }
      [[ -t 0 ]] || printf 'codex needs a TTY (docker run -it) and stdin is not one'
      ;;
    claude)
      [[ -d "$HOME/.claude" ]] || { printf '~/.claude is missing'; return; }
      [[ -f "$HOME/.claude.json" ]] || printf '~/.claude.json is missing'
      ;;
    opencode)
      for dir in "$HOME/.config/opencode" "$HOME/.local/share/opencode" "$HOME/.local/state/opencode"; do
        [[ -d "$dir" ]] || { printf '%s is missing' "$dir"; return; }
      done
      ;;
    pi)
      [[ -d "$HOME/.pi/agent" ]] || printf '~/.pi/agent is missing'
      ;;
  esac
}

case_failures=0
case_checks=0

ok() {
  case_checks=$((case_checks + 1))
  printf '  ok %d - %s\n' "$case_checks" "$1"
}

ko() {
  case_failures=$((case_failures + 1))
  printf '  not ok - %s\n' "$1" >&2
}


run_case() {
  local agent=$1 status=0
  case_failures=0
  case_checks=0

  cur_branch="agent-sandbox-test-$agent-$(date +%Y%m%d%H%M%S)-$$"
  cur_worktree="$(dirname -- "$repo_root")/$cur_branch"

  printf '  branch:   %s\n' "$cur_branch"
  printf '  worktree: %s\n' "$cur_worktree"

  if [[ $agent == codex ]]; then
    # codex corre con `docker run -it`, así que necesita la TTY de quien lo invoca.
    "$sandbox_script" "$cur_branch" --agent "$agent" "$prompt" || status=$?
  else
    "$sandbox_script" "$cur_branch" --agent "$agent" "$prompt" </dev/null || status=$?
  fi

  if [[ $status -eq 0 ]]; then
    ok 'agent-sandbox.sh exited 0'
  else
    ko "agent-sandbox.sh exited with status $status"
  fi


  if [[ -d "$cur_worktree" ]]; then
    ok 'worktree created'
  else
    ko "worktree not created at $cur_worktree"
  fi

  if [[ -s "$cur_worktree/hello.txt" ]]; then
    ok 'the agent wrote a non-empty hello.txt in the worktree'
  else
    ko 'hello.txt is missing or empty in the worktree'
  fi


  cleanup_current

  [[ $case_failures -eq 0 ]]
}

declare -A results=()
failed=0

for agent in "${agents[@]}"; do
  printf '\n=== %s ===\n' "$agent"

  reason=$(missing_config "$agent")
  if [[ -n $reason ]]; then
    printf '  SKIP: %s\n' "$reason"
    results[$agent]="SKIP ($reason)"
    continue
  fi

  if run_case "$agent"; then
    results[$agent]='PASS'
  else
    results[$agent]="FAIL ($case_failures of $case_checks checks)"
    failed=$((failed + 1))
  fi
done

printf '\n=== summary ===\n'
for agent in "${agents[@]}"; do
  printf '%-9s %s\n' "$agent" "${results[$agent]}"
done

if [[ $failed -gt 0 ]]; then
  printf '\n%d agent(s) failed\n' "$failed" >&2
  exit 1
fi

printf '\nAll good\n'
