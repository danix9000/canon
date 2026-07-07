# Canon

A fuzzy command launcher for your terminal. Define common commands in a `.canon` file, then launch Canon to search, pick, and run them.

## Install

```
brew install danix9000/tap/canon
```

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
