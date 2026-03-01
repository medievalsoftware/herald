package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const inputMaxHeight = 6

type inputModel struct {
	textarea    textarea.Model
	style       lipgloss.Style
	commandMode bool // true = command (:), false = chat (>)
}

func newInput() inputModel {
	ta := textarea.New()
	ta.Placeholder = ""
	ta.CharLimit = 0
	ta.MaxHeight = inputMaxHeight
	ta.ShowLineNumbers = false
	ta.Prompt = "> "
	ta.SetHeight(1)
	// Remove the default Enter key binding — we handle Enter ourselves for submit.
	ta.KeyMap.InsertNewline.SetEnabled(false)

	return inputModel{
		textarea: ta,
		style:    lipgloss.NewStyle().BorderTop(true).BorderStyle(lipgloss.NormalBorder()),
	}
}

func (m inputModel) Update(msg tea.Msg) (inputModel, tea.Cmd) {
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	m.syncHeight()
	return m, cmd
}

// syncHeight grows/shrinks the textarea to match its content.
func (m *inputModel) syncHeight() {
	lines := m.textarea.LineCount()
	if lines < 1 {
		lines = 1
	}
	if lines > inputMaxHeight {
		lines = inputMaxHeight
	}
	m.textarea.SetHeight(lines)
}

// SetWidth updates the textarea width so wrapping is correct during Update.
func (m *inputModel) SetWidth(width int) {
	m.textarea.SetWidth(width)
}

func (m inputModel) View(width int) string {
	if !m.textarea.Focused() {
		return m.style.Width(width).Render("")
	}
	return m.style.Width(width).Render(m.textarea.View())
}

// CommandMode returns whether the input is in command mode.
func (m *inputModel) CommandMode() bool {
	return m.commandMode
}

// SetCommandMode switches between command (:) and chat (>) mode.
func (m *inputModel) SetCommandMode(cmd bool) {
	m.commandMode = cmd
	if cmd {
		m.textarea.Prompt = ": "
	} else {
		m.textarea.Prompt = "> "
	}
}

func (m *inputModel) Value() string {
	return m.textarea.Value()
}

// LineCount returns the number of lines in the textarea.
func (m *inputModel) LineCount() int {
	n := m.textarea.LineCount()
	if n < 1 {
		return 1
	}
	if n > inputMaxHeight {
		return inputMaxHeight
	}
	return n
}

// InsertNewline inserts a newline at the cursor position.
func (m *inputModel) InsertNewline() {
	m.textarea.InsertRune('\n')
	m.syncHeight()
}

// TrimTrailingChar removes the last character from the value.
func (m *inputModel) TrimTrailingChar() {
	val := m.textarea.Value()
	if len(val) > 0 {
		m.textarea.SetValue(val[:len(val)-1])
		m.textarea.CursorEnd()
		m.syncHeight()
	}
}

func (m *inputModel) Reset() {
	m.textarea.Reset()
	m.textarea.SetHeight(1)
}

func (m *inputModel) Focus() tea.Cmd {
	return m.textarea.Focus()
}

func (m *inputModel) Blur() {
	m.textarea.Blur()
}

func (m *inputModel) Focused() bool {
	return m.textarea.Focused()
}

func (m *inputModel) SetValue(s string) {
	m.textarea.SetValue(s)
	m.textarea.CursorEnd()
}
