package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/medievalsoftware/herald/internal/client"
	"github.com/medievalsoftware/herald/internal/config"
	"github.com/medievalsoftware/herald/internal/format"
)

const serverBuffer = "*server*"

type model struct {
	client   *client.Client
	config   config.Config
	addr     string
	nick     string
	channels channelsModel
	chat     chatModel
	input    inputModel
	status   statusModel
	users    usersModel
	palette  paletteModel
	width    int
	height   int
	quitting bool

	// namesBuffer accumulates nicks from RPL_NAMREPLY (353) until RPL_ENDOFNAMES (366).
	namesBuffer map[string][]string

	// availableChannels holds the full channel list from LIST.
	availableChannels []string
	// listBuffer accumulates channel names from RPL_LIST (322) until RPL_LISTEND (323).
	listBuffer []string
}

// New creates a new TUI model wired to connect to the given server.
func New(addr, nick string, cfg config.Config) *model {
	m := &model{
		addr:        addr,
		nick:        nick,
		config:      cfg,
		channels:    newChannels(),
		chat:        newChat(cfg.Timestamp),
		input:       newInput(),
		status:      newStatus(),
		users:       newUsers(),
		palette:     newPalette(),
		namesBuffer: make(map[string][]string),
	}
	m.status.server = addr
	m.status.nick = nick
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

	middle := lipgloss.JoinHorizontal(lipgloss.Top, m.chat.View(), m.users.View())

	result := m.channels.View(m.width) + "\n" +
		middle + "\n" +
		m.status.View(m.width) + "\n"
	if pv := m.palette.View(m.width); pv != "" {
		result += pv + "\n"
	}
	result += m.input.View(m.width)
	return result
}

func (m *model) SetProgram(p *tea.Program) {
	c := client.New(func(msg any) { p.Send(msg) })
	m.client = c
	go func() {
		if err := c.Connect(context.Background(), m.addr, m.nick); err != nil {
			p.Send(client.ErrorMsg{Err: err})
		}
	}()
}

// send is a fire-and-forget IRC send; errors are non-fatal in the TUI context.
func (m *model) send(line string) {
	if m.client != nil {
		_ = m.client.Send(context.Background(), line)
	}
}

func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global bindings (both modes).
	switch msg.Type {
	case tea.KeyCtrlC:
		if m.input.Focused() && m.input.Value() != "" {
			m.input.Reset()
			m.input.ResetHistory()
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

	case tea.KeyCtrlN:
		name := m.channels.Next()
		m.chat.SetActive(name)
		m.users.SetActive(name)
		m.status.users = m.users.Count()
		return m, nil

	case tea.KeyCtrlP:
		name := m.channels.Prev()
		m.chat.SetActive(name)
		m.users.SetActive(name)
		m.status.users = m.users.Count()
		return m, nil
	}

	if m.input.Focused() {
		// Insert mode.
		switch msg.Type {
		case tea.KeyUp:
			if m.palette.visible {
				m.palette.Prev()
				return m, nil
			}
			m.input.HistoryPrev()
			m.updatePalette()
			return m, nil
		case tea.KeyDown:
			if m.palette.visible {
				m.palette.Next()
				return m, nil
			}
			m.input.HistoryNext()
			m.updatePalette()
			return m, nil
		case tea.KeyTab:
			if m.palette.visible {
				if m.palette.completionMode {
					if name, ok := m.palette.SelectedName(); ok {
						m.fillCompletion(name)
					}
				} else if cmd, ok := m.palette.Selected(); ok {
					m.input.SetValue(cmd.Name + " ")
					m.palette.Hide()
					m.resize()
				}
				return m, nil
			}
		case tea.KeyEnter:
			if msg.Alt || strings.HasSuffix(m.input.Value(), `\`) {
				if strings.HasSuffix(m.input.Value(), `\`) {
					m.input.TrimTrailingChar()
				}
				m.input.InsertNewline()
				m.resize()
				return m, nil
			}
			if m.palette.visible {
				if m.palette.completionMode {
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
		case tea.KeyEscape:
			m.input.Reset()
			m.input.ResetHistory()
			m.input.Blur()
			m.palette.Hide()
			m.resize()
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.updatePalette()
		m.resize()
		return m, cmd
	}

	// Normal mode.
	switch msg.Type {
	case tea.KeyEnter:
		m.input.SetCommandMode(false)
		cmd := m.input.Focus()
		return m, cmd
	case tea.KeyRunes:
		if msg.String() == ":" {
			m.input.SetCommandMode(true)
			cmd := m.input.Focus()
			m.updatePalette()
			return m, cmd
		}
	}
	return m, nil
}

func (m *model) updatePalette() {
	if !m.input.CommandMode() {
		m.palette.Hide()
		m.resize()
		return
	}

	val := m.input.Value()
	if !strings.Contains(val, " ") {
		// Command name completion.
		m.palette.Update(val)
		m.resize()
		return
	}

	parts := strings.SplitN(val, " ", 2)
	cmd := resolveAlias(parts[0])
	partial := parts[1]

	// Don't complete if arg already has a space (second arg started).
	if strings.Contains(partial, " ") {
		m.palette.Hide()
		m.resize()
		return
	}

	switch cmd {
	case "join":
		m.palette.UpdateCompletions(partial, m.availableChannels)
	case "leave":
		m.palette.UpdateCompletions(partial, m.joinedChannels())
	case "msg":
		m.palette.UpdateCompletions(partial, m.knownNicks())
	default:
		m.palette.Hide()
	}
	m.resize()
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

// fillCompletion replaces the argument portion of input with the selected completion.
func (m *model) fillCompletion(name string) {
	val := m.input.Value()
	parts := strings.SplitN(val, " ", 2)
	m.input.SetValue(parts[0] + " " + name)
	m.palette.Hide()
	m.resize()
}

func (m *model) handleInput() (tea.Model, tea.Cmd) {
	text := strings.TrimSpace(m.input.Value())
	commandMode := m.input.CommandMode()
	m.input.PushHistory(text)
	m.input.Reset()
	m.input.Blur()
	m.palette.Hide()
	m.resize()

	if text == "" {
		return m, nil
	}

	if commandMode {
		return m.handleCommand(text)
	}

	return m.sendChat(text)
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
		m.send("PART " + target)

	case "MSG":
		msgParts := strings.SplitN(args, " ", 2)
		if len(msgParts) < 2 {
			m.chat.AddSystemMessage(m.channels.Active(), "Usage: :msg <target> <message>")
			return m, nil
		}
		m.send("PRIVMSG " + msgParts[0] + " :" + msgParts[1])
		m.chat.AddMessage(msgParts[0], m.nick, msgParts[1])

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

	case "RAW":
		m.send(args)

	default:
		// Send unknown commands as raw IRC.
		m.send(text)
	}
	return m, nil
}

func (m *model) handleIRC(msg client.IRCMsg) (tea.Model, tea.Cmd) {
	switch msg.Command {
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

		// CTCP ACTION.
		if strings.HasPrefix(text, "\x01ACTION ") && strings.HasSuffix(text, "\x01") {
			action := text[8 : len(text)-1]
			// DM actions show in a query buffer named after the nick.
			if !isChannel(target) {
				target = nick
			}
			m.channels.Add(target)
			m.chat.AddAction(target, nick, format.Strip(action))
			return m, nil
		}

		// Private message goes to a query buffer.
		if !isChannel(target) {
			target = nick
		}
		m.channels.Add(target)
		m.chat.AddMessage(target, nick, format.Strip(text))

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
		m.chat.AddSystemMessage(target, fmt.Sprintf("[%s] %s", nick, format.Strip(text)))

	case "JOIN":
		if len(msg.Params) < 1 {
			return m, nil
		}
		channel := msg.Params[0]
		nick := parseNick(msg.Nick())
		if strings.EqualFold(nick, m.nick) {
			m.channels.Add(channel)
			name := m.channels.SetActive(channel)
			m.chat.SetActive(name)
			m.users.SetActive(name)
		} else {
			m.users.AddMember(channel, nick)
		}
		m.chat.AddSystemMessage(channel, nick+" has joined "+channel)
		m.status.users = m.users.Count()

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
			m.channels.Remove(channel)
			active := m.channels.Active()
			m.chat.SetActive(active)
			m.users.SetActive(active)
		} else {
			m.users.RemoveMember(channel, nick)
			m.chat.AddSystemMessage(channel, nick+" has left "+channel+reason)
		}
		m.status.users = m.users.Count()

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
					m.chat.AddSystemMessage(ch, nick+" has quit"+reason)
					break
				}
			}
		}
		m.users.RemoveMemberAll(nick)
		m.status.users = m.users.Count()

	case "NICK":
		if len(msg.Params) < 1 {
			return m, nil
		}
		oldNick := parseNick(msg.Nick())
		newNick := msg.Params[0]
		if strings.EqualFold(oldNick, m.nick) {
			m.nick = newNick
			m.status.nick = newNick
			if m.client != nil {
				m.client.SetNick(newNick)
			}
		}
		m.users.RenameMember(oldNick, newNick)
		// Show nick change in every channel the user is in.
		for ch, nicks := range m.users.members {
			for _, n := range nicks {
				if strings.EqualFold(stripPrefix(n), newNick) {
					m.chat.AddSystemMessage(ch, oldNick+" is now known as "+newNick)
					break
				}
			}
		}
		// Fallback: if user isn't in any tracked channel, show in active buffer.
		if len(m.users.members) == 0 {
			m.chat.AddSystemMessage(m.channels.Active(), oldNick+" is now known as "+newNick)
		}

	case "001": // RPL_WELCOME
		if len(msg.Params) > 1 {
			m.chat.AddSystemMessage(serverBuffer, msg.Params[1])
		}
		m.status.server = msg.Source
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
			m.chat.AddSystemMessage(channel, "Topic: "+format.Strip(topic))
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
				m.status.users = m.users.Count()
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
		m.status.nick = m.nick
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
	channelsHeight := 2 // tab bar + border
	statusHeight := 1
	inputHeight := 1 + m.input.LineCount() // border + textarea lines
	paletteHeight := m.palette.Height()
	chatHeight := m.height - channelsHeight - statusHeight - inputHeight - paletteHeight
	if chatHeight < 1 {
		chatHeight = 1
	}

	// Users panel gets fixed width; the border adds 1 char.
	panelWidth := usersWidth + 1
	chatWidth := m.width - panelWidth
	if chatWidth < 1 {
		chatWidth = 1
	}

	m.chat.SetSize(chatWidth, chatHeight)
	m.users.SetSize(usersWidth, chatHeight)
}

func parseNick(prefix string) string {
	if i := strings.Index(prefix, "!"); i >= 0 {
		return prefix[:i]
	}
	return prefix
}

func isChannel(s string) bool {
	return len(s) > 0 && (s[0] == '#' || s[0] == '&')
}
