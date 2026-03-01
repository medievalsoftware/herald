package tui

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

// Command describes an IRC command available in the palette.
type Command struct {
	Name    string
	Aliases []string
	Desc    string
	Args    bool
}

var commands = []Command{
	{Name: "join", Aliases: []string{"j"}, Desc: "Join a channel", Args: true},
	{Name: "leave", Aliases: []string{"part", "q"}, Desc: "Leave current channel", Args: true},
	{Name: "msg", Aliases: []string{"m", "query"}, Desc: "Send a direct message", Args: true},
	{Name: "me", Aliases: []string{"action"}, Desc: "Send an action", Args: true},
	{Name: "nick", Desc: "Change nickname", Args: true},
	{Name: "quit", Aliases: []string{"exit", "q!"}, Desc: "Disconnect from server", Args: false},
	{Name: "raw", Aliases: []string{"quote"}, Desc: "Send raw IRC command", Args: true},
	{Name: "set", Desc: "Change a setting", Args: true},
	{Name: "theme", Desc: "Switch color theme", Args: true},
}

var (
	paletteSelStyle    = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("235")).Padding(0, 1)
	paletteNormalStyle = lipgloss.NewStyle().Faint(true).Padding(0, 1)
)

type paletteModel struct {
	matches        []Command
	selected       int
	visible        bool
	completionMode bool
	maxShow        int
}

func newPalette() paletteModel {
	return paletteModel{maxShow: 8}
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
	if p.selected >= len(p.matches) {
		p.selected = max(0, len(p.matches)-1)
	}
	p.visible = len(p.matches) > 0
	p.completionMode = true
}

// SelectedName returns the Name of the selected item.
func (p *paletteModel) SelectedName() (string, bool) {
	if len(p.matches) == 0 {
		return "", false
	}
	return p.matches[p.selected].Name, true
}

// Update filters commands against the given pattern and makes the palette visible.
func (p *paletteModel) Update(filter string) {
	p.completionMode = false
	if filter == "" {
		p.matches = commands
		p.selected = 0
		p.visible = true
		return
	}

	type scored struct {
		cmd   Command
		score int
	}
	var results []scored
	for _, c := range commands {
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
	if p.selected >= len(p.matches) {
		p.selected = max(0, len(p.matches)-1)
	}
	p.visible = len(p.matches) > 0
}

// View renders the palette rows. Returns empty string when hidden or no matches.
func (p *paletteModel) View(width int) string {
	if !p.visible || len(p.matches) == 0 {
		return ""
	}

	selStyle := paletteSelStyle.Width(width)
	normalStyle := paletteNormalStyle.Width(width)

	show := len(p.matches)
	if show > p.maxShow {
		show = p.maxShow
	}

	var rows []string
	for i := range show {
		cmd := p.matches[i]

		var content string
		if p.completionMode {
			content = cmd.Name
		} else {
			name := cmd.Name
			desc := cmd.Desc
			if len(cmd.Aliases) > 0 {
				desc += " (" + strings.Join(cmd.Aliases, ", ") + ")"
			}
			gap := width - lipgloss.Width(name) - lipgloss.Width(desc) - 2
			if gap < 1 {
				gap = 1
			}
			content = name + fmt.Sprintf("%*s", gap, "") + desc
		}

		if i == p.selected {
			rows = append(rows, selStyle.Render(content))
		} else {
			rows = append(rows, normalStyle.Render(content))
		}
	}

	return strings.Join(rows, "\n")
}

// Height returns the number of terminal lines the palette occupies.
func (p *paletteModel) Height() int {
	if !p.visible || len(p.matches) == 0 {
		return 0
	}
	h := len(p.matches)
	if h > p.maxShow {
		h = p.maxShow
	}
	return h
}

// Selected returns the currently selected command, if any.
func (p *paletteModel) Selected() (Command, bool) {
	if len(p.matches) == 0 {
		return Command{}, false
	}
	return p.matches[p.selected], true
}

// Next moves selection down, wrapping around.
func (p *paletteModel) Next() {
	if len(p.matches) == 0 {
		return
	}
	p.selected = (p.selected + 1) % len(p.matches)
}

// Prev moves selection up, wrapping around.
func (p *paletteModel) Prev() {
	if len(p.matches) == 0 {
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
