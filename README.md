# tracker

A terminal note-taking app built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). One note per day, plus pinned notes that stick around. Edits autosave to a local SQLite database at `~/.tracker/notes.db`.

## Install

Requires Go 1.26+. No CGO needed.

```bash
make install   # installs to $(go env GOPATH)/bin
make run       # or run straight from source
```

Cross-compile with `make linux`, `make linux-arm64`, `make windows`, `make windows-arm64` (output in `dist/`).

## Keys

| Where | Key | Action |
|---|---|---|
| Lists | `tab` | Switch between daily and pinned notes |
| | `↑`/`k`, `↓`/`j` | Move |
| | `enter` | Open note |
| | `q` | Quit |
| Daily | `n` | Open today's note |
| | `p` | Pin a copy of the selected day |
| Pinned | `n` | New pinned note |
| | `r` | Rename |
| | `d` then `y` | Delete |
| Editor | `ctrl+y` | Copy note to clipboard |
| | `esc` | Save and go back |
| | `ctrl+c` | Save and quit |
