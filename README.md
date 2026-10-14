<div align="center">

# 📝 tracker

**A tiny, fast, keyboard-driven terminal notebook — daily notes and pinned notes, one keystroke away.**

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Bubble Tea](https://img.shields.io/badge/Built%20with-Bubble%20Tea-FF5FAF?style=for-the-badge)
![SQLite](https://img.shields.io/badge/Storage-SQLite-003B57?style=for-the-badge&logo=sqlite&logoColor=white)
![Platforms](https://img.shields.io/badge/Linux%20%7C%20Windows%20%7C%20macOS-8A8A8A?style=for-the-badge)

<br>

<img src="demo.gif" alt="tracker demo" width="860">

</div>

<br>

## ✨ Features

- 📅 **Daily notes** — hit `n` and start typing; today's note is always one key away
- 📌 **Pinned notes** — keep long-lived notes on a separate tab, rename, delete or promote a daily note
- 💾 **Autosave** — every keystroke is saved, nothing to lose
- 📋 **Clean copy** — `ctrl+y` copies the raw note, no line numbers or gutters
- 🪶 **Single binary** — pure Go, no CGO, no runtime dependencies
- 🗄️ **Local-first** — everything lives in `~/.tracker/notes.db`

## 🚀 Install

```bash
git clone <repo-url> tracker && cd tracker
make install     # installs to $(go env GOPATH)/bin
```

Or just run it from source:

```bash
make run
```

Cross-compile with `make linux`, `make linux-arm64`, `make windows` or `make windows-arm64` (binaries land in `dist/`).

## ⌨️ Keybindings

| Where | Key | Action |
| --- | --- | --- |
| Anywhere | `tab` | Switch between **Notes** and **Pinned** |
| Anywhere | `q` / `ctrl+c` | Quit |
| Lists | `↑`/`k` · `↓`/`j` | Move selection |
| Lists | `enter` | Open the selected note |
| Notes | `n` | Open today's note |
| Notes | `p` | Pin the selected note (copies it to Pinned) |
| Pinned | `n` | Create a new pinned note |
| Pinned | `r` | Rename |
| Pinned | `d` then `y` | Delete (with confirmation) |
| Editor | `ctrl+y` | Copy the note to the clipboard |
| Editor | `esc` | Save and go back |

## 🛠️ Development

```bash
make build   # build for the current OS
make test    # run tests
make vet     # go vet
```

The demo above is recorded with [VHS](https://github.com/charmbracelet/vhs): `vhs demo.tape`.

<div align="center">

Made with 💜 and [Bubble Tea](https://github.com/charmbracelet/bubbletea)

</div>
