# Herald

A terminal IRC client that connects over WebSocket.

## Usage

```
herald connect [-nick name] wss://server
herald proxy [-listen addr] wss://server
```

**connect** opens a TUI session. **proxy** bridges a local TCP port to a WebSocket server so traditional IRC clients can connect through it.

## Building

Requires Go 1.24+ (managed via [mise](https://mise.jdx.dev)).

```
mise run build
```

Or directly:

```
go build -o herald ./cmd/herald
```

## Configuration

Herald reads `$XDG_CONFIG_HOME/herald/config.toml` (default `~/.config/herald/config.toml`).

```toml
timestamp = "15:04"  # Go time format string
```

## Development

```
mise run check   # fmt + lint + test
mise run fmt     # gofumpt
mise run lint    # golangci-lint
mise run test    # go test ./...
```
