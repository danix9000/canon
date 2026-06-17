# Canon

Canon is a CLI for accessing and sharing common bash commands. It is currently WIP.

## Usage

Define commands in `~/.canon` (global) or `.canon` (per-directory):

```
# List files in the current directory
ls -la

# Run clippy linter
cargo clippy
```

Lines starting with `#` become the description for the command that follows.

Launch `canon`, type to fuzzy-filter, pick a command, and it's printed to stdout.

## Installation

```sh
go install canon@latest
```

### ZSH keybinding

Add to your `.zshrc` to bind Canon to `Ctrl+Y`:

```zsh
canon_widget() {
  LBUFFER="$(canon)"
  local ret=$?
  zle reset-prompt
  return $ret
}
zle -N canon_widget
bindkey '^y' canon_widget
```
