package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/medievalsoftware/herald/internal/format"
)

// pickerItem represents a message that can be selected in the picker.
type pickerItem struct {
	nick    string
	content string
	time    time.Time
	msgid   string
	action  bool
	system  bool
}

// pickerAction identifies an action the user can take on a selected message.
type pickerAction int

const (
	pickerActionReply pickerAction = iota
	pickerActionReact
	pickerActionDelete
	pickerActionCopy
)

var pickerActions = []struct {
	action pickerAction
	label  string
}{
	{pickerActionReply, "Reply"},
	{pickerActionReact, "React"},
	{pickerActionDelete, "Delete (HISTSERV)"},
	{pickerActionCopy, "Copy msgid"},
}

// pickerResult is returned when the user selects an action.
type pickerResult struct {
	action  pickerAction
	msgid   string
	channel string
	item    pickerItem
}

type pickerModel struct {
	items    []pickerItem // all items (unfiltered)
	filtered []int        // indices into items that match the filter
	cursor   int          // index into filtered
	input    textinput.Model
	width    int
	height   int
	visible  bool
	channel  string // source channel

	// Action menu state.
	showMenu   bool
	menuCursor int
}

func newPicker() pickerModel {
	ti := textinput.New()
	ti.Placeholder = "Search messages..."
	ti.Prompt = "/ "
	ti.CharLimit = 256
	return pickerModel{input: ti}
}

// Open populates the picker with messages from the given channel.
func (p *pickerModel) Open(channel string, lines []chatLine, width, height int) tea.Cmd {
	p.channel = channel
	p.width = width
	p.height = height
	p.items = make([]pickerItem, len(lines))
	for i, l := range lines {
		p.items[i] = pickerItem(l)
	}
	p.input.SetValue("")
	p.applyFilter()
	// Place cursor on last item.
	if len(p.filtered) > 0 {
		p.cursor = len(p.filtered) - 1
	}
	p.visible = true
	p.showMenu = false
	p.menuCursor = 0
	return p.input.Focus()
}

// Close hides the picker and resets state.
func (p *pickerModel) Close() {
	p.visible = false
	p.showMenu = false
	p.items = nil
	p.filtered = nil
	p.input.Blur()
	p.input.SetValue("")
}

// OpenMenu opens the action menu for the currently selected message.
func (p *pickerModel) OpenMenu() bool {
	item, ok := p.SelectedItem()
	if !ok || item.msgid == "" {
		return false
	}
	p.showMenu = true
	p.menuCursor = 0
	p.input.Blur()
	return true
}

// CloseMenu returns to the message list.
func (p *pickerModel) CloseMenu() {
	p.showMenu = false
	p.input.Focus()
}

// MenuUp moves the menu cursor up.
func (p *pickerModel) MenuUp() {
	if p.menuCursor > 0 {
		p.menuCursor--
	}
}

// MenuDown moves the menu cursor down.
func (p *pickerModel) MenuDown() {
	if p.menuCursor < len(pickerActions)-1 {
		p.menuCursor++
	}
}

// MenuSelect returns the selected action result.
func (p *pickerModel) MenuSelect() (pickerResult, bool) {
	item, ok := p.SelectedItem()
	if !ok {
		return pickerResult{}, false
	}
	return pickerResult{
		action:  pickerActions[p.menuCursor].action,
		msgid:   item.msgid,
		channel: p.channel,
		item:    item,
	}, true
}

// SelectedItem returns the currently highlighted item.
func (p *pickerModel) SelectedItem() (pickerItem, bool) {
	if len(p.filtered) == 0 || p.cursor < 0 || p.cursor >= len(p.filtered) {
		return pickerItem{}, false
	}
	return p.items[p.filtered[p.cursor]], true
}

func (p *pickerModel) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.input.Width = width - 4 // account for prompt and padding
}

func (p *pickerModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	prev := p.input.Value()
	p.input, cmd = p.input.Update(msg)
	if p.input.Value() != prev {
		p.applyFilter()
	}
	return cmd
}

func (p *pickerModel) CursorUp() {
	if p.cursor > 0 {
		p.cursor--
	}
}

func (p *pickerModel) CursorDown() {
	if p.cursor < len(p.filtered)-1 {
		p.cursor++
	}
}

func (p *pickerModel) PageUp() {
	pageSize := p.listHeight()
	p.cursor -= pageSize
	if p.cursor < 0 {
		p.cursor = 0
	}
}

func (p *pickerModel) PageDown() {
	pageSize := p.listHeight()
	p.cursor += pageSize
	if p.cursor >= len(p.filtered) {
		p.cursor = len(p.filtered) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

func (p *pickerModel) listHeight() int {
	h := p.height - 5
	if h < 1 {
		h = 1
	}
	return h
}

// applyFilter filters items by fuzzy matching the search input against nick + content.
func (p *pickerModel) applyFilter() {
	filter := strings.TrimSpace(p.input.Value())
	p.filtered = p.filtered[:0]

	if filter == "" {
		for i := range p.items {
			p.filtered = append(p.filtered, i)
		}
	} else {
		for i, item := range p.items {
			candidate := item.nick + " " + item.content
			if _, ok := fuzzyScore(filter, candidate); ok {
				p.filtered = append(p.filtered, i)
			}
		}
	}

	// Clamp cursor.
	if len(p.filtered) == 0 {
		p.cursor = 0
	} else if p.cursor >= len(p.filtered) {
		p.cursor = len(p.filtered) - 1
	}
}

var (
	pickerBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))
	pickerCursorStyle  = lipgloss.NewStyle().Background(lipgloss.Color("237")).Bold(true)
	pickerNickStyle    = lipgloss.NewStyle().Bold(true)
	pickerTsStyle      = lipgloss.NewStyle().Faint(true)
	pickerSystemStyle  = lipgloss.NewStyle().Faint(true)
	pickerActionStyle  = lipgloss.NewStyle().Italic(true)
	pickerCountStyle   = lipgloss.NewStyle().Faint(true)
	pickerMsgidStyle   = lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("243"))
	pickerInputStyle   = lipgloss.NewStyle().PaddingLeft(1)
	pickerNoMsgidStyle = lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("240"))
	menuBorderStyle    = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("214")).
				Padding(0, 1)
	menuSelStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	menuNormalStyle = lipgloss.NewStyle().Faint(true)
	menuHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
)

func (p *pickerModel) View(tsFormat string) string {
	if !p.visible {
		return ""
	}

	// Layout: border eats 2 rows top/bottom, input at bottom takes 1 row.
	innerWidth := p.width - 4 // border left/right + padding
	if innerWidth < 1 {
		innerWidth = 1
	}
	// 2 for top/bottom border, 1 for input, 1 for counter, 1 for separator.
	listHeight := p.height - 5
	if listHeight < 1 {
		listHeight = 1
	}

	// Build message lines (ascending time order).
	var lines []string
	for fi, idx := range p.filtered {
		item := p.items[idx]
		line := p.renderItem(item, fi == p.cursor, tsFormat, innerWidth)
		lines = append(lines, line)
	}

	// Scroll window: keep cursor visible.
	start := 0
	if p.cursor >= listHeight {
		start = p.cursor - listHeight + 1
	}
	end := start + listHeight
	if end > len(lines) {
		end = len(lines)
		start = max(end-listHeight, 0)
	}
	visible := lines[start:end]

	// Pad to fill height.
	for len(visible) < listHeight {
		visible = append(visible, strings.Repeat(" ", innerWidth))
	}

	// Counter line.
	counter := pickerCountStyle.Render(
		padRight(
			"  "+itoa(len(p.filtered))+"/"+itoa(len(p.items))+" messages",
			innerWidth,
		),
	)

	// Msgid preview of selected item.
	var msgidLine string
	if item, ok := p.SelectedItem(); ok && item.msgid != "" {
		msgidLine = pickerMsgidStyle.Render(padRight("  msgid: "+item.msgid, innerWidth))
	} else {
		msgidLine = pickerNoMsgidStyle.Render(padRight("  no msgid", innerWidth))
	}

	// Assemble content.
	content := strings.Join(visible, "\n") + "\n" + counter + "\n" + msgidLine

	box := pickerBorderStyle.Width(p.width - 2).Render(content)
	input := pickerInputStyle.Render(p.input.View())

	result := box + "\n" + input

	// Overlay action menu if open.
	if p.showMenu {
		result = p.overlayMenu(result)
	}

	return result
}

// overlayMenu renders the action menu centered over the picker content.
func (p *pickerModel) overlayMenu(base string) string {
	item, _ := p.SelectedItem()

	// Build menu content.
	var b strings.Builder
	// Header: show the message being acted on.
	header := item.nick + ": " + item.content
	maxHeader := p.width/2 - 4
	if maxHeader > 50 {
		maxHeader = 50
	}
	if lipgloss.Width(header) > maxHeader {
		header = truncate(header, maxHeader)
	}
	b.WriteString(menuHeaderStyle.Render(header))
	b.WriteByte('\n')

	for i, a := range pickerActions {
		prefix := "  "
		if i == p.menuCursor {
			prefix = "> "
			b.WriteString(menuSelStyle.Render(prefix + a.label))
		} else {
			b.WriteString(menuNormalStyle.Render(prefix + a.label))
		}
		if i < len(pickerActions)-1 {
			b.WriteByte('\n')
		}
	}

	menuBox := menuBorderStyle.Render(b.String())

	// Use lipgloss.Place to center the menu over the base.
	baseH := lipgloss.Height(base)
	return lipgloss.Place(
		p.width, baseH,
		lipgloss.Center, lipgloss.Center,
		menuBox,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.NoColor{}),
	)
}

func (p *pickerModel) renderItem(item pickerItem, selected bool, tsFormat string, width int) string {
	ts := item.time.Format(tsFormat) + " "
	tsWidth := lipgloss.Width(ts)

	var text string
	switch {
	case item.system:
		text = "-- " + item.content
	case item.action:
		text = "* " + item.nick + " " + item.content
	default:
		text = item.nick + ": " + item.content
	}

	// Truncate to fit width.
	maxText := width - tsWidth
	if maxText < 0 {
		maxText = 0
	}
	if lipgloss.Width(text) > maxText {
		text = truncate(text, maxText)
	}

	line := ts + text
	// Pad to full width.
	line = padRight(line, width)

	if selected {
		return pickerCursorStyle.Render(line)
	}

	switch {
	case item.system:
		return pickerSystemStyle.Render(line)
	case item.action:
		colored := format.NickColor(item.nick).Render(item.nick)
		ts = pickerTsStyle.Render(ts)
		actionText := "* " + colored + " " + item.content
		if lipgloss.Width(actionText) > maxText {
			actionText = truncate(actionText, maxText)
		}
		rendered := ts + pickerActionStyle.Render(actionText)
		return rendered
	default:
		colored := format.NickColor(item.nick).Render(item.nick)
		ts = pickerTsStyle.Render(ts)
		msgText := pickerNickStyle.Render(colored) + ": " + item.content
		if lipgloss.Width(msgText) > maxText {
			msgText = truncate(msgText, maxText)
		}
		return ts + msgText
	}
}

// truncate cuts a string to fit within width, appending "…" if truncated.
func truncate(s string, width int) string {
	if width <= 1 {
		return "…"
	}
	runes := []rune(s)
	w := 0
	for i, r := range runes {
		rw := lipgloss.Width(string(r))
		if w+rw > width-1 {
			return string(runes[:i]) + "…"
		}
		w += rw
	}
	return s
}

// padRight pads a string with spaces to reach the target width.
func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// itoa converts an int to a string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
