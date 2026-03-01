package format

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/ergochat/irc-go/ircfmt"
)

// IRCToStyled converts IRC formatting codes to a lipgloss-styled string.
// For now, strips formatting and returns plain text. Full color support
// can be layered in later by parsing control codes directly.
func IRCToStyled(text string) string {
	return ircfmt.Strip(text)
}

// Strip removes all IRC formatting codes from text.
func Strip(text string) string {
	return ircfmt.Strip(text)
}

// nickStyles holds pre-computed styles for nick coloring.
var nickStyles = buildNickStyles([]string{
	"1", "2", "3", "4", "5", "6",
	"9", "10", "11", "12", "13", "14",
})

func buildNickStyles(colors []string) []lipgloss.Style {
	styles := make([]lipgloss.Style, len(colors))
	for i, c := range colors {
		styles[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(c))
	}
	return styles
}

// SetNickColors rebuilds the nick color palette from the given color strings.
func SetNickColors(colors []string) {
	if len(colors) == 0 {
		return
	}
	nickStyles = buildNickStyles(colors)
}

// NickColor returns a consistent lipgloss style for a nickname.
func NickColor(nick string) lipgloss.Style {
	h := 0
	for _, c := range nick {
		h = (h*31 + int(c)) & 0x7fffffff
	}
	return nickStyles[h%len(nickStyles)]
}
