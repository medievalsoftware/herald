package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/medievalsoftware/herald/internal/client"
	"github.com/medievalsoftware/herald/internal/config"
	"github.com/medievalsoftware/herald/internal/format"
)

const serverBuffer = "*server*"

var (
	notifyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Italic(true).PaddingLeft(1)
	topicBarStyle = lipgloss.NewStyle().Background(lipgloss.Color("235")).Padding(0, 1)
)

// batchState tracks an in-progress IRCv3 BATCH.
type batchState struct {
	batchType string
	target    string
	messages  []client.IRCMsg
}

type model struct {
	client   *client.Client
	config   config.Config
	addr     string
	nick     string
	pass     string
	channels channelsModel
	chat     chatModel
	input    inputModel
	users    usersModel
	palette  paletteModel
	keymap   KeyMap
	width    int
	height   int
	quitting bool

	// namesBuffer accumulates nicks from RPL_NAMREPLY (353) until RPL_ENDOFNAMES (366).
	namesBuffer map[string][]string

	// availableChannels holds the full channel list from LIST.
	availableChannels []string
	// listBuffer accumulates channel names from RPL_LIST (322) until RPL_LISTEND (323).
	listBuffer []string

	// batches tracks in-progress IRCv3 BATCH spans keyed by reference ID.
	batches map[string]*batchState
	// chathistorySupported is true when the server advertised draft/chathistory.
	chathistorySupported bool

	// topics stores the current topic for each channel.
	topics map[string]string

	// notifyLines holds service response messages displayed in the input area.
	notifyLines []string
	// expectService is the IRC nick we expect NOTICE responses from (e.g. "NickServ").
	expectService string
}

// New creates a new TUI model wired to connect to the given server.
func New(addr, nick, pass string, cfg config.Config) *model {
	km := BuildKeyMap(cfg.Keys)
	m := &model{
		addr:        addr,
		nick:        nick,
		pass:        pass,
		config:      cfg,
		channels:    newChannels(),
		chat:        newChat(cfg.Timestamp),
		input:       newInput(),
		users:       newUsers(cfg.UsersWidth),
		palette:     newPalette(),
		keymap:      km,
		namesBuffer: make(map[string][]string),
		topics:      make(map[string]string),
		batches:     make(map[string]*batchState),
	}
	m.input.nick = nick
	m.chat.SetActive(serverBuffer)
	return m
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case client.ConnectedMsg:
		m.chat.AddSystemMessage(serverBuffer, "Connected to "+m.addr)
		return m, nil

	case client.IRCMsg:
		return m.handleIRC(msg)

	case client.ErrorMsg:
		m.chat.AddSystemMessage(serverBuffer, "Error: "+msg.Err.Error())
		return m, nil

	case client.DisconnectedMsg:
		detail := "Disconnected"
		if msg.Err != nil {
			detail += ": " + msg.Err.Error()
		}
		m.chat.AddSystemMessage(serverBuffer, detail)
		return m, nil
	}

	// Forward to sub-models.
	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.chat, cmd = m.chat.Update(msg)
	cmds = append(cmds, cmd)

	if m.input.Focused() {
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		return "Connecting..."
	}

	chatWidth := m.width
	if m.showUsers() {
		chatWidth = m.width - m.users.width - 1
	}

	chatView := m.chat.View()
	if tv := m.topicView(chatWidth); tv != "" {
		chatView = tv + "\n" + chatView
	}

	var middle string
	if m.showUsers() {
		middle = lipgloss.JoinHorizontal(lipgloss.Top, chatView, m.users.View())
	} else {
		middle = chatView
	}

	result := m.channels.View(m.width) + "\n" +
		middle + "\n"
	if m.notificationActive() {
		result += m.renderNotification(m.width)
	} else {
		if pv := m.palette.View(m.width); pv != "" {
			result += pv + "\n"
		}
		result += m.input.View(m.width)
	}
	return result
}

func (m *model) SetProgram(p *tea.Program) {
	var opts []client.Option
	if m.pass != "" {
		opts = append(opts, client.WithPass(m.pass))
	}
	c := client.New(func(msg any) { p.Send(msg) }, opts...)
	m.client = c
	go func() {
		if err := c.Connect(context.Background(), m.addr, m.nick); err != nil {
			p.Send(client.ErrorMsg{Err: err})
		}
	}()
}

// topicView returns the rendered topic bar for the active channel, or "" if none.
func (m *model) topicView(width int) string {
	active := m.channels.Active()
	if active == serverBuffer {
		return ""
	}
	topic := m.topics[active]
	if topic == "" {
		return ""
	}
	return topicBarStyle.Width(width).Render(topic)
}

// showUsers returns true when the users panel should be visible.
func (m *model) showUsers() bool {
	return isChannel(m.channels.Active())
}

// switchChannel activates the given channel tab and updates all dependent views.
func (m *model) switchChannel(name string) {
	m.chat.SetActive(name)
	m.users.SetActive(name)

	m.resize()
}

// send is a fire-and-forget IRC send; errors are non-fatal in the TUI context.
func (m *model) send(line string) {
	if m.client != nil {
		_ = m.client.Send(context.Background(), line)
	}
}

// sendRawWithNotify formats trailing args, sends the line, and sets up
// service response notification if the line targets a service.
func (m *model) sendRawWithNotify(text string) {
	text = formatTrailingArg(text)
	m.send(text)
	if sn := serviceNickFor(text); sn != "" {
		m.notifyLines = nil
		m.expectService = sn
	}
}

func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keyStr := msg.String()

	if m.input.Focused() {
		// Insert mode.

		// Alt+Enter / backslash-newline for soft newlines (hardcoded, not configurable).
		if msg.Type == tea.KeyEnter && (msg.Alt || strings.HasSuffix(m.input.Value(), `\`)) {
			if strings.HasSuffix(m.input.Value(), `\`) {
				m.input.TrimTrailingChar()
			}
			m.input.InsertNewline()
			m.resize()
			return m, nil
		}

		if action, ok := m.keymap.Insert[keyStr]; ok {
			return m.executeAction(action)
		}

		// Fall through to textarea update.
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.updatePalette()
		m.resize()
		return m, cmd
	}

	// Normal mode.
	if action, ok := m.keymap.Normal[keyStr]; ok {
		return m.executeAction(action)
	}
	return m, nil
}

func (m *model) executeAction(action Action) (tea.Model, tea.Cmd) {
	switch action {
	case ActionQuit:
		if m.input.Focused() && m.input.Value() != "" {
			m.input.Reset()
			m.input.Blur()
			m.palette.Hide()
			m.resize()
			return m, nil
		}
		m.quitting = true
		if m.client != nil {
			_ = m.client.Close()
		}
		return m, tea.Quit

	case ActionNextChannel:
		m.switchChannel(m.channels.Next())
		return m, nil

	case ActionPrevChannel:
		m.switchChannel(m.channels.Prev())
		return m, nil

	case ActionChat:
		m.clearNotification()
		m.input.SetCommandMode(false)
		cmd := m.input.Focus()
		return m, cmd

	case ActionCommand:
		m.clearNotification()
		m.input.SetCommandMode(true)
		cmd := m.input.Focus()
		m.updatePalette()
		return m, cmd

	case ActionCancel:
		m.clearNotification()
		m.input.Reset()
		m.input.Blur()
		m.palette.Hide()
		m.resize()
		return m, nil

	case ActionSubmit:
		if m.palette.visible {
			if m.palette.fillsLastArg() {
				if name, ok := m.palette.SelectedName(); ok {
					m.fillCompletion(name)
					return m.handleInput()
				}
			} else if cmd, ok := m.palette.Selected(); ok {
				m.input.SetValue(cmd.Name)
				m.palette.Hide()
				m.resize()
				return m.handleInput()
			}
		}
		m.palette.Hide()
		m.resize()
		return m.handleInput()

	case ActionPaletteUp:
		if m.palette.visible {
			m.palette.Prev()
			m.fillFromPalette()
			m.resize()
		}
		return m, nil

	case ActionPaletteDown:
		if m.palette.visible {
			m.palette.Next()
			m.fillFromPalette()
			m.resize()
		}
		return m, nil

	case ActionScrollUp:
		m.chat.ScrollUp()
		return m, nil

	case ActionScrollDown:
		m.chat.ScrollDown()
		return m, nil

	case ActionJoin:
		return m.enterCommandWith("join ")

	case ActionLeave:
		return m.enterCommandWith("leave ")

	case ActionDM:
		return m.enterCommandWith("dm ")

	case ActionMe:
		return m.enterCommandWith("me ")

	case ActionNick:
		return m.enterCommandWith("nick ")

	case ActionTheme:
		return m.enterCommandWith("theme ")

	case ActionSet:
		return m.enterCommandWith("set ")

	case ActionIRCQuit:
		return m.handleCommand("quit")

	case ActionRawMode:
		m.clearNotification()
		m.input.SetMode(modeRaw)
		cmd := m.input.Focus()
		m.updatePalette()
		return m, cmd
	}

	return m, nil
}

// enterCommandWith opens command input pre-filled with the given value.
func (m *model) enterCommandWith(val string) (tea.Model, tea.Cmd) {
	m.clearNotification()
	m.input.SetCommandMode(true)
	cmd := m.input.Focus()
	m.input.SetValue(val)
	m.updatePalette()
	m.resize()
	return m, cmd
}

func (m *model) updatePalette() {
	m.palette.ClearSyntaxHint()

	var cmds []Command
	if m.input.RawMode() {
		cmds = rawCommands
	} else if m.input.CommandMode() {
		cmds = commands
	} else {
		m.palette.Hide()
		m.resize()
		return
	}

	val := m.input.Value()
	fields := strings.Fields(val)

	// No args yet — show command name completions.
	if len(fields) <= 1 && !strings.HasSuffix(val, " ") {
		if m.input.RawMode() {
			m.palette.UpdateRaw(val)
		} else {
			m.palette.Update(val)
		}
		m.resize()
		return
	}

	// Find the command definition.
	cmdName := fields[0]

	// Check for service commands (NS, CS, NICKSERV, etc.) first.
	if canonical, ok := isServiceCommand(cmdName); ok {
		m.updateServicePalette(canonical, fields, val)
		return
	}

	cmd, ok := findCommand(cmds, cmdName)
	if !ok {
		m.palette.Hide()
		m.resize()
		return
	}

	// Determine which arg position we're editing.
	// fields[0] is the command, fields[1..] are completed args.
	// If the input ends with a space, we're starting a new arg.
	trailing := strings.HasSuffix(val, " ")
	argIdx := len(fields) - 1
	if trailing {
		argIdx = len(fields)
	}
	argIdx-- // adjust: fields[0] is command, so arg 0 = fields[1]
	if trailing {
		argIdx = len(fields) - 1
	}

	// Set syntax hint if available.
	if len(cmd.Syntax) > 0 {
		syntaxIdx := argIdx
		if syntaxIdx < 0 {
			syntaxIdx = 0
		}
		m.palette.SetSyntaxHint(cmd.Name, cmd.Syntax, syntaxIdx)
	}

	if len(cmd.Args) == 0 || argIdx < 0 || argIdx >= len(cmd.Args) {
		// No completable arg at this position — keep syntax visible if set.
		m.palette.UpdateCompletions("", nil)
		m.resize()
		return
	}

	// Extract the partial text being typed for the current arg.
	partial := ""
	if !trailing && len(fields) > 1 {
		partial = fields[len(fields)-1]
	}

	candidates := m.completionsFor(cmd.Args[argIdx])
	if candidates == nil {
		m.palette.UpdateCompletions("", nil)
		m.resize()
		return
	}
	m.palette.UpdateCompletions(partial, candidates)
	m.resize()
}

// updateServicePalette handles palette updates for service commands (NS, CHANSERV, etc.).
// It walks the command chain to arbitrary depth, showing subcommand or arg completions
// at each level.
func (m *model) updateServicePalette(service string, fields []string, val string) {
	cmds := serviceSubcommands[service]
	trailing := strings.HasSuffix(val, " ")
	pos := 1 // skip service name (fields[0])
	var chain []string

	for {
		// At typing position → show/filter subcommand palette.
		if pos >= len(fields) || (pos == len(fields)-1 && !trailing) {
			partial := ""
			if pos < len(fields) {
				partial = fields[pos]
			}
			m.palette.UpdateSubcommands(partial, cmds)
			m.resize()
			return
		}

		// Word at pos is complete → match it.
		cmd, ok := findCommand(cmds, fields[pos])
		if !ok {
			m.palette.Hide()
			m.resize()
			return
		}
		chain = append(chain, cmd.Name)
		pos++

		// If matched command has subcommands, descend.
		if len(cmd.Subcommands) > 0 {
			cmds = cmd.Subcommands
			continue
		}

		// Leaf command → arg completion.
		m.completeServiceArgs(cmd, fields, pos, trailing, strings.Join(chain, " "))
		return
	}
}

// completeServiceArgs handles argument completion for a leaf service subcommand.
func (m *model) completeServiceArgs(cmd Command, fields []string, argStart int, trailing bool, chainPrefix string) {
	argFields := len(fields) - argStart
	var argIdx int
	if trailing {
		argIdx = argFields
	} else {
		argIdx = argFields - 1
	}

	// Set syntax hint if available.
	if len(cmd.Syntax) > 0 {
		syntaxIdx := argIdx
		if syntaxIdx < 0 {
			syntaxIdx = 0
		}
		m.palette.SetSyntaxHint(chainPrefix, cmd.Syntax, syntaxIdx)
	}

	if len(cmd.Args) == 0 {
		// No completable args, but syntax hint may be showing.
		m.palette.UpdateCompletions("", nil)
		m.resize()
		return
	}

	if argIdx < 0 || argIdx >= len(cmd.Args) {
		// Past completable args — clear matches but keep syntax visible.
		m.palette.UpdateCompletions("", nil)
		m.resize()
		return
	}

	partial := ""
	if !trailing && len(fields) > argStart {
		partial = fields[len(fields)-1]
	}

	candidates := m.completionsFor(cmd.Args[argIdx])
	if candidates == nil {
		// No candidates for this arg type — clear matches but keep syntax.
		m.palette.UpdateCompletions("", nil)
		m.resize()
		return
	}
	m.palette.UpdateCompletions(partial, candidates)
	m.resize()
}

// findCommand looks up a command by name (or alias for herald commands).
func findCommand(cmds []Command, name string) (Command, bool) {
	lower := strings.ToLower(name)
	upper := strings.ToUpper(name)
	for _, c := range cmds {
		if c.Name == lower || c.Name == upper || strings.EqualFold(c.Name, name) {
			return c, true
		}
		for _, a := range c.Aliases {
			if strings.EqualFold(a, name) {
				return c, true
			}
		}
	}
	return Command{}, false
}

// completionsFor returns the candidate list for a given ArgType.
func (m *model) completionsFor(arg ArgType) []string {
	switch arg {
	case ArgChannel:
		// Combine available (from LIST) and joined channels.
		seen := make(map[string]bool)
		var out []string
		for _, ch := range m.availableChannels {
			if !seen[ch] {
				seen[ch] = true
				out = append(out, ch)
			}
		}
		for _, ch := range m.joinedChannels() {
			if !seen[ch] {
				seen[ch] = true
				out = append(out, ch)
			}
		}
		return out
	case ArgNick:
		return m.knownNicks()
	case ArgTarget:
		// Channels + nicks.
		var out []string
		out = append(out, m.joinedChannels()...)
		out = append(out, m.knownNicks()...)
		return out
	case ArgSetting:
		return settingNames()
	case ArgTheme:
		return config.AvailableThemes()
	}
	return nil
}

func (m *model) joinedChannels() []string {
	var out []string
	for _, t := range m.channels.Tabs() {
		if t != serverBuffer {
			out = append(out, t)
		}
	}
	return out
}

func (m *model) knownNicks() []string {
	return m.users.AllNicks()
}

func settingNames() []string {
	settings := config.AvailableSettings()
	names := make([]string, len(settings))
	for i, s := range settings {
		names[i] = s.Name
	}
	return names
}

// fillFromPalette replaces the current arg in the input with the selected palette item.
func (m *model) fillFromPalette() {
	if m.palette.fillsLastArg() {
		if name, ok := m.palette.SelectedName(); ok {
			m.replaceLastArg(name)
		}
	} else if cmd, ok := m.palette.Selected(); ok {
		m.input.SetValue(cmd.Name)
	}
}

// fillCompletion replaces the current arg with the selected completion.
func (m *model) fillCompletion(name string) {
	m.replaceLastArg(name)
	m.palette.Hide()
	m.resize()
}

// replaceLastArg replaces the last whitespace-delimited token (or appends if input ends with space).
func (m *model) replaceLastArg(name string) {
	val := m.input.Value()
	if strings.HasSuffix(val, " ") {
		m.input.SetValue(val + name)
		return
	}
	// Find the last space and replace everything after it.
	if i := strings.LastIndex(val, " "); i >= 0 {
		m.input.SetValue(val[:i+1] + name)
	} else {
		m.input.SetValue(name)
	}
}

// clearNotification removes any pending service notification.
func (m *model) clearNotification() {
	m.notifyLines = nil
	m.expectService = ""
}

// notificationActive returns true when a notification should be displayed.
func (m *model) notificationActive() bool {
	return len(m.notifyLines) > 0 && m.channels.Active() != serverBuffer && !m.input.Focused()
}

// renderNotification renders the service response notification for the input area.
func (m *model) renderNotification(width int) string {
	lines := m.notifyLines
	if len(lines) > inputMaxHeight {
		lines = lines[len(lines)-inputMaxHeight:]
	}
	return notifyStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func (m *model) handleInput() (tea.Model, tea.Cmd) {
	text := strings.TrimSpace(m.input.Value())
	mode := m.input.mode
	m.input.Reset()
	m.input.Blur()
	m.palette.Hide()
	m.resize()

	if text == "" {
		return m, nil
	}

	text = expandEnvBraces(text)

	switch mode {
	case modeCommand:
		return m.handleCommand(text)
	case modeRaw:
		m.sendRawWithNotify(text)
		return m, nil
	default:
		return m.sendChat(text)
	}
}

func (m *model) sendChat(text string) (tea.Model, tea.Cmd) {
	target := m.channels.Active()
	if target == serverBuffer {
		m.chat.AddSystemMessage(serverBuffer, "Cannot send to server buffer. Join a channel first.")
		return m, nil
	}
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		m.send("PRIVMSG " + target + " :" + line)
		m.chat.AddMessage(target, m.nick, line)

	}
	return m, nil
}

func (m *model) handleCommand(text string) (tea.Model, tea.Cmd) {
	parts := strings.SplitN(text, " ", 2)
	cmd := strings.ToUpper(resolveAlias(parts[0]))
	var args string
	if len(parts) > 1 {
		args = parts[1]
	}

	switch cmd {
	case "JOIN":
		if args == "" {
			m.chat.AddSystemMessage(m.channels.Active(), "Usage: :join <channel>")
			return m, nil
		}
		if !isChannel(args) {
			args = "#" + args
		}
		m.send("JOIN " + args)

	case "LEAVE":
		target := args
		if target == "" {
			target = m.channels.Active()
		}
		if target == serverBuffer {
			return m, nil
		}
		if isChannel(target) {
			m.send("PART " + target)
		} else {
			// DM/query buffer — just close the tab locally.
			m.chat.ClearHistory(target)
			m.channels.Remove(target)
			m.switchChannel(m.channels.Active())
		}

	case "DM":
		msgParts := strings.SplitN(args, " ", 2)
		if len(msgParts) < 2 {
			m.chat.AddSystemMessage(m.channels.Active(), "Usage: :msg <target> <message>")
			return m, nil
		}
		target := msgParts[0]
		m.send("PRIVMSG " + target + " :" + msgParts[1])
		m.channels.Add(target)
		m.switchChannel(m.channels.SetActive(target))
		m.chat.AddMessage(target, m.nick, msgParts[1])

	case "ME":
		target := m.channels.Active()
		if target == serverBuffer {
			return m, nil
		}
		m.send("PRIVMSG " + target + " :\x01ACTION " + args + "\x01")
		m.chat.AddAction(target, m.nick, args)

	case "NICK":
		if args == "" {
			m.chat.AddSystemMessage(m.channels.Active(), "Usage: :nick <nickname>")
			return m, nil
		}
		m.send("NICK " + args)

	case "QUIT":
		m.quitting = true
		quitMsg := "Leaving"
		if args != "" {
			quitMsg = args
		}
		m.send("QUIT :" + quitMsg)
		if m.client != nil {
			_ = m.client.Close()
		}
		return m, tea.Quit

	case "SET":
		buf := m.channels.Active()
		if args == "" {
			for _, s := range config.AvailableSettings() {
				val := m.config.Get(s.Name)
				m.chat.AddSystemMessage(buf, fmt.Sprintf("%s = %s  (%s)", s.Name, val, s.Desc))
			}
			return m, nil
		}
		setParts := strings.SplitN(args, " ", 2)
		key := setParts[0]
		if len(setParts) == 1 {
			val := m.config.Get(key)
			if val == "" {
				m.chat.AddSystemMessage(buf, "Unknown setting: "+key)
			} else {
				m.chat.AddSystemMessage(buf, key+" = "+val)
			}
			return m, nil
		}
		value := setParts[1]
		if err := m.config.Set(key, value); err != nil {
			m.chat.AddSystemMessage(buf, "Error: "+err.Error())
			return m, nil
		}
		m.applySetting(key)
		if err := m.config.Save(); err != nil {
			m.chat.AddSystemMessage(buf, "Setting applied but failed to save: "+err.Error())
			return m, nil
		}
		m.chat.AddSystemMessage(buf, key+" = "+value)

	case "THEME":
		buf := m.channels.Active()
		if args == "" {
			available := config.AvailableThemes()
			if len(available) == 0 {
				m.chat.AddSystemMessage(buf, "No themes found in "+config.ThemesDir())
			} else {
				m.chat.AddSystemMessage(buf, "Available themes: "+strings.Join(available, ", "))
			}
			current := m.config.Theme
			if current == "" {
				current = "(default)"
			}
			m.chat.AddSystemMessage(buf, "Current theme: "+current)
			return m, nil
		}
		theme, err := config.LoadTheme(args)
		if err != nil {
			m.chat.AddSystemMessage(buf, "Error: "+err.Error())
			return m, nil
		}
		ApplyTheme(theme)
		m.config.Theme = args
		if err := m.config.Save(); err != nil {
			m.chat.AddSystemMessage(buf, "Theme applied but failed to save: "+err.Error())
			return m, nil
		}
		m.chat.AddSystemMessage(buf, "Theme switched to "+args)
		m.chat.refreshViewport()

	case "TOPIC":
		target := m.channels.Active()
		if target == serverBuffer || !isChannel(target) {
			m.chat.AddSystemMessage(m.channels.Active(), "Join a channel first")
			return m, nil
		}
		if args == "" {
			// Show current topic.
			if topic, ok := m.topics[target]; ok && topic != "" {
				m.chat.AddSystemMessage(target, "Topic: "+topic)
			} else {
				m.chat.AddSystemMessage(target, "No topic set")
			}
			return m, nil
		}
		m.send("TOPIC " + target + " :" + args)

	case "TEST_HISTORY":
		m.testHistory()

	default:
		// Send unknown commands as raw IRC.
		m.sendRawWithNotify(text)
	}
	return m, nil
}

// applySetting propagates a config change to the running TUI.
func (m *model) applySetting(key string) {
	switch key {
	case "timestamp":
		m.chat.timestampFormat = m.config.Timestamp
		m.chat.refreshViewport()
	case "users_width":
		m.users.width = m.config.UsersWidth
		m.resize()
	}
}

func (m *model) handleIRC(msg client.IRCMsg) (tea.Model, tea.Cmd) {
	// Accumulate messages that belong to an in-progress batch.
	if ok, batchTag := msg.GetTag("batch"); ok {
		if batch, exists := m.batches[batchTag]; exists {
			batch.messages = append(batch.messages, msg)
			return m, nil
		}
	}

	switch msg.Command {
	case "BATCH":
		if len(msg.Params) < 1 {
			return m, nil
		}
		ref := msg.Params[0]
		if strings.HasPrefix(ref, "+") {
			refID := ref[1:]
			bs := &batchState{}
			if len(msg.Params) >= 2 {
				bs.batchType = msg.Params[1]
			}
			if len(msg.Params) >= 3 {
				bs.target = msg.Params[2]
			}
			m.batches[refID] = bs
		} else if strings.HasPrefix(ref, "-") {
			refID := ref[1:]
			if batch, ok := m.batches[refID]; ok {
				delete(m.batches, refID)
				m.finalizeBatch(batch)
			}
		}
		return m, nil

	case "PING":
		arg := ""
		if len(msg.Params) > 0 {
			arg = msg.Params[0]
		}
		m.send("PONG :" + arg)

	case "PRIVMSG":
		if len(msg.Params) < 2 {
			return m, nil
		}
		target := msg.Params[0]
		text := msg.Params[1]
		nick := parseNick(msg.Nick())

		// Skip own messages — already displayed by the send path.
		if strings.EqualFold(nick, m.nick) {
			return m, nil
		}

		// CTCP ACTION.
		if strings.HasPrefix(text, "\x01ACTION ") && strings.HasSuffix(text, "\x01") {
			action := text[8 : len(text)-1]
			// DM actions show in a query buffer named after the nick.
			if !isChannel(target) {
				target = nick
			}
			m.channels.Add(target)
			stripped := format.Strip(action)
			m.chat.AddAction(target, nick, stripped)
			return m, nil
		}

		// Private message goes to a query buffer.
		if !isChannel(target) {
			target = nick
		}
		m.channels.Add(target)
		stripped := format.Strip(text)
		m.chat.AddMessage(target, nick, stripped)

	case "NOTICE":
		if len(msg.Params) < 2 {
			return m, nil
		}
		nick := parseNick(msg.Nick())
		text := msg.Params[1]
		target := serverBuffer
		if len(msg.Params) > 0 && isChannel(msg.Params[0]) {
			target = msg.Params[0]
			m.channels.Add(target)
		}
		stripped := format.Strip(text)

		// Accumulate service responses for the input-area notification.
		if m.expectService != "" && strings.EqualFold(nick, m.expectService) {
			m.notifyLines = append(m.notifyLines, stripped)
			m.resize()
		}

		var content string
		if strings.EqualFold(nick, "HistServ") {
			content = stripped
		} else {
			content = fmt.Sprintf("[%s] %s", nick, stripped)
		}
		m.chat.AddSystemMessage(target, content)

	case "JOIN":
		if len(msg.Params) < 1 {
			return m, nil
		}
		channel := msg.Params[0]
		nick := parseNick(msg.Nick())
		if strings.EqualFold(nick, m.nick) {
			m.channels.Add(channel)
			m.switchChannel(m.channels.SetActive(channel))
			if m.chathistorySupported {
				m.send(fmt.Sprintf("CHATHISTORY LATEST %s * %d", channel, m.config.HistoryLimit))
			}
		} else {
			m.users.AddMember(channel, nick)
			joinContent := nick + " has joined " + channel
			m.chat.AddSystemMessage(channel, joinContent)
		}

	case "PART":
		if len(msg.Params) < 1 {
			return m, nil
		}
		channel := msg.Params[0]
		nick := parseNick(msg.Nick())
		reason := ""
		if len(msg.Params) > 1 {
			reason = " (" + msg.Params[1] + ")"
		}
		if strings.EqualFold(nick, m.nick) {
			m.chat.ClearHistory(channel)
			delete(m.topics, channel)
			m.channels.Remove(channel)
			m.switchChannel(m.channels.Active())
		} else {
			m.users.RemoveMember(channel, nick)
			partContent := nick + " has left " + channel + reason
			m.chat.AddSystemMessage(channel, partContent)
		}

	case "QUIT":
		nick := parseNick(msg.Nick())
		reason := ""
		if len(msg.Params) > 0 {
			reason = " (" + msg.Params[0] + ")"
		}
		// Show quit in every channel the user was in, then remove them.
		for ch, nicks := range m.users.members {
			for _, n := range nicks {
				if strings.EqualFold(stripPrefix(n), nick) {
					quitContent := nick + " has quit" + reason
					m.chat.AddSystemMessage(ch, quitContent)
					break
				}
			}
		}
		m.users.RemoveMemberAll(nick)

	case "NICK":
		if len(msg.Params) < 1 {
			return m, nil
		}
		oldNick := parseNick(msg.Nick())
		newNick := msg.Params[0]
		if strings.EqualFold(oldNick, m.nick) {
			m.nick = newNick
			m.input.nick = newNick
			if m.client != nil {
				m.client.SetNick(newNick)
			}
		}
		m.users.RenameMember(oldNick, newNick)
		// Show nick change in every channel the user is in.
		nickContent := oldNick + " is now known as " + newNick
		for ch, nicks := range m.users.members {
			for _, n := range nicks {
				if strings.EqualFold(stripPrefix(n), newNick) {
					m.chat.AddSystemMessage(ch, nickContent)
					break
				}
			}
		}
		// Fallback: if user isn't in any tracked channel, show in active buffer.
		if len(m.users.members) == 0 {
			active := m.channels.Active()
			m.chat.AddSystemMessage(active, nickContent)
		}

	case "TOPIC":
		if len(msg.Params) < 1 {
			return m, nil
		}
		channel := msg.Params[0]
		nick := parseNick(msg.Nick())
		if len(msg.Params) >= 2 && msg.Params[1] != "" {
			topic := format.Strip(msg.Params[1])
			m.topics[channel] = topic
			m.chat.AddSystemMessage(channel, nick+" changed the topic: "+topic)
		} else {
			delete(m.topics, channel)
			m.chat.AddSystemMessage(channel, nick+" cleared the topic")
		}
		m.resize()

	case "001": // RPL_WELCOME
		if len(msg.Params) > 1 {
			m.chat.AddSystemMessage(serverBuffer, msg.Params[1])
		}
		m.channels.SetDisplay(serverBuffer, msg.Source)
		m.chathistorySupported = m.client.HasCap("chathistory") || m.client.HasCap("draft/chathistory")
		m.send("LIST")

	case "002", "003", "004": // Server info numerics.
		if len(msg.Params) > 1 {
			m.chat.AddSystemMessage(serverBuffer, strings.Join(msg.Params[1:], " "))
		}

	case "005": // RPL_ISUPPORT
		if len(msg.Params) > 1 {
			m.chat.AddSystemMessage(serverBuffer, strings.Join(msg.Params[1:], " "))
		}

	case "322": // RPL_LIST
		if len(msg.Params) >= 2 {
			m.listBuffer = append(m.listBuffer, msg.Params[1])
		}

	case "323": // RPL_LISTEND
		m.availableChannels = m.listBuffer
		m.listBuffer = nil

	case "332": // RPL_TOPIC
		if len(msg.Params) >= 3 {
			channel := msg.Params[1]
			topic := msg.Params[2]
			m.topics[channel] = format.Strip(topic)
			m.chat.AddSystemMessage(channel, "Topic: "+format.Strip(topic))
			m.resize()
		}

	case "353": // RPL_NAMREPLY
		if len(msg.Params) >= 4 {
			channel := msg.Params[2]
			names := strings.Fields(msg.Params[3])
			m.namesBuffer[channel] = append(m.namesBuffer[channel], names...)
		}

	case "366": // RPL_ENDOFNAMES
		if len(msg.Params) >= 2 {
			channel := msg.Params[1]
			if nicks, ok := m.namesBuffer[channel]; ok {
				m.users.SetMembers(channel, nicks)
				delete(m.namesBuffer, channel)

			}
		}

	case "372", "375", "376": // MOTD lines.
		if len(msg.Params) > 1 {
			m.chat.AddSystemMessage(serverBuffer, msg.Params[len(msg.Params)-1])
		}

	case "433": // ERR_NICKNAMEINUSE
		m.chat.AddSystemMessage(serverBuffer, "Nickname already in use")
		// Try with underscore.
		m.nick += "_"
		m.input.nick = m.nick
		m.send("NICK " + m.nick)
		if m.client != nil {
			m.client.SetNick(m.nick)
		}

	default:
		// Show unhandled numerics/commands in server buffer.
		if len(msg.Params) > 1 {
			m.chat.AddSystemMessage(serverBuffer, fmt.Sprintf("[%s] %s", msg.Command, strings.Join(msg.Params[1:], " ")))
		}
	}

	return m, nil
}

func (m *model) resize() {
	m.input.SetWidth(m.width)
	channelsHeight := 2 // tab bar + border

	chatWidth := m.width
	if m.showUsers() {
		chatWidth = m.width - m.users.width - 1
		if chatWidth < 1 {
			chatWidth = 1
		}
	}

	topicHeight := 0
	if tv := m.topicView(chatWidth); tv != "" {
		topicHeight = lipgloss.Height(tv) + 1 // +1 for newline
	}

	var middleHeight int
	if m.notificationActive() {
		notifyH := min(len(m.notifyLines), inputMaxHeight)
		middleHeight = m.height - channelsHeight - notifyH
	} else {
		inputHeight := m.input.LineCount()
		paletteHeight := m.palette.Height(m.width)
		middleHeight = m.height - channelsHeight - inputHeight - paletteHeight
	}

	chatHeight := middleHeight - topicHeight
	if chatHeight < 1 {
		chatHeight = 1
	}

	m.chat.SetSize(chatWidth, chatHeight)
	if m.showUsers() {
		usersHeight := middleHeight
		if usersHeight < 1 {
			usersHeight = 1
		}
		m.users.SetSize(m.users.width, usersHeight)
	}
}

func parseNick(prefix string) string {
	if i := strings.Index(prefix, "!"); i >= 0 {
		return prefix[:i]
	}
	return prefix
}

// expandEnvBraces replaces ${VAR} with the environment variable value.
// Only the braced form is expanded; bare $VAR is left as-is.
func expandEnvBraces(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if i+2 < len(s) && s[i] == '$' && s[i+1] == '{' {
			if end := strings.IndexByte(s[i+2:], '}'); end >= 0 {
				key := s[i+2 : i+2+end]
				b.WriteString(os.Getenv(key))
				i += 2 + end // skip past '}'
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func isChannel(s string) bool {
	return len(s) > 0 && (s[0] == '#' || s[0] == '&')
}

// resolveRawCommand resolves fields to a Command definition by checking
// service subcommand chains first, then rawCommands.
// Returns the matched command and the number of leading fields consumed.
func resolveRawCommand(fields []string) (*Command, int) {
	if len(fields) == 0 {
		return nil, 0
	}

	// Check service commands first (they need chain walking).
	if canonical, ok := isServiceCommand(fields[0]); ok {
		cmds := serviceSubcommands[canonical]
		pos := 1
		for pos < len(fields) {
			cmd, ok := findCommand(cmds, fields[pos])
			if !ok {
				return nil, 0
			}
			pos++
			if len(cmd.Subcommands) > 0 {
				cmds = cmd.Subcommands
				continue
			}
			return &cmd, pos
		}
		return nil, 0
	}

	// Check rawCommands.
	if cmd, ok := findCommand(rawCommands, fields[0]); ok {
		return &cmd, 1
	}

	return nil, 0
}

// formatTrailingArg auto-joins excess arguments into a colon-prefixed trailing
// parameter when the last Syntax token is a string type (:string).
func formatTrailingArg(line string) string {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return line
	}

	cmd, prefixCount := resolveRawCommand(fields)
	if cmd == nil || len(cmd.Syntax) == 0 {
		return line
	}

	lastToken := cmd.Syntax[len(cmd.Syntax)-1]
	if !strings.Contains(lastToken, ":string") {
		return line
	}

	expected := prefixCount + len(cmd.Syntax)
	if len(fields) <= expected {
		return line
	}

	// Join everything from the last syntax token's position onward.
	trailingStart := expected - 1
	trailing := strings.Join(fields[trailingStart:], " ")
	if !strings.HasPrefix(trailing, ":") {
		trailing = ":" + trailing
	}

	return strings.Join(fields[:trailingStart], " ") + " " + trailing
}

// finalizeBatch dispatches a completed batch to the appropriate handler.
func (m *model) finalizeBatch(batch *batchState) {
	switch batch.batchType {
	case "chathistory":
		m.finalizeChatHistory(batch)
	}
}

// finalizeChatHistory converts batch messages into chatLines and prepends them.
func (m *model) finalizeChatHistory(batch *batchState) {
	var lines []chatLine
	for _, msg := range batch.messages {
		nick := parseNick(msg.Nick())
		t := parseServerTime(msg)

		switch msg.Command {
		case "PRIVMSG":
			if len(msg.Params) < 2 {
				continue
			}
			text := msg.Params[1]

			// CTCP ACTION.
			if strings.HasPrefix(text, "\x01ACTION ") && strings.HasSuffix(text, "\x01") {
				content := format.Strip(text[8 : len(text)-1])
				lines = append(lines, chatLine{nick: nick, content: content, time: t, action: true})
				continue
			}

			lines = append(lines, chatLine{nick: nick, content: format.Strip(text), time: t})

		case "NOTICE":
			if len(msg.Params) < 2 {
				continue
			}
			stripped := format.Strip(msg.Params[1])
			var content string
			if strings.EqualFold(nick, "HistServ") {
				content = stripped
			} else {
				content = fmt.Sprintf("[%s] %s", nick, stripped)
			}
			lines = append(lines, chatLine{content: content, time: t, system: true})

		case "JOIN":
			if len(msg.Params) < 1 {
				continue
			}
			lines = append(lines, chatLine{content: nick + " has joined " + msg.Params[0], time: t, system: true})

		case "PART":
			channel := batch.target
			if len(msg.Params) >= 1 {
				channel = msg.Params[0]
			}
			reason := ""
			if len(msg.Params) > 1 {
				reason = " (" + msg.Params[1] + ")"
			}
			lines = append(lines, chatLine{content: nick + " has left " + channel + reason, time: t, system: true})

		case "QUIT":
			reason := ""
			if len(msg.Params) > 0 {
				reason = " (" + msg.Params[0] + ")"
			}
			lines = append(lines, chatLine{content: nick + " has quit" + reason, time: t, system: true})

		case "NICK":
			if len(msg.Params) < 1 {
				continue
			}
			lines = append(lines, chatLine{content: nick + " is now known as " + msg.Params[0], time: t, system: true})
		}
	}

	if len(lines) > 0 {
		m.chat.PrependMessages(batch.target, lines)
	}
}

// testHistory injects fake chat history to preview day separators and formatting.
func (m *model) testHistory() {
	target := m.channels.Active()
	if target == serverBuffer {
		m.chat.AddSystemMessage(serverBuffer, "Join a channel first.")
		return
	}

	now := time.Now()
	threeDaysAgo := now.AddDate(0, 0, -3)
	yesterday := now.AddDate(0, 0, -1)

	lines := []chatLine{
		{nick: "alice", content: "anyone around?", time: threeDaysAgo.Add(10 * time.Hour)},
		{nick: "bob", content: "hey alice!", time: threeDaysAgo.Add(10*time.Hour + 5*time.Minute)},
		{nick: "alice", content: "working on the new feature branch", time: threeDaysAgo.Add(10*time.Hour + 12*time.Minute)},
		{nick: "charlie", content: "waves", time: yesterday.Add(9 * time.Hour), action: true},
		{nick: "bob", content: "morning charlie", time: yesterday.Add(9*time.Hour + 1*time.Minute)},
		{nick: "alice", content: "pushed the PR, take a look when you get a chance", time: yesterday.Add(14 * time.Hour)},
		{nick: "charlie", content: "LGTM, just left a small comment", time: yesterday.Add(15*time.Hour + 30*time.Minute)},
		{nick: "bob", content: "let's ship it today", time: now.Add(-2 * time.Hour)},
		{nick: "alice", content: "merged!", time: now.Add(-1 * time.Hour)},
	}

	m.chat.PrependMessages(target, lines)
}

// parseServerTime extracts the server-time tag from an IRC message.
func parseServerTime(msg client.IRCMsg) time.Time {
	ok, val := msg.GetTag("time")
	if !ok {
		return time.Now()
	}
	if t, err := time.Parse(time.RFC3339Nano, val); err == nil {
		return t.Local()
	}
	if t, err := time.Parse(time.RFC3339, val); err == nil {
		return t.Local()
	}
	return time.Now()
}
