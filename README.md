# cmux-git-diff

A standalone CLI tool that displays `git diff` in a browser with live updates.

Built for use with [cmux](https://cmux.com) browser panes, but works with any browser.

## Features

- **Live reload** — WebSocket pushes diff updates as you edit files
- **Rich diff view** — Powered by [diff2html](https://diff2html.xyz/) with side-by-side and unified views
- **Dark theme** — GitHub-style dark theme, designed for cmux
- **Staged/Unstaged tabs** — Switch between staged, unstaged, or all changes
- **cmux integration** — Automatically opens a browser pane when running inside cmux
- **Single binary** — No runtime dependencies

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

Open `http://localhost:6848` in your browser.

### Options

```
-port     int       Server port (default: 6848)
-bind     string    Bind address (default: localhost)
-interval duration  Polling interval (default: 3s)
```

### cmux

When running inside cmux, `cmux-git-diff` detects `CMUX_WORKSPACE_ID` and automatically opens a browser pane:

```bash
# In a cmux terminal pane
cmux-git-diff
```

## License

MIT
