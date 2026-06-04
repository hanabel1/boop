# boop

A TUI dashboard for your open GitHub pull requests, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Setup

Requires a GitHub personal access token with repo access:

```
export GITHUB_TOKEN=your_token_here
```

## Run

```
go run main.go
```

## Controls

| Key | Action |
|-----|--------|
| `j` / `↓` | Next PR |
| `k` / `↑` | Previous PR |
| `Enter` | Open PR in browser |
| `q` | Quit |