<img width="128" height="128" alt="image" src="https://github.com/user-attachments/assets/f319ca63-1f07-4e6b-bf59-b63b3282f53b" />

# Herald 

A terminal IRC client that connects over WebSocket.

<img width="561" height="431" alt="image" src="https://github.com/user-attachments/assets/ce04c0fd-be4c-4571-a29b-ee469756ad0c" />

## Install

```
go install github.com/medievalsoftware/herald@latest
```

## Usage

```
herald connect [-nick name] [-pass password] wss://server
herald proxy [-listen addr] wss://server
```

**connect** opens a TUI session over WebSocket. **proxy** bridges a local TCP port to a WebSocket server so traditional IRC clients (irssi, weechat, etc.) can connect through it.

### Flags

| Command | Flag | Default | Description |
|---|---|---|---|
| `connect` | `-nick` | `herald` | IRC nickname |
| `connect` | `-pass` | | Server password (SASL PLAIN or PASS fallback) |
| `proxy` | `-listen` | `127.0.0.1:6667` | TCP listen address |

## Authentication

Herald supports two authentication methods, chosen automatically:

1. **SASL PLAIN** — If the server advertises the `sasl` capability, Herald authenticates during capability negotiation using SASL PLAIN (base64-encoded credentials).
2. **PASS fallback** — If SASL is unavailable, the password is sent via the `PASS` command during registration.

Pass the password with `-pass` or use environment variable expansion: `-pass '${IRC_PASS}'`.

## IRC Capabilities

Herald negotiates the following capabilities via `CAP LS 302`:

| Capability | Purpose |
|---|---|
| `batch` | Group related messages (used for chat history) |
| `server-time` | Timestamps on messages from the server |
| `chathistory` | Fetch message history on join |
| `draft/chathistory` | Draft version of chathistory for broader server support |
| `sasl` | SASL authentication (requested when `-pass` is provided) |

## Interface

The TUI has four regions:

- **Tab bar** — Channel tabs across the top. The server buffer tab shows the server hostname.
- **Chat area** — Message history with timestamps, day separators, word wrapping, and clickable links (OSC 8 hyperlinks in supported terminals).
- **Users panel** — Right sidebar showing channel members with op (`@`) and voice (`+`) prefixes, plus a member count header.
- **Input** — Bottom line showing your nick when unfocused. Focuses into one of three input modes.

### Input Modes

| Prompt | Mode | Trigger | Behavior |
|---|---|---|---|
| `>` | Chat | `Enter` | Sends as `PRIVMSG` to current channel |
| `:` | Command | `:` | Runs a herald command with palette completion |
| `"` | Raw | `"` | Sends line directly as raw IRC protocol |

### Command Palette

When typing in `:` command mode or `"` raw mode, a multi-column palette appears above the input showing matching commands. The palette uses fuzzy matching — type any subsequence to filter.

- **Tab** / **Shift+Tab** — Navigate palette entries (nothing is selected until first Tab press)
- **Tab on a match** — Inserts the selected value into the input
- **Argument completion** — After a command name, the palette switches to completing arguments (channels, nicks, settings, themes) based on the command's parameter types

## Key Bindings

### Normal Mode (input unfocused)

| Key | Action |
|---|---|
| `Enter` | Focus input in chat mode |
| `:` | Focus input in command mode |
| `"` | Focus input in raw IRC mode |
| `Ctrl+N` / `Alt+Right` | Next channel tab |
| `Ctrl+P` / `Alt+Left` | Previous channel tab |
| `PgUp` | Scroll chat up |
| `PgDn` | Scroll chat down |
| `Ctrl+C` | Quit |

### Insert Mode (input focused)

| Key | Action |
|---|---|
| `Enter` | Submit input |
| `Escape` / `Ctrl+C` | Cancel and return to normal mode |
| `Tab` | Select next palette entry |
| `Shift+Tab` | Select previous palette entry |
| `Alt+Enter` | Insert newline (multi-line input) |

All key bindings are configurable — see [Configuration](#configurable-key-bindings).

## Commands

### Herald Commands

Entered with `:` prefix in command mode.

| Command | Aliases | Description |
|---|---|---|
| `:join <channel>` | `j` | Join a channel |
| `:leave [channel]` | `part` | Leave current or specified channel |
| `:dm <nick>` | `msg`, `m`, `query` | Send a direct message |
| `:me <action>` | `action` | Send an action (/me) |
| `:nick <name>` | | Change nickname |
| `:set [key [value]]` | | View or change a setting |
| `:theme [name]` | | List or switch color themes |
| `:quit [message]` | `exit`, `q`, `q!` | Disconnect from server |

### Raw IRC Commands

Entered with `"` prefix in raw mode. The palette shows parameter syntax for each command.

| Command | Parameters |
|---|---|
| `JOIN` | `<channel>{,<channel>} [<key>{,<key>}]` |
| `PART` | `<channel>{,<channel>} [<reason>]` |
| `PRIVMSG` | `<target> :<message>` |
| `NOTICE` | `<target> :<message>` |
| `NICK` | `<nickname>` |
| `QUIT` | `[<reason>]` |
| `MODE` | `<target> [<modestring> [<args>...]]` |
| `TOPIC` | `<channel> [<topic>]` |
| `KICK` | `<channel> <user> [<comment>]` |
| `INVITE` | `<nickname> <channel>` |
| `WHO` | `[<mask>]` |
| `WHOIS` | `<nick>{,<nick>}` |
| `LIST` | `[<channel>{,<channel>}]` |
| `NAMES` | `[<channel>{,<channel>}]` |
| `MOTD` | |
| `OPER` | `<name> <password>` |
| `KILL` | `<nickname> <reason>` |
| `SAMODE` | `<target> [<modestring> [<args>...]]` |
| `SAJOIN` | `[<nick>] <channel>{,<channel>}` |
| `UBAN` | `<ADD\|DEL\|LIST\|INFO> [<args>...]` |
| `DLINE` | `[ANDKILL] [<duration>] <ip/net> [<reason>]` |
| `KLINE` | `[ANDKILL] [<duration>] <mask> [<reason>]` |
| `DEFCON` | `[<level>]` |
| `CHATHISTORY` | `<subcommand> <target> <reference> <limit>` |

## Environment Variable Expansion

All input modes support `${VAR}` expansion before sending. This is useful for passwords and secrets:

```
" OPER dane ${IRC_PASS}
```

Only the `${VAR}` form is expanded; bare `$VAR` is left as-is. Missing variables expand to empty string.

## Configuration

Herald reads `$XDG_CONFIG_HOME/herald/config.toml` (default `~/.config/herald/config.toml`).

```toml
timestamp = "3:04 PM"    # Go time format string
users_width = 20         # Width of the users panel
history_limit = 100      # Messages to fetch on join (requires chathistory cap)
theme = "gruvbox"        # Name of a theme file (optional)
```

Settings can be changed at runtime with `:set key value` and are persisted to the config file.

### Configurable Key Bindings

Key bindings use Helix-style notation and merge onto defaults — only specify keys you want to change.

```toml
[keys.normal]
"C-j" = "join"          # Ctrl+J opens :join command
"C-s" = "scroll_up"     # Ctrl+S scrolls chat up

[keys.insert]
# Override insert-mode bindings here
```

**Modifier notation:** `C-` = Ctrl, `A-` = Alt, `S-` = Shift. **Special keys:** `ret`/`enter`, `esc`, `space`, `tab`, `del`, `bs`, `pgup`, `pgdown`, `up`, `down`, `left`, `right`, `home`, `end`.

<details>
<summary>Available actions</summary>

| Action | Description |
|---|---|
| `next_channel` | Switch to next channel tab |
| `prev_channel` | Switch to previous channel tab |
| `quit` | Quit herald |
| `chat` | Focus input in chat mode |
| `command` | Focus input in command mode |
| `raw_mode` | Focus input in raw IRC mode |
| `cancel` | Clear input and return to normal mode |
| `submit` | Submit current input |
| `palette_up` | Move palette selection up |
| `palette_down` | Move palette selection down |
| `scroll_up` | Scroll chat up one page |
| `scroll_down` | Scroll chat down one page |
| `join` | Open `:join` command |
| `leave` | Open `:leave` command |
| `dm` | Open `:dm` command |
| `me` | Open `:me` command |
| `nick` | Open `:nick` command |
| `theme` | Open `:theme` command |
| `set` | Open `:set` command |
| `irc_quit` | Send IRC QUIT and disconnect |

</details>

## Themes

Theme files live in `~/.config/herald/themes/<name>.toml`:

```toml
bar_bg = "#3c3836"
bar_fg = "#ebdbb2"
sel_bg = "#504945"
border = "#928374"
accent = "#fe8019"
green  = "#b8bb26"
yellow = "#fabd2f"
nicks  = [
    "#fb4934", "#b8bb26", "#fabd2f", "#83a598",
    "#d3869b", "#8ec07c", "#fe8019", "#cc241d",
    "#98971a", "#d79921", "#458588", "#b16286",
]
```

| Field | Used for |
|---|---|
| `bar_bg` | Tab bar, users panel header, palette background |
| `bar_fg` | Tab bar text, users panel header text |
| `sel_bg` | Selected palette entry background |
| `border` | Panel borders, palette description border |
| `accent` | Active tab, selected palette text, link color |
| `green` | Nick display in input, ops in users panel |
| `yellow` | Day separators, voiced users in users panel |
| `nicks` | Deterministic nick coloring in chat |

Colors can be hex (`"#83a598"`) or ANSI numbers (`"12"`). Omitted fields fall back to built-in defaults. Switch themes at runtime with `:theme <name>`.

## Proxy Mode

The proxy bridges traditional IRC clients to WebSocket servers:

```
herald proxy [-listen 127.0.0.1:6667] wss://irc.example.com
```

It listens on a local TCP port (default 6667) and relays IRC protocol bidirectionally — TCP lines to WebSocket frames and back. Point any IRC client at the listen address to connect through it.

## Building

Requires Go 1.24+ (managed via [mise](https://mise.jdx.dev)).

```
mise run build
```

Or directly:

```
go build -o herald .
```

## Development

```
mise run check   # fmt + lint + test
mise run fmt     # gofumpt
mise run lint    # golangci-lint
mise run test    # go test ./...
```
