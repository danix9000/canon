#!/usr/bin/env bash

set -euo pipefail

function help() {
  cat << EOF
Usage: $0 <command> [-s session] [arg ...]

Global Commands:
  kill-server      Kill all sessions
  list-sessions    List all sessions

Session Commands:
  capture          Print the contents of a session
  create           Create or recreate a session
  kill             Kill a session
  send             Send keys to a session
  attach           Attach to a session
EOF
}

function check_no_args() {
  if [ "$#" -gt 0 ]; then
    help
    exit 1
  fi
}

if [[ -z "${1:-}" ]]; then
  help
  exit 1
fi

command="${1:-}"
shift

case "$command" in
  kill-server)
    check_no_args
    tmux -S "$socket" kill-server 2>/dev/null || true
    exit
    ;;

  list-sessions)
    check_no_args
    tmux -S "$socket" list-sessions
    exit
    ;;
esac

session="default"

if [[ "${1:-}" == "-s" ]]; then
  if [[ -z "${2:-}" ]]; then
    help
    exit 1
  fi
  session="$2"
  shift 2
fi

home="${RUNNER_TEMP:-/tmp/canon}/$session"
socket="$home/socket"

case "$command" in
  capture)
    check_no_args
    output=$(tmux -S "$socket" capture-pane -t "$session" -p)
    printf "%s\n" "$output"
    ;;

  create)
    check_no_args
    tmux -S "$socket" kill-session -t "$session" 2>/dev/null || true

    rm -rf "$home"
    mkdir -p "$home"

    cp .canon "$home/.canon"
    cat << EOF | sed 's/^ *//' > "$home/zshrc"
      PROMPT="> "

      alias canon="$(pwd)/canon"
      source <(canon --zsh)
      bindkey '^]' canon_widget

      export HOME="$home"
      cd "$home"
      clear
EOF

    tmux -S "$socket" new-session -d -x 80 -y 20 -s "$session" zsh -f \; send-keys "source '$home/zshrc'" Enter
    ;;

  kill)
    check_no_args
    tmux -S "$socket" kill-session -t "$session" 2>/dev/null || true
    ;;

  send)
    if [ "$#" -eq 0 ]; then
      exit 1
    fi
    tmux -S "$socket" send-keys -t "$session" "$@"
    ;;

  attach)
    check_no_args
    tmux -S "$socket" attach-session -t "$session"
    ;;

  *)
    help
    exit 1
    ;;
esac
