package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type channelsModel struct {
	tabs   []string
	active int
}

func newChannels() channelsModel {
	return channelsModel{
		tabs: []string{"*server*"},
	}
}

// Add appends a channel tab if it doesn't already exist. Returns the index.
func (m *channelsModel) Add(name string) int {
	for i, t := range m.tabs {
		if strings.EqualFold(t, name) {
			return i
		}
	}
	m.tabs = append(m.tabs, name)
	return len(m.tabs) - 1
}

// Remove removes a channel tab and adjusts the active index.
func (m *channelsModel) Remove(name string) {
	for i, t := range m.tabs {
		if strings.EqualFold(t, name) {
			m.tabs = append(m.tabs[:i], m.tabs[i+1:]...)
			if m.active >= len(m.tabs) {
				m.active = len(m.tabs) - 1
			}
			return
		}
	}
}

// SetActive switches to the tab by name, returns the name for buffer lookup.
func (m *channelsModel) SetActive(name string) string {
	for i, t := range m.tabs {
		if strings.EqualFold(t, name) {
			m.active = i
			return t
		}
	}
	return m.tabs[m.active]
}

// Next cycles to the next tab.
func (m *channelsModel) Next() string {
	m.active = (m.active + 1) % len(m.tabs)
	return m.tabs[m.active]
}

// Prev cycles to the previous tab.
func (m *channelsModel) Prev() string {
	m.active--
	if m.active < 0 {
		m.active = len(m.tabs) - 1
	}
	return m.tabs[m.active]
}

// Active returns the current tab name.
func (m *channelsModel) Active() string {
	return m.tabs[m.active]
}

// Tabs returns a copy of the tab list.
func (m *channelsModel) Tabs() []string {
	out := make([]string, len(m.tabs))
	copy(out, m.tabs)
	return out
}

var (
	channelActiveStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Background(lipgloss.Color("235")).Padding(0, 1)
	channelInactiveStyle = lipgloss.NewStyle().Faint(true).Padding(0, 1)
	channelBarStyle      = lipgloss.NewStyle().BorderBottom(true).BorderStyle(lipgloss.NormalBorder())
)

func (m channelsModel) View(width int) string {
	var parts []string
	for i, tab := range m.tabs {
		if i == m.active {
			parts = append(parts, channelActiveStyle.Render(tab))
		} else {
			parts = append(parts, channelInactiveStyle.Render(tab))
		}
	}

	bar := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	return channelBarStyle.Width(width).Render(bar)
}
