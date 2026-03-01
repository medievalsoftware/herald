# Herald

A terminal IRC client that connects over WebSocket.

<img width="561" height="431" alt="image" src="https://github.com/user-attachments/assets/ce04c0fd-be4c-4571-a29b-ee469756ad0c" />


## Usage

```
herald connect [-nick name] wss://server
herald proxy [-listen addr] wss://server
```

**connect** opens a TUI session. **proxy** bridges a local TCP port to a WebSocket server so traditional IRC clients can connect through it.

## Key Bindings

| Key | Action |
|---|---|
| `Enter` | Start typing a message |
| `:` | Enter command mode |
| `Escape` | Cancel input |
| `Ctrl+N` / `Alt+Right` | Next channel |
| `Ctrl+P` / `Alt+Left` | Previous channel |
| `Tab` | Accept palette completion |
| `Alt+Enter` | Insert newline |
| `Ctrl+C` | Quit |

## Commands

| Command | Aliases | Description |
|---|---|---|
| `:join <channel>` | `j` | Join a channel |
| `:leave [channel]` | `part`, `q` | Leave current or specified channel |
| `:msg <target> <text>` | `m`, `query` | Send a direct message |
| `:me <action>` | `action` | Send an action |
| `:nick <name>` | | Change nickname |
| `:set [key [value]]` | | View or change settings |
| `:theme [name]` | | List or switch color themes |
| `:raw <line>` | `quote` | Send raw IRC command |
| `:quit [message]` | `exit`, `q!` | Disconnect from server |

All commands support tab-completion via the command palette.

## Configuration

Herald reads `$XDG_CONFIG_HOME/herald/config.toml` (default `~/.config/herald/config.toml`).

```toml
timestamp = "3:04 PM"    # Go time format string
users_width = 20         # Width of the users panel
history_limit = 100      # Messages to fetch on join (requires chathistory)
theme = "gruvbox"        # Name of a theme file (optional)
```

Settings can also be changed at runtime with `:set`.

## Themes

Theme files live in `~/.config/herald/themes/<name>.toml`. Each file defines colors directly at the top level:

```toml
bar_bg = "#3c3836"
bar_fg = "#ebdbb2"
border = "#928374"
accent = "#83a598"
green  = "#b8bb26"
yellow = "#fabd2f"
nicks  = [
    "#fb4934", "#b8bb26", "#fabd2f", "#83a598",
    "#d3869b", "#8ec07c", "#fe8019", "#cc241d",
    "#98971a", "#d79921", "#458588", "#b16286",
]
```

Color values can be hex (`"#83a598"`) or ANSI numbers (`"12"`). Any omitted field falls back to the built-in default. Use `:theme <name>` to switch at runtime, or set `theme = "name"` in config.toml.

## Building

Requires Go 1.24+ (managed via [mise](https://mise.jdx.dev)).

```
mise run build
```

Or directly:

```
go build -o herald ./cmd/herald
```

## Development

```
mise run check   # fmt + lint + test
mise run fmt     # gofumpt
mise run lint    # golangci-lint
mise run test    # go test ./...
```
