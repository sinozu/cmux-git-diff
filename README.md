# cmux-git-diff

A standalone CLI tool that displays `git diff` in a browser with live updates.

Built for use with [cmux](https://cmux.com) browser panes, but works with any browser.

## Features

- **Live reload** — WebSocket pushes diff updates as you edit files
- **Rich diff view** — Powered by [diff2html](https://diff2html.xyz/) with side-by-side and unified views
- **Dark theme** — GitHub-style dark theme, designed for cmux
- **Staged/Unstaged tabs** — Switch between staged, unstaged, or all changes
- **cmux integration** — Automatically opens a browser tab in the current pane when running inside cmux
- **Multi-workspace safe** — Each instance uses an OS-assigned port, no conflicts
- **Single binary** — No runtime dependencies
- **Security** — Localhost-only by default; WebSocket Origin check enforced on non-localhost binds

## Installation

```bash
go install github.com/sinozu/cmux-git-diff@latest
```

Or build from source:

```bash
git clone https://github.com/sinozu/cmux-git-diff.git
cd cmux-git-diff
go build -o cmux-git-diff .
```

## Usage

Run inside any git repository:

```bash
cmux-git-diff
```

The server starts on a random available port. The URL is printed to the terminal.

### Options

```
-port     int       Server port (default: 0 = auto-assign)
-bind     string    Bind address (default: localhost)
-interval duration  Polling interval (default: 3s)
-pane               Open in a new pane instead of a tab
```

### cmux

When running inside cmux, `cmux-git-diff` detects `CMUX_WORKSPACE_ID` and opens a browser tab alongside the terminal in the same pane:

```bash
# Browser tab opens in the same pane (default)
cmux-git-diff

# Browser opens in a separate pane
cmux-git-diff -pane
```

Multiple workspaces can run `cmux-git-diff` simultaneously without port conflicts.

### Security

By default the server binds to `localhost` only. If you bind to a non-localhost address (`-bind 0.0.0.0`), a warning is logged and WebSocket Origin checks are enforced to prevent cross-site access.

## License

MIT
