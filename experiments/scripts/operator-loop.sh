#!/usr/bin/env bash
set -euo pipefail

repo_root="/Users/james/dev/salescraft"
config_path="experiments/configs/phase1-codex-gpt.json"
prompt_path="experiments/prompts/operator-loop.md"
operator_session="salescraft-operator"
watch_session="salescraft-watch"
experiment_session="salescraft-exp"
interval_seconds=900
once=0

usage() {
  cat <<'EOF'
usage: operator-loop.sh [options]

Options:
  --config PATH              Trial config path. Default: experiments/configs/phase1-codex-gpt.json
  --repo PATH                Golden repo path. Default: /Users/james/dev/salescraft
  --prompt PATH              Operator prompt path. Default: experiments/prompts/operator-loop.md
  --operator-session NAME    tmux session name used for this loop. Default: salescraft-operator
  --watch-session NAME       watcher tmux session name. Default: salescraft-watch
  --experiment-session NAME  experiment tmux session name. Default: salescraft-exp
  --interval SECONDS         Delay between AI operator turns. Default: 900
  --once                     Run one operator turn, then exit.
  -h, --help                 Show this help.

This script is a durable AI operator loop. It periodically snapshots local
experiment state and invokes Codex in no-approval/full-access mode for one
bounded operator turn. The operator prompt forbids generated app-code edits and
limits intervention to experiment setup, host environment, tmux, Podman, config,
logging, and artifact handling.

Run it from an unrestricted host shell in its own tmux session. Do not use this
as the evaluated model under test.
EOF
}

log() {
  printf '[salescraft-operator] %s\n' "$*"
}

die() {
  log "error: $*"
  exit 1
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --config)
      config_path="${2:-}"
      [ -n "$config_path" ] || die "--config requires a path"
      shift 2
      ;;
    --repo)
      repo_root="${2:-}"
      [ -n "$repo_root" ] || die "--repo requires a path"
      shift 2
      ;;
    --prompt)
      prompt_path="${2:-}"
      [ -n "$prompt_path" ] || die "--prompt requires a path"
      shift 2
      ;;
    --operator-session)
      operator_session="${2:-}"
      [ -n "$operator_session" ] || die "--operator-session requires a name"
      shift 2
      ;;
    --watch-session)
      watch_session="${2:-}"
      [ -n "$watch_session" ] || die "--watch-session requires a name"
      shift 2
      ;;
    --experiment-session)
      experiment_session="${2:-}"
      [ -n "$experiment_session" ] || die "--experiment-session requires a name"
      shift 2
      ;;
    --interval)
      interval_seconds="${2:-}"
      [ -n "$interval_seconds" ] || die "--interval requires a value"
      shift 2
      ;;
    --once)
      once=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown argument: $1"
      ;;
  esac
done

cd "$repo_root"

case "$config_path" in
  /*) config_abs="$config_path" ;;
  *) config_abs="$repo_root/$config_path" ;;
esac

case "$prompt_path" in
  /*) prompt_abs="$prompt_path" ;;
  *) prompt_abs="$repo_root/$prompt_path" ;;
esac

[ -f "$config_abs" ] || die "config not found: $config_abs"
[ -f "$prompt_abs" ] || die "prompt not found: $prompt_abs"
command -v codex >/dev/null 2>&1 || die "codex CLI is required for the AI operator loop"
command -v tmux >/dev/null 2>&1 || die "tmux is required"

json_string() {
  local key="$1"
  sed -nE "s/^[[:space:]]*\"${key}\"[[:space:]]*:[[:space:]]*\"([^\"]*)\".*/\1/p" "$config_abs" | head -n 1
}

trial_id="$(json_string trial_id)"
workspace_root="$(json_string workspace_root)"
artifact_root="$(json_string artifact_root)"

[ -n "$trial_id" ] || die "trial_id not found in $config_abs"
[ -n "$workspace_root" ] || workspace_root="experiments/trials"
[ -n "$artifact_root" ] || artifact_root="experiments/artifacts"

workspace_path="$repo_root/$workspace_root/$trial_id"
artifact_path="$repo_root/$artifact_root/$trial_id"
operator_root="${TMPDIR:-/tmp}/salescraft-exp-operator"
operator_dir="$operator_root/$trial_id"
snapshot_file="$operator_dir/snapshot.md"
turn_prompt="$operator_dir/operator-turn.md"
turn_log="$operator_dir/operator-turn.log"

mkdir -p "$operator_dir"

append_command() {
  local title="$1"
  shift

  {
    printf '\n### %s\n\n' "$title"
    printf '```text\n'
    "$@" 2>&1 || true
    printf '```\n'
  } >> "$snapshot_file"
}

capture_session() {
  local session="$1"

  if tmux has-session -t "$session" >/dev/null 2>&1; then
    append_command "tmux list-panes -t $session" tmux list-panes -t "$session" -F '#{session_name} #{pane_current_command} dead=#{pane_dead} status=#{pane_dead_status}'
    append_command "tmux capture-pane -t $session" tmux capture-pane -t "$session" -p -S -120
  else
    {
      printf '\n### tmux session %s\n\n' "$session"
      printf '```text\nnot running\n```\n'
    } >> "$snapshot_file"
  fi
}

relevant_processes() {
  ps -axo pid,etime,command | grep -E 'salescraft|codex exec|claude|opencode|aider|openhands|pnpm|turbo|podman|tmux' | grep -v grep || true
}

write_snapshot() {
  {
    printf '# Current Operator Snapshot\n\n'
    printf '- time: %s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
    printf '- repo: %s\n' "$repo_root"
    printf '- config: %s\n' "$config_abs"
    printf '- trial_id: %s\n' "$trial_id"
    printf '- workspace: %s\n' "$workspace_path"
    printf '- artifacts: %s\n' "$artifact_path"
  } > "$snapshot_file"

  append_command "git status --short" git status --short
  append_command "tmux list-sessions" tmux list-sessions
  capture_session "$watch_session"
  capture_session "$experiment_session"
  append_command "relevant processes" relevant_processes
  append_command "podman ps" podman ps

  if [ -d "$artifact_path" ]; then
    append_command "artifact files" find "$artifact_path" -maxdepth 1 -type f -print
  fi

  if [ -f "$workspace_path/BUILD_STATE.md" ]; then
    append_command "BUILD_STATE.md" sed -n '1,240p' "$workspace_path/BUILD_STATE.md"
  fi

  if [ -f "$workspace_path/VERIFY_LOG.md" ]; then
    append_command "VERIFY_LOG.md" sed -n '1,240p' "$workspace_path/VERIFY_LOG.md"
  fi
}

run_operator_turn() {
  write_snapshot
  {
    cat "$prompt_abs"
    printf '\n\n'
    cat "$snapshot_file"
  } > "$turn_prompt"

  log "starting AI operator turn for trial=$trial_id"
  set +e
  codex exec --disable tui_app_server --dangerously-bypass-approvals-and-sandbox < "$turn_prompt" 2>&1 | tee -a "$turn_log"
  local status=${PIPESTATUS[0]}
  set -e
  log "AI operator turn exited with status $status"
  return "$status"
}

log "operator loop ready for trial=$trial_id"
log "state directory: $operator_dir"
log "watching tmux sessions: $watch_session, $experiment_session"
log "this loop session should normally be named: $operator_session"

while true; do
  run_operator_turn || true

  if [ "$once" -eq 1 ]; then
    log "--once set; exiting"
    exit 0
  fi

  log "sleeping ${interval_seconds}s before next operator turn"
  sleep "$interval_seconds"
done
