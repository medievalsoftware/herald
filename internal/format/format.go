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
var nickStyles = func() [12]lipgloss.Style {
	colors := [12]lipgloss.Color{
		"1", "2", "3", "4", "5", "6",
		"9", "10", "11", "12", "13", "14",
	}
	var styles [12]lipgloss.Style
	for i, c := range colors {
		styles[i] = lipgloss.NewStyle().Foreground(c)
	}
	return styles
}()

// NickColor returns a consistent lipgloss style for a nickname.
func NickColor(nick string) lipgloss.Style {
	h := 0
	for _, c := range nick {
		h = (h*31 + int(c)) & 0x7fffffff
	}
	return nickStyles[h%len(nickStyles)]
}
