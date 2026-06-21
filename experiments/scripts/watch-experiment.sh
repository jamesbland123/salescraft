#!/usr/bin/env bash
set -euo pipefail

repo_root="/Users/james/dev/salescraft"
config_path="experiments/configs/phase1-codex-gpt.json"
session_name="salescraft-exp"
interval_seconds=60
skip_podman=0

usage() {
  cat <<'EOF'
usage: watch-experiment.sh [options]

Options:
  --config PATH       Trial config path. Default: experiments/configs/phase1-codex-gpt.json
  --repo PATH         Golden repo path. Default: /Users/james/dev/salescraft
  --session NAME      tmux session for the experiment. Default: salescraft-exp
  --interval SECONDS  Poll interval. Default: 60
  --skip-podman       Skip Podman preflight. Use only for trials that cannot reach Podman work.
  -h, --help          Show this help.

This script is an outer experiment watcher. It keeps the experiment tmux session
alive after runner exit, polls status, and preserves trial workspaces/artifacts.
It does not modify generated app code, clean workspaces, or reuse failed trials.

Run this from an unrestricted host shell, or from an AI operator session that was
started with no approval prompts and full shell/filesystem/network access. The
watcher cannot answer interactive approval prompts after the human leaves.
EOF
}

log() {
  printf '[salescraft-watch] %s\n' "$*"
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
    --session)
      session_name="${2:-}"
      [ -n "$session_name" ] || die "--session requires a name"
      shift 2
      ;;
    --interval)
      interval_seconds="${2:-}"
      [ -n "$interval_seconds" ] || die "--interval requires a value"
      shift 2
      ;;
    --skip-podman)
      skip_podman=1
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

[ -f "$config_abs" ] || die "config not found: $config_abs"
[ -x "$repo_root/experiments/runner/salescraft-exp" ] || die "runner binary is missing or not executable: experiments/runner/salescraft-exp"

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
watch_root="${TMPDIR:-/tmp}/salescraft-exp-watch"
watch_dir="$watch_root/$trial_id"
status_file="$watch_dir/runner-exit-status.txt"
run_script="$watch_dir/run-in-tmux.sh"

mkdir -p "$watch_dir"

preflight() {
  command -v git >/dev/null 2>&1 || die "git is required"
  command -v tmux >/dev/null 2>&1 || die "tmux is required"
  command -v ps >/dev/null 2>&1 || die "ps is required"

  local dirty
  dirty="$(git status --short)"
  if [ -n "$dirty" ]; then
    printf '%s\n' "$dirty"
    die "golden repo has uncommitted changes; commit or stash before preparing a trial"
  fi

  if [ -e "$workspace_path" ]; then
    die "trial workspace already exists: $workspace_path"
  fi
  if [ -e "$artifact_path" ]; then
    die "artifact directory already exists: $artifact_path"
  fi

  if [ "$skip_podman" -eq 0 ]; then
    command -v podman >/dev/null 2>&1 || die "podman is required; install it or use --skip-podman for non-Podman trials"
    command -v podman-compose >/dev/null 2>&1 || die "podman-compose is required; install it or use --skip-podman for non-Podman trials"
    podman ps >/dev/null || die "podman ps failed; start/fix the Podman machine before launching the experiment"
  fi
}

write_run_script() {
  cat > "$run_script" <<EOF
#!/usr/bin/env bash
set +e
cd "$repo_root"
rm -f "$status_file"
./experiments/runner/salescraft-exp trial --config "$config_abs"
status=\$?
printf "\\n[salescraft-exp] exited with status %s\\n" "\$status"
printf "%s\\n" "\$status" > "$status_file"
echo "Workspace/artifacts retained."
echo "Attach here for inspection, or close this tmux session when done."
while true; do sleep 3600; done
EOF
  chmod +x "$run_script"
}

start_session() {
  rm -f "$status_file"
  write_run_script
  log "starting tmux session '$session_name' for trial=$trial_id"
  tmux new-session -d -s "$session_name" -c "$repo_root" "$run_script"
  log "attach with: tmux attach -t $session_name"
}

session_exists() {
  tmux has-session -t "$session_name" >/dev/null 2>&1
}

print_snapshot() {
  log "trial=$trial_id session=$session_name"
  if session_exists; then
    tmux list-panes -t "$session_name" -F '#{pane_current_command} #{pane_dead} #{pane_dead_status}' || true
    tmux capture-pane -t "$session_name" -p -S -40 || true
  else
    log "tmux session is not running"
  fi
}

print_terminal_summary() {
  local status="$1"
  log "runner exited with status $status"
  log "workspace: $workspace_path"
  log "artifacts:  $artifact_path"

  if [ -f "$workspace_path/BUILD_STATE.md" ]; then
    log "BUILD_STATE.md:"
    sed -n '1,220p' "$workspace_path/BUILD_STATE.md"
  fi

  if [ -d "$artifact_path" ]; then
    log "artifact files:"
    find "$artifact_path" -maxdepth 1 -type f -print | sort
  fi

  if [ "$status" = "0" ]; then
    log "trial completed successfully; workspace retained for manual testing"
    exit 0
  fi

  log "trial stopped before completion; classify this as setup vs app-building before restarting"
  exit "$status"
}

preflight
start_session

while true; do
  sleep "$interval_seconds"

  if [ -f "$status_file" ]; then
    status="$(tr -d '[:space:]' < "$status_file")"
    [ -n "$status" ] || status=1
    print_snapshot
    print_terminal_summary "$status"
  fi

  if ! session_exists; then
    die "tmux session disappeared before the runner wrote an exit status; inspect $workspace_path and $artifact_path"
  fi

  log "still watching trial=$trial_id; next check in ${interval_seconds}s"
done
