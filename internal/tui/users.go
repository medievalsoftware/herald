package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const usersWidth = 20

type usersModel struct {
	members  map[string][]string // channel -> sorted nicks (with prefix @/+)
	active   string
	width    int
	height   int
	viewport viewport.Model
}

func newUsers() usersModel {
	vp := viewport.New(0, 0)
	vp.MouseWheelEnabled = true
	return usersModel{
		members:  make(map[string][]string),
		width:    usersWidth,
		viewport: vp,
	}
}

// SetMembers bulk-sets the nick list for a channel (from RPL_NAMREPLY).
func (m *usersModel) SetMembers(channel string, nicks []string) {
	sorted := make([]string, len(nicks))
	copy(sorted, nicks)
	sortNicks(sorted)
	m.members[channel] = sorted
	if strings.EqualFold(channel, m.active) {
		m.refreshViewport()
	}
}

// AddMember adds a nick to a channel's member list.
func (m *usersModel) AddMember(channel, nick string) {
	for _, existing := range m.members[channel] {
		if stripPrefix(existing) == stripPrefix(nick) {
			return
		}
	}
	m.members[channel] = append(m.members[channel], nick)
	sortNicks(m.members[channel])
	if strings.EqualFold(channel, m.active) {
		m.refreshViewport()
	}
}

// RemoveMember removes a nick from a channel's member list.
func (m *usersModel) RemoveMember(channel, nick string) {
	nicks := m.members[channel]
	for i, n := range nicks {
		if strings.EqualFold(stripPrefix(n), nick) {
			m.members[channel] = append(nicks[:i], nicks[i+1:]...)
			if strings.EqualFold(channel, m.active) {
				m.refreshViewport()
			}
			return
		}
	}
}

// RemoveMemberAll removes a nick from all channels (for QUIT).
func (m *usersModel) RemoveMemberAll(nick string) {
	for ch := range m.members {
		m.RemoveMember(ch, nick)
	}
}

// RenameMember updates a nick across all channels (for NICK).
func (m *usersModel) RenameMember(oldNick, newNick string) {
	for ch, nicks := range m.members {
		for i, n := range nicks {
			if strings.EqualFold(stripPrefix(n), oldNick) {
				prefix := nickPrefix(n)
				m.members[ch][i] = prefix + newNick
				sortNicks(m.members[ch])
				break
			}
		}
	}
	if m.active != "" {
		m.refreshViewport()
	}
}

// SetActive switches which channel's members are displayed.
func (m *usersModel) SetActive(channel string) {
	m.active = channel
	m.refreshViewport()
}

// SetSize updates the panel dimensions.
func (m *usersModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Reserve 1 line for the title.
	vpHeight := height - 1
	if vpHeight < 0 {
		vpHeight = 0
	}
	m.viewport.Width = width
	m.viewport.Height = vpHeight
	m.refreshViewport()
}

// Count returns the number of members in the active channel.
func (m *usersModel) Count() int {
	return len(m.members[m.active])
}

// AllNicks returns deduplicated, prefix-stripped, sorted nicks across all channels.
func (m *usersModel) AllNicks() []string {
	seen := make(map[string]struct{})
	for _, nicks := range m.members {
		for _, n := range nicks {
			seen[stripPrefix(n)] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for nick := range seen {
		out = append(out, nick)
	}
	sort.Strings(out)
	return out
}

func (m usersModel) Update(msg tea.Msg) (usersModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m usersModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("235")).
		Width(m.width).
		Padding(0, 1)

	title := titleStyle.Render(fmt.Sprintf("Users (%d)", m.Count()))

	border := lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	content := title + "\n" + m.viewport.View()
	return border.Render(content)
}

func (m *usersModel) refreshViewport() {
	nicks := m.members[m.active]
	var b strings.Builder
	nickStyle := lipgloss.NewStyle().Padding(0, 1)
	opStyle := nickStyle.Foreground(lipgloss.Color("10"))
	voiceStyle := nickStyle.Foreground(lipgloss.Color("11"))

	for i, nick := range nicks {
		if i > 0 {
			b.WriteByte('\n')
		}
		switch {
		case strings.HasPrefix(nick, "@"):
			b.WriteString(opStyle.Render(nick))
		case strings.HasPrefix(nick, "+"):
			b.WriteString(voiceStyle.Render(nick))
		default:
			b.WriteString(nickStyle.Render(nick))
		}
	}
	m.viewport.SetContent(b.String())
}

// sortNicks sorts nicks: ops (@) first, then voiced (+), then regular, alphabetical within each group.
func sortNicks(nicks []string) {
	sort.Slice(nicks, func(i, j int) bool {
		pi, pj := nickPriority(nicks[i]), nickPriority(nicks[j])
		if pi != pj {
			return pi < pj
		}
		return strings.ToLower(stripPrefix(nicks[i])) < strings.ToLower(stripPrefix(nicks[j]))
	})
}

func nickPriority(nick string) int {
	if strings.HasPrefix(nick, "@") {
		return 0
	}
	if strings.HasPrefix(nick, "+") {
		return 1
	}
	return 2
}

func stripPrefix(nick string) string {
	if len(nick) > 0 && (nick[0] == '@' || nick[0] == '+' || nick[0] == '%' || nick[0] == '~' || nick[0] == '&') {
		return nick[1:]
	}
	return nick
}

func nickPrefix(nick string) string {
	if len(nick) > 0 && (nick[0] == '@' || nick[0] == '+' || nick[0] == '%' || nick[0] == '~' || nick[0] == '&') {
		return string(nick[0])
	}
	return ""
}
