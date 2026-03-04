package tui

import (
	"testing"
)

func TestFormatTrailingArg(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty input", "", ""},
		{"unknown command", "FAKECMD #chan hello", "FAKECMD #chan hello"},
		{"no syntax", "MOTD", "MOTD"},
		{"no excess args", "TOPIC #general", "TOPIC #general"},
		{"trailing joined", "TOPIC #general Hello, world!", "TOPIC #general :Hello, world!"},
		{"already prefixed", "TOPIC #general :Already prefixed", "TOPIC #general :Already prefixed"},
		{"multi-arg trailing", "KICK #chan user bad behavior", "KICK #chan user :bad behavior"},
		{"last token not string", "INVITE nick #chan extra", "INVITE nick #chan extra"},
		{"lowercase command", "topic #general Hello world", "topic #general :Hello world"},
		{"service no excess", "NS IDENTIFY myuser mypass", "NS IDENTIFY myuser mypass"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTrailingArg(tt.input)
			if got != tt.want {
				t.Errorf("formatTrailingArg(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExpandEnvBraces(t *testing.T) {
	t.Setenv("FOO", "bar")
	t.Setenv("IRC_PASS", "secret123")
	t.Setenv("EMPTY", "")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "hello ${FOO} world", "hello bar world"},
		{"bare dollar ignored", "hello $FOO world", "hello $FOO world"},
		{"multiple vars", "${FOO} and ${IRC_PASS}", "bar and secret123"},
		{"no vars", "plain text", "plain text"},
		{"adjacent to text", "pre${FOO}post", "prebarpost"},
		{"missing var expands to empty", "hello ${NONEXISTENT} world", "hello  world"},
		{"empty var value", "hello ${EMPTY} world", "hello  world"},
		{"unclosed brace", "hello ${FOO world", "hello ${FOO world"},
		{"dollar alone", "costs $5", "costs $5"},
		{"dollar brace no name", "hello ${} world", "hello  world"},
		{"nested dollar signs", "$$${FOO}", "$$bar"},
		{"command at start", "OPER dane ${IRC_PASS}", "OPER dane secret123"},
		{"only var", "${FOO}", "bar"},
		{"empty input", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandEnvBraces(tt.input)
			if got != tt.want {
				t.Errorf("expandEnvBraces(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
