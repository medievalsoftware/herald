package tui

import (
	"testing"
)

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
