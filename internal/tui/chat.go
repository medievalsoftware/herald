package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/medievalsoftware/herald/internal/format"
	"github.com/muesli/termenv"
)

type chatLine struct {
	nick    string
	content string
	time    time.Time
	action  bool // true for /me actions
	system  bool // true for join/part/quit/etc
}

type chatModel struct {
	viewport        viewport.Model
	messages        map[string][]chatLine // channel -> messages
	active          string                // current channel/buffer
	width           int
	height          int
	timestampFormat string
}

func newChat(timestampFormat string) chatModel {
	vp := viewport.New(0, 0)
	vp.MouseWheelEnabled = true
	return chatModel{
		viewport:        vp,
		messages:        make(map[string][]chatLine),
		timestampFormat: timestampFormat,
	}
}

func (m chatModel) Update(msg tea.Msg) (chatModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m chatModel) View() string {
	return m.viewport.View()
}

// AddMessage appends a message to a channel buffer.
func (m *chatModel) AddMessage(channel, nick, content string) {
	m.messages[channel] = append(m.messages[channel], chatLine{
		nick:    nick,
		content: content,
		time:    time.Now(),
	})
	if channel == m.active {
		m.refreshViewport()
	}
}

// AddAction appends an action (/me) to a channel buffer.
func (m *chatModel) AddAction(channel, nick, content string) {
	m.messages[channel] = append(m.messages[channel], chatLine{
		nick:    nick,
		content: content,
		time:    time.Now(),
		action:  true,
	})
	if channel == m.active {
		m.refreshViewport()
	}
}

// AddSystemMessage appends a system message (join, part, etc.) to a channel buffer.
func (m *chatModel) AddSystemMessage(channel, content string) {
	m.messages[channel] = append(m.messages[channel], chatLine{
		content: content,
		time:    time.Now(),
		system:  true,
	})
	if channel == m.active {
		m.refreshViewport()
	}
}

// PrependMessages inserts history messages at the beginning of a channel buffer.
func (m *chatModel) PrependMessages(channel string, lines []chatLine) {
	m.messages[channel] = append(lines, m.messages[channel]...)
	if channel == m.active {
		m.refreshViewport()
	}
}

// SetActive switches the displayed channel.
func (m *chatModel) SetActive(channel string) {
	m.active = channel
	m.refreshViewport()
}

// ClearHistory removes all stored messages for the given channel.
func (m *chatModel) ClearHistory(channel string) {
	delete(m.messages, channel)
}

// ScrollUp scrolls the viewport up by one page.
func (m *chatModel) ScrollUp() {
	m.viewport.PageUp()
}

// ScrollDown scrolls the viewport down by one page.
func (m *chatModel) ScrollDown() {
	m.viewport.PageDown()
}

// SetSize updates the viewport dimensions.
func (m *chatModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
	m.refreshViewport()
}

var (
	chatNickStyle   = lipgloss.NewStyle().Bold(true)
	chatSystemStyle = lipgloss.NewStyle().Faint(true)
	chatActionStyle = lipgloss.NewStyle().Italic(true)
	chatTsStyle     = lipgloss.NewStyle().Faint(true)
	separatorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	linkStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
)

func (m *chatModel) refreshViewport() {
	lines := m.messages[m.active]
	var b strings.Builder

	today := toDate(time.Now())
	var prevDate time.Time

	// Compute timestamp prefix width once (format a sample time).
	tsPrefix := time.Date(2006, 1, 2, 15, 4, 5, 0, time.Local).Format(m.timestampFormat) + " "
	tsPrefixWidth := lipgloss.Width(tsPrefix)

	for _, line := range lines {
		lineDate := toDate(line.time)

		// Insert day separator when date changes.
		if !lineDate.IsZero() && lineDate != prevDate {
			label := dateSeparatorLabel(lineDate, today)
			if label != "" {
				if b.Len() > 0 {
					b.WriteByte('\n')
				}
				b.WriteString(renderSeparator(separatorStyle, label, m.width))
			}
		}
		prevDate = lineDate

		if b.Len() > 0 {
			b.WriteByte('\n')
		}

		ts := chatTsStyle.Render(line.time.Format(m.timestampFormat)) + " "

		switch {
		case line.system:
			text := "-- " + linkify(line.content)
			wrapped := wordWrap(text, m.width-tsPrefixWidth)
			wrapped = indent(wrapped, tsPrefixWidth)
			b.WriteString(ts + chatSystemStyle.Render(wrapped))
		case line.action:
			colored := format.NickColor(line.nick).Render(line.nick)
			prefix := "* " + colored + " "
			prefixWidth := lipgloss.Width(prefix) + tsPrefixWidth
			wrapped := wordWrap(linkify(line.content), m.width-prefixWidth)
			wrapped = indent(wrapped, prefixWidth)
			b.WriteString(ts + chatActionStyle.Render(prefix+wrapped))
		default:
			colored := format.NickColor(line.nick).Render(line.nick)
			prefix := chatNickStyle.Render(colored) + " "
			prefixWidth := lipgloss.Width(prefix) + tsPrefixWidth
			wrapped := wordWrap(linkify(line.content), m.width-prefixWidth)
			wrapped = indent(wrapped, prefixWidth)
			b.WriteString(ts + prefix + wrapped)
		}
	}

	m.viewport.SetContent(b.String())
	m.viewport.GotoBottom()
}

// toDate truncates a time to midnight in local timezone.
func toDate(t time.Time) time.Time {
	y, mo, d := t.Date()
	return time.Date(y, mo, d, 0, 0, 0, 0, t.Location())
}

// dateSeparatorLabel returns the label for a day separator between messages.
func dateSeparatorLabel(lineDate, today time.Time) string {
	days := int(today.Sub(lineDate).Hours() / 24)
	switch {
	case days == 0:
		return "Today"
	case days == 1:
		return "Yesterday"
	case days >= 2 && days <= 6:
		return lineDate.Weekday().String()
	default:
		return lineDate.Format("02 Jan 2006")
	}
}

// renderSeparator returns a centered label padded with ─ fill characters.
func renderSeparator(style lipgloss.Style, label string, width int) string {
	if width <= 0 {
		return style.Render(label)
	}
	padded := " " + label + " "
	paddedWidth := lipgloss.Width(padded)
	remaining := width - paddedWidth
	if remaining <= 0 {
		return style.Render(padded)
	}
	left := remaining / 2
	right := remaining - left
	line := strings.Repeat("─", left) + padded + strings.Repeat("─", right)
	return style.Render(line)
}

// wordWrap breaks text into lines that fit within width, splitting on word
// boundaries. Words longer than width are hard-broken.
func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}

	var b strings.Builder
	for li, line := range strings.Split(text, "\n") {
		if li > 0 {
			b.WriteByte('\n')
		}
		col := 0
		for _, token := range splitTokens(line) {
			tl := lipgloss.Width(token)
			if col+tl > width && col > 0 {
				b.WriteByte('\n')
				col = 0
				// Skip leading space on wrapped line.
				token = strings.TrimLeft(token, " ")
				tl = lipgloss.Width(token)
			}
			if tl <= width-col || col == 0 {
				b.WriteString(token)
				col += tl
			} else {
				col = hardBreak(&b, token, width, col)
			}
		}
	}
	return b.String()
}

// splitTokens splits a line into alternating whitespace and non-whitespace tokens,
// preserving all original spacing.
func splitTokens(line string) []string {
	var tokens []string
	i := 0
	for i < len(line) {
		if line[i] == ' ' || line[i] == '\t' {
			j := i
			for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
				j++
			}
			tokens = append(tokens, line[i:j])
			i = j
		} else {
			j := i
			for j < len(line) && line[j] != ' ' && line[j] != '\t' {
				j++
			}
			tokens = append(tokens, line[i:j])
			i = j
		}
	}
	return tokens
}

// hardBreak writes a word that exceeds width by splitting it across lines.
// Words containing ANSI escape sequences (e.g. OSC 8 hyperlinks) are written
// intact to avoid corrupting the sequences.
func hardBreak(b *strings.Builder, word string, width, col int) int {
	if strings.Contains(word, "\x1b") {
		b.WriteString(word)
		return col + lipgloss.Width(word)
	}
	for _, r := range word {
		rw := lipgloss.Width(string(r))
		if col+rw > width && col > 0 {
			b.WriteByte('\n')
			col = 0
		}
		b.WriteRune(r)
		col += rw
	}
	return col
}

// linkify wraps URLs in OSC 8 hyperlink sequences so they're clickable
// in terminals that support it, and applies underline styling for visibility.
// Applied before word-wrapping; hardBreak skips words containing escape
// sequences to avoid corrupting the hyperlink.
func linkify(text string) string {
	var b strings.Builder
	remaining := text
	for {
		// Find the next URL start.
		idx := strings.Index(remaining, "http://")
		idxs := strings.Index(remaining, "https://")
		if idx < 0 && idxs < 0 {
			b.WriteString(remaining)
			break
		}
		if idx < 0 || (idxs >= 0 && idxs < idx) {
			idx = idxs
		}

		// Write everything before the URL.
		b.WriteString(remaining[:idx])

		// Find the end of the URL (next whitespace or end of string).
		rest := remaining[idx:]
		end := strings.IndexAny(rest, " \t\n")
		if end < 0 {
			end = len(rest)
		}
		url := rest[:end]

		b.WriteString(termenv.Hyperlink(url, linkStyle.Render(url)))
		remaining = rest[end:]
	}
	return b.String()
}

// indent prefixes all lines after the first with n spaces.
func indent(text string, n int) string {
	if !strings.Contains(text, "\n") {
		return text
	}
	pad := strings.Repeat(" ", n)
	lines := strings.Split(text, "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = pad + lines[i]
	}
	return strings.Join(lines, "\n")
}
