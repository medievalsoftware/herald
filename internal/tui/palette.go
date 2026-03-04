package tui

import (
	"sort"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

// ArgType identifies what kind of completion a command argument expects.
type ArgType int

const (
	ArgChannel ArgType = iota + 1 // complete from available/joined channels
	ArgNick                       // complete from known nicks
	ArgTarget                     // channel or nick
	ArgSetting                    // complete from available settings
	ArgTheme                      // complete from available themes
)

// Command describes an IRC command available in the palette.
type Command struct {
	Name        string
	Aliases     []string
	Desc        string
	Args        []ArgType // completion types for each positional argument
	Syntax      []string  // e.g. ["<#channel:channel>", "[code:string]"]
	Subcommands []Command // nested sub-subcommands
}

var commands = []Command{
	{Name: "join", Aliases: []string{"j"}, Desc: "Join a channel", Args: []ArgType{ArgChannel}, Syntax: []string{"<channel:channel>"}},
	{Name: "leave", Aliases: []string{"part"}, Desc: "Leave current channel", Args: []ArgType{ArgChannel}, Syntax: []string{"[channel:channel]"}},
	{Name: "msg", Aliases: []string{"m", "query"}, Desc: "Send a direct message", Args: []ArgType{ArgNick}, Syntax: []string{"<nick:nick>", "<message:string>"}},
	{Name: "open", Aliases: []string{"o"}, Desc: "Open a channel or nick", Args: []ArgType{ArgTarget}, Syntax: []string{"<target:channel|nick>"}},
	{Name: "me", Aliases: []string{"action"}, Desc: "Send an action", Syntax: []string{"<action:string>"}},
	{Name: "nick", Desc: "Change nickname", Syntax: []string{"<nickname:string>"}},
	{Name: "quit", Aliases: []string{"exit", "q", "q!"}, Desc: "Disconnect from server", Syntax: []string{"[reason:string]"}},
	{Name: "set", Desc: "Change a setting", Args: []ArgType{ArgSetting}, Syntax: []string{"<setting:setting>", "[value:string]"}},
	{Name: "theme", Desc: "Switch color theme", Args: []ArgType{ArgTheme}, Syntax: []string{"<name:theme>"}},
	{Name: "topic", Desc: "Set or view channel topic", Syntax: []string{"[topic:string]"}},
}

var rawCommands = []Command{
	{Name: "JOIN", Desc: "JOIN <channel>{,<channel>} [<key>{,<key>}]", Args: []ArgType{ArgChannel}, Syntax: []string{"<channel:channel>", "[key:string]"}},
	{Name: "PART", Desc: "PART <channel>{,<channel>} [<reason>]", Args: []ArgType{ArgChannel}, Syntax: []string{"<channel:channel>", "[reason:string]"}},
	{Name: "PRIVMSG", Desc: "PRIVMSG <target> :<message>", Args: []ArgType{ArgTarget}, Syntax: []string{"<target:channel|nick>", ":<message:string>"}},
	{Name: "NOTICE", Desc: "NOTICE <target> :<message>", Args: []ArgType{ArgTarget}, Syntax: []string{"<target:channel|nick>", ":<message:string>"}},
	{Name: "NICK", Desc: "NICK <nickname>", Syntax: []string{"<nickname:string>"}},
	{Name: "QUIT", Desc: "QUIT [<reason>]", Syntax: []string{"[reason:string]"}},
	{Name: "MODE", Desc: "MODE <target> [<modestring> [<args>...]]", Args: []ArgType{ArgTarget}, Syntax: []string{"<target:channel|nick>", "[modestring:string]", "[args:string]"}},
	{Name: "TOPIC", Desc: "TOPIC <channel> [<topic>]", Args: []ArgType{ArgChannel}, Syntax: []string{"<channel:channel>", "[topic:string]"}},
	{Name: "KICK", Desc: "KICK <channel> <user> [<comment>]", Args: []ArgType{ArgChannel, ArgNick}, Syntax: []string{"<channel:channel>", "<user:nick>", "[comment:string]"}},
	{Name: "INVITE", Desc: "INVITE <nickname> <channel>", Args: []ArgType{ArgNick, ArgChannel}, Syntax: []string{"<nickname:nick>", "<channel:channel>"}},
	{Name: "WHO", Desc: "WHO [<mask>]", Syntax: []string{"[mask:string]"}},
	{Name: "WHOIS", Desc: "WHOIS <nick>{,<nick>}", Args: []ArgType{ArgNick}, Syntax: []string{"<nick:nick>"}},
	{Name: "LIST", Desc: "LIST [<channel>{,<channel>}]", Args: []ArgType{ArgChannel}, Syntax: []string{"[channel:channel]"}},
	{Name: "NAMES", Desc: "NAMES [<channel>{,<channel>}]", Args: []ArgType{ArgChannel}, Syntax: []string{"[channel:channel]"}},
	{Name: "MOTD", Desc: "MOTD"},
	{Name: "OPER", Desc: "OPER <name> <password>", Syntax: []string{"<name:string>", "<password:string>"}},
	{Name: "KILL", Desc: "KILL <nickname> <reason>", Args: []ArgType{ArgNick}, Syntax: []string{"<nickname:nick>", "<reason:string>"}},
	{Name: "SAMODE", Desc: "SAMODE <target> [<modestring> [<args>...]]", Args: []ArgType{ArgTarget}, Syntax: []string{"<target:channel|nick>", "[modestring:string]", "[args:string]"}},
	{Name: "SAJOIN", Desc: "SAJOIN [<nick>] <channel>{,<channel>}", Args: []ArgType{ArgNick, ArgChannel}, Syntax: []string{"[nick:nick]", "<channel:channel>"}},
	{Name: "UBAN", Desc: "UBAN <ADD|DEL|LIST|INFO> [<args>...]", Syntax: []string{"<ADD|DEL|LIST|INFO>", "[args:string]"}},
	{Name: "DLINE", Desc: "DLINE [ANDKILL] [<duration>] <ip/net> [<reason>]", Syntax: []string{"[ANDKILL]", "[duration:string]", "<ip/net:string>", "[reason:string]"}},
	{Name: "KLINE", Desc: "KLINE [ANDKILL] [<duration>] <mask> [<reason>]", Syntax: []string{"[ANDKILL]", "[duration:string]", "<mask:string>", "[reason:string]"}},
	{Name: "DEFCON", Desc: "DEFCON [<level>]", Syntax: []string{"[level:string]"}},
	{Name: "CHATHISTORY", Desc: "CHATHISTORY <subcommand> <target> <reference> <limit>", Syntax: []string{"<subcommand:string>", "<target:string>", "<reference:string>", "<limit:string>"}},
	// Ergo services shorthand aliases — routed through service path, no Syntax needed here.
	{Name: "NS", Desc: "NS <command> [<args>...] — NickServ shorthand"},
	{Name: "CS", Desc: "CS <command> [<args>...] — ChanServ shorthand"},
	{Name: "NICKSERV", Desc: "NICKSERV <command> [<args>...] — Account management"},
	{Name: "CHANSERV", Desc: "CHANSERV <command> [<args>...] — Channel management"},
	{Name: "HOSTSERV", Desc: "HOSTSERV <command> [<args>...] — Virtual host management"},
	{Name: "HISTSERV", Desc: "HISTSERV <command> [<args>...] — Message history management"},
}

var paletteBg = lipgloss.Color("235")

var (
	paletteSelStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).Background(lipgloss.Color("237"))
	paletteNormalStyle = lipgloss.NewStyle().Faint(true).Background(paletteBg)
	palettePadStyle    = lipgloss.NewStyle().Background(paletteBg)
	paletteDescStyle   = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")).
				Background(lipgloss.Color("233")).
				BorderBackground(lipgloss.Color("233")).
				Padding(0, 1)
)

// completionKind distinguishes palette display and fill behavior.
type completionKind int

const (
	completionCommand    completionKind = iota // command name list (fill whole input)
	completionArg                              // argument completion (fill last arg, no desc)
	completionSubcommand                       // service subcommand (fill last arg, show desc)
)

type paletteModel struct {
	matches  []Command
	selected int
	visible  bool
	kind     completionKind
	maxShow  int

	// Syntax hint state.
	syntaxPrefix string   // command chain, e.g. "PURGE ADD"
	syntaxTokens []string // from Command.Syntax
	syntaxArgIdx int      // active token index, -1 for none
}

func newPalette() paletteModel {
	return paletteModel{maxShow: 8}
}

// SetSyntaxHint configures the syntax hint line. argIdx is clamped to valid range.
func (p *paletteModel) SetSyntaxHint(prefix string, tokens []string, argIdx int) {
	p.syntaxPrefix = prefix
	p.syntaxTokens = tokens
	if argIdx >= len(tokens) {
		argIdx = len(tokens) - 1
	}
	p.syntaxArgIdx = argIdx
	if len(tokens) > 0 {
		p.visible = true
	}
}

// ClearSyntaxHint resets all syntax hint state.
func (p *paletteModel) ClearSyntaxHint() {
	p.syntaxPrefix = ""
	p.syntaxTokens = nil
	p.syntaxArgIdx = -1
}

// hasSyntaxHint reports whether a syntax hint is configured.
func (p *paletteModel) hasSyntaxHint() bool {
	return len(p.syntaxTokens) > 0
}

// renderSyntax renders the syntax hint line with the active token highlighted.
func (p *paletteModel) renderSyntax(width int) string {
	if !p.hasSyntaxHint() {
		return ""
	}
	var b strings.Builder
	b.WriteString(palettePadStyle.Render(" "))
	used := 1

	// Prefix (e.g. "PURGE ADD") in faint.
	if p.syntaxPrefix != "" {
		pfx := p.syntaxPrefix + " "
		b.WriteString(paletteNormalStyle.Render(pfx))
		used += lipgloss.Width(pfx)
	}

	// Tokens — active in bold yellow-orange, others faint.
	for i, tok := range p.syntaxTokens {
		if i > 0 {
			b.WriteString(paletteNormalStyle.Render(" "))
			used++
		}
		if i == p.syntaxArgIdx {
			b.WriteString(paletteSelStyle.Render(tok))
		} else {
			b.WriteString(paletteNormalStyle.Render(tok))
		}
		used += lipgloss.Width(tok)
	}

	// Pad remainder.
	if rem := width - used; rem > 0 {
		b.WriteString(palettePadStyle.Render(strings.Repeat(" ", rem)))
	}
	return b.String()
}

// UpdateCompletions filters items against the given pattern for argument completion.
func (p *paletteModel) UpdateCompletions(filter string, items []string) {
	type scored struct {
		name  string
		score int
	}
	var results []scored
	for _, item := range items {
		if filter == "" {
			results = append(results, scored{item, 0})
		} else if s, ok := fuzzyScore(filter, item); ok {
			results = append(results, scored{item, s})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score > results[j].score
		}
		return results[i].name < results[j].name
	})

	p.matches = make([]Command, len(results))
	for i, r := range results {
		p.matches[i] = Command{Name: r.name}
	}
	p.selected = -1 // nothing selected until Tab
	p.visible = len(p.matches) > 0 || p.hasSyntaxHint()
	p.kind = completionArg
}

// UpdateSubcommands filters service subcommands — shows descriptions but fills last arg.
func (p *paletteModel) UpdateSubcommands(filter string, cmds []Command) {
	p.updateWith(filter, cmds)
	p.kind = completionSubcommand
}

// fillsLastArg reports whether the palette replaces the last arg (vs whole input).
func (p *paletteModel) fillsLastArg() bool {
	return p.kind != completionCommand
}

// SelectedName returns the Name of the selected item.
func (p *paletteModel) SelectedName() (string, bool) {
	if len(p.matches) == 0 || p.selected < 0 {
		return "", false
	}
	return p.matches[p.selected].Name, true
}

// Update filters herald commands against the given pattern.
func (p *paletteModel) Update(filter string) {
	p.updateWith(filter, commands)
}

// UpdateRaw filters raw IRC commands against the given pattern.
func (p *paletteModel) UpdateRaw(filter string) {
	p.updateWith(filter, rawCommands)
}

func (p *paletteModel) updateWith(filter string, cmds []Command) {
	p.kind = completionCommand
	if filter == "" {
		p.matches = cmds
		p.selected = -1
		p.visible = true
		return
	}

	type scored struct {
		cmd   Command
		score int
	}
	var results []scored
	for _, c := range cmds {
		best, matched := fuzzyScore(filter, c.Name)
		for _, alias := range c.Aliases {
			if s, ok := fuzzyScore(filter, alias); ok {
				if !matched || s > best {
					best = s
					matched = true
				}
			}
		}
		if matched {
			results = append(results, scored{c, best})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score > results[j].score
		}
		return results[i].cmd.Name < results[j].cmd.Name
	})

	p.matches = make([]Command, len(results))
	for i, r := range results {
		p.matches[i] = r.cmd
	}
	p.selected = -1
	p.visible = len(p.matches) > 0
}

// gridLayout computes column dimensions for the current matches.
// Always uses 4 columns (or fewer if there aren't enough items).
func (p *paletteModel) gridLayout(width int) (numCols, numRows, colWidth int) {
	numCols = max(min(4, len(p.matches)), 1)
	colWidth = (width - 2) / numCols // 2 = 1 cell padding on each side

	numRows = min((len(p.matches)+numCols-1)/numCols, p.maxShow)
	return
}

// renderDesc renders the description box for the selected item, or "" if none.
func (p *paletteModel) renderDesc(width int) string {
	if p.kind == completionArg || len(p.matches) == 0 || p.selected < 0 {
		return ""
	}
	sel := p.matches[p.selected]
	if sel.Desc == "" {
		return ""
	}
	content := sel.Desc
	if len(sel.Aliases) > 0 {
		content += "\nAliases: " + strings.Join(sel.Aliases, ", ")
	}
	boxWidth := max(width-2, 1)
	return paletteDescStyle.Width(boxWidth).Render(content)
}

// View renders the palette as a multi-column grid with a description box.
func (p *paletteModel) View(width int) string {
	if !p.visible || (len(p.matches) == 0 && !p.hasSyntaxHint()) {
		return ""
	}

	var sections []string

	// Syntax hint or description box (mutually exclusive — syntax takes priority).
	if syn := p.renderSyntax(width); syn != "" {
		sections = append(sections, syn)
	} else if desc := p.renderDesc(width); desc != "" {
		sections = append(sections, desc)
	}

	// Grid rows — only when there are matches.
	if len(p.matches) > 0 {
		numCols, numRows, colWidth := p.gridLayout(width)
		lpad := palettePadStyle.Render(" ")
		for r := range numRows {
			var row strings.Builder
			row.WriteString(lpad)
			used := 1
			for c := range numCols {
				idx := c*numRows + r
				if idx >= len(p.matches) {
					break
				}
				name := p.matches[idx].Name
				nameWidth := lipgloss.Width(name)
				pad := max(colWidth-nameWidth, 0)
				if idx == p.selected {
					row.WriteString(paletteSelStyle.Render(name))
				} else {
					row.WriteString(paletteNormalStyle.Render(name))
				}
				row.WriteString(palettePadStyle.Render(strings.Repeat(" ", pad)))
				used += nameWidth + pad
			}
			// Fill remainder (right padding + integer division gap).
			if rem := width - used; rem > 0 {
				row.WriteString(palettePadStyle.Render(strings.Repeat(" ", rem)))
			}
			sections = append(sections, row.String())
		}
	}

	return strings.Join(sections, "\n")
}

// Height returns the number of terminal lines the palette occupies.
func (p *paletteModel) Height(width int) int {
	if !p.visible || (len(p.matches) == 0 && !p.hasSyntaxHint()) {
		return 0
	}
	h := 0
	if p.hasSyntaxHint() {
		h = 1
	} else if desc := p.renderDesc(width); desc != "" {
		h = lipgloss.Height(desc)
	}
	if len(p.matches) > 0 {
		_, numRows, _ := p.gridLayout(width)
		h += numRows
	}
	return h
}

// Selected returns the currently selected command, if any.
func (p *paletteModel) Selected() (Command, bool) {
	if len(p.matches) == 0 || p.selected < 0 {
		return Command{}, false
	}
	return p.matches[p.selected], true
}

// Next moves selection down, wrapping around.
func (p *paletteModel) Next() {
	if len(p.matches) == 0 {
		return
	}
	if p.selected < 0 {
		p.selected = 0
		return
	}
	p.selected = (p.selected + 1) % len(p.matches)
}

// Prev moves selection up, wrapping around.
func (p *paletteModel) Prev() {
	if len(p.matches) == 0 {
		return
	}
	if p.selected < 0 {
		p.selected = len(p.matches) - 1
		return
	}
	p.selected = (p.selected - 1 + len(p.matches)) % len(p.matches)
}

// Hide hides the palette.
func (p *paletteModel) Hide() {
	p.visible = false
}

// resolveAlias returns the canonical command name for an alias, or the input unchanged.
func resolveAlias(name string) string {
	lower := strings.ToLower(name)
	for _, c := range commands {
		if strings.EqualFold(c.Name, lower) {
			return c.Name
		}
		for _, a := range c.Aliases {
			if strings.EqualFold(a, lower) {
				return c.Name
			}
		}
	}
	return name
}

// fuzzyScore checks if pattern is a subsequence of candidate (case-insensitive).
// Returns (score, true) on match. Higher scores indicate better matches.
// Bonuses: prefix match, consecutive chars.
func fuzzyScore(pattern, candidate string) (int, bool) {
	p := []rune(strings.ToLower(pattern))
	c := []rune(strings.ToLower(candidate))

	if len(p) == 0 {
		return 0, true
	}
	if len(p) > len(c) {
		return 0, false
	}

	score := 0
	pi := 0
	lastMatch := -1
	for ci := 0; ci < len(c) && pi < len(p); ci++ {
		if unicode.ToLower(c[ci]) == unicode.ToLower(p[pi]) {
			if ci == pi {
				score += 3 // prefix bonus
			}
			if lastMatch == ci-1 {
				score += 2 // consecutive bonus
			}
			score += 1 // base match
			lastMatch = ci
			pi++
		}
	}

	if pi < len(p) {
		return 0, false
	}
	if len(p) == len(c) {
		score += 10 // exact match bonus
	}
	return score, true
}
