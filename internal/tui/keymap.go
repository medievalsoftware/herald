package tui

import (
	"strings"

	"github.com/medievalsoftware/herald/internal/config"
)

// Action identifies a user-facing operation triggered by a key binding.
type Action string

const (
	ActionNextChannel   Action = "next_channel"
	ActionPrevChannel   Action = "prev_channel"
	ActionQuit          Action = "quit"
	ActionChat          Action = "chat"
	ActionCommand       Action = "command"
	ActionCancel        Action = "cancel"
	ActionSubmit        Action = "submit"
	ActionPaletteUp     Action = "palette_up"
	ActionPaletteDown   Action = "palette_down"
	ActionPaletteSelect Action = "palette_select"
	ActionScrollUp      Action = "scroll_up"
	ActionScrollDown    Action = "scroll_down"
	ActionJoin          Action = "join"
	ActionLeave         Action = "leave"
	ActionDM            Action = "dm"
	ActionMe            Action = "me"
	ActionNick          Action = "nick"
	ActionTheme         Action = "theme"
	ActionSet           Action = "set"
	ActionRaw           Action = "raw"
	ActionIRCQuit       Action = "irc_quit"
)

// KeyMap holds resolved key bindings for all modes.
type KeyMap struct {
	Normal map[string]Action
	Insert map[string]Action
}

// DefaultKeyMap returns the built-in key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Normal: map[string]Action{
			"ctrl+c":    ActionQuit,
			"ctrl+n":    ActionNextChannel,
			"ctrl+p":    ActionPrevChannel,
			"alt+right": ActionNextChannel,
			"alt+left":  ActionPrevChannel,
			"enter":     ActionChat,
			":":         ActionCommand,
			"pgup":      ActionScrollUp,
			"pgdown":    ActionScrollDown,
		},
		Insert: map[string]Action{
			"ctrl+c": ActionCancel,
			"esc":    ActionCancel,
			"enter":  ActionSubmit,
			"up":     ActionPaletteUp,
			"down":   ActionPaletteDown,
			"tab":    ActionPaletteSelect,
		},
	}
}

// BuildKeyMap constructs a KeyMap by merging user config onto defaults.
// User overrides individual keys; unspecified keys keep their defaults.
func BuildKeyMap(cfg config.KeysConfig) KeyMap {
	km := DefaultKeyMap()

	for spec, action := range cfg.Normal {
		km.Normal[parseHelixKey(spec)] = Action(action)
	}
	for spec, action := range cfg.Insert {
		km.Insert[parseHelixKey(spec)] = Action(action)
	}

	return km
}

// helixSpecialNames maps Helix key names to bubbletea KeyMsg.String() values.
var helixSpecialNames = map[string]string{
	"ret":       "enter",
	"enter":     "enter",
	"space":     " ",
	"tab":       "tab",
	"del":       "delete",
	"delete":    "delete",
	"bs":        "backspace",
	"backspace": "backspace",
	"esc":       "esc",
	"up":        "up",
	"down":      "down",
	"left":      "left",
	"right":     "right",
	"home":      "home",
	"end":       "end",
	"pageup":    "pgup",
	"pgup":      "pgup",
	"pagedown":  "pgdown",
	"pgdown":    "pgdown",
}

// parseHelixKey converts a Helix-style key spec to a bubbletea KeyMsg.String() value.
// Examples: "C-n" -> "ctrl+n", "A-left" -> "alt+left", "ret" -> "enter".
func parseHelixKey(spec string) string {
	var mods []string
	remaining := spec

	// Parse modifier prefixes (C-, A-, S-).
	for {
		if len(remaining) > 2 && remaining[1] == '-' {
			switch remaining[0] {
			case 'C':
				mods = append(mods, "ctrl")
				remaining = remaining[2:]
				continue
			case 'A':
				mods = append(mods, "alt")
				remaining = remaining[2:]
				continue
			case 'S':
				mods = append(mods, "shift")
				remaining = remaining[2:]
				continue
			}
		}
		break
	}

	// Resolve special name or use as-is.
	key := remaining
	if mapped, ok := helixSpecialNames[strings.ToLower(key)]; ok {
		key = mapped
	}

	if len(mods) == 0 {
		return key
	}
	return strings.Join(mods, "+") + "+" + key
}
