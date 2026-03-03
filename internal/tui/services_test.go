package tui

import "testing"

func TestIsServiceCommand(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantFound bool
	}{
		{"canonical nickserv", "NICKSERV", "NICKSERV", true},
		{"canonical chanserv", "CHANSERV", "CHANSERV", true},
		{"canonical hostserv", "HOSTSERV", "HOSTSERV", true},
		{"canonical histserv", "HISTSERV", "HISTSERV", true},
		{"alias NS", "NS", "NICKSERV", true},
		{"alias CS", "CS", "CHANSERV", true},
		{"case insensitive", "nickserv", "NICKSERV", true},
		{"alias lowercase", "ns", "NICKSERV", true},
		{"non-service", "PRIVMSG", "", false},
		{"empty", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, ok := isServiceCommand(tt.input)
			if ok != tt.wantFound {
				t.Errorf("isServiceCommand(%q) found=%v, want %v", tt.input, ok, tt.wantFound)
			}
			if name != tt.wantName {
				t.Errorf("isServiceCommand(%q) name=%q, want %q", tt.input, name, tt.wantName)
			}
		})
	}
}

func TestFindServiceSubcommand(t *testing.T) {
	tests := []struct {
		name      string
		service   string
		subcmd    string
		wantName  string
		wantFound bool
	}{
		{"direct name", "NICKSERV", "IDENTIFY", "IDENTIFY", true},
		{"alias PASSWORD→PASSWD", "NICKSERV", "PASSWORD", "PASSWD", true},
		{"alias DROP→UNREGISTER for chanserv", "CHANSERV", "DROP", "UNREGISTER", true},
		{"case insensitive", "NICKSERV", "identify", "IDENTIFY", true},
		{"not found", "NICKSERV", "NONEXISTENT", "", false},
		{"invalid service", "BOGUS", "IDENTIFY", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, ok := findServiceSubcommand(tt.service, tt.subcmd)
			if ok != tt.wantFound {
				t.Errorf("findServiceSubcommand(%q, %q) found=%v, want %v", tt.service, tt.subcmd, ok, tt.wantFound)
			}
			if ok && cmd.Name != tt.wantName {
				t.Errorf("findServiceSubcommand(%q, %q) name=%q, want %q", tt.service, tt.subcmd, cmd.Name, tt.wantName)
			}
		})
	}
}

func TestPaletteUpdateSubcommands(t *testing.T) {
	p := newPalette()

	t.Run("shows all subcommands unfiltered", func(t *testing.T) {
		cmds := serviceSubcommands["NICKSERV"]
		p.UpdateSubcommands("", cmds)
		if !p.visible {
			t.Fatal("palette should be visible")
		}
		if p.kind != completionSubcommand {
			t.Errorf("kind=%v, want completionSubcommand", p.kind)
		}
		if len(p.matches) != len(cmds) {
			t.Errorf("matches=%d, want %d", len(p.matches), len(cmds))
		}
	})

	t.Run("descriptions preserved", func(t *testing.T) {
		cmds := serviceSubcommands["NICKSERV"]
		p.UpdateSubcommands("", cmds)
		found := false
		for _, m := range p.matches {
			if m.Name == "IDENTIFY" {
				found = true
				if m.Desc == "" {
					t.Error("IDENTIFY should have a description")
				}
			}
		}
		if !found {
			t.Error("IDENTIFY not found in matches")
		}
	})

	t.Run("fuzzy filtering", func(t *testing.T) {
		cmds := serviceSubcommands["NICKSERV"]
		p.UpdateSubcommands("IDE", cmds)
		if len(p.matches) == 0 {
			t.Fatal("expected matches for 'IDE'")
		}
		if p.matches[0].Name != "IDENTIFY" {
			t.Errorf("top match=%q, want IDENTIFY", p.matches[0].Name)
		}
	})

	t.Run("fillsLastArg true for subcommand", func(t *testing.T) {
		cmds := serviceSubcommands["NICKSERV"]
		p.UpdateSubcommands("", cmds)
		if !p.fillsLastArg() {
			t.Error("fillsLastArg should be true for subcommand completion")
		}
	})

	t.Run("renderDesc shows for subcommand", func(t *testing.T) {
		cmds := serviceSubcommands["NICKSERV"]
		p.UpdateSubcommands("", cmds)
		p.selected = 0
		desc := p.renderDesc(80)
		if desc == "" {
			t.Error("renderDesc should show description for subcommand kind")
		}
	})
}

func TestSubcommandChain(t *testing.T) {
	t.Run("SUSPEND has subcommands", func(t *testing.T) {
		cmd, ok := findServiceSubcommand("NICKSERV", "SUSPEND")
		if !ok {
			t.Fatal("SUSPEND not found")
		}
		if len(cmd.Subcommands) != 3 {
			t.Fatalf("SUSPEND subcommands=%d, want 3", len(cmd.Subcommands))
		}
		names := map[string]bool{}
		for _, sc := range cmd.Subcommands {
			names[sc.Name] = true
		}
		for _, want := range []string{"ADD", "DEL", "LIST"} {
			if !names[want] {
				t.Errorf("missing subcommand %q", want)
			}
		}
	})

	t.Run("SUSPEND ADD has ArgNick", func(t *testing.T) {
		cmd, _ := findServiceSubcommand("NICKSERV", "SUSPEND")
		add, ok := findCommand(cmd.Subcommands, "ADD")
		if !ok {
			t.Fatal("ADD not found in SUSPEND subcommands")
		}
		if len(add.Args) != 1 || add.Args[0] != ArgNick {
			t.Errorf("SUSPEND ADD args=%v, want [ArgNick]", add.Args)
		}
	})

	t.Run("PURGE has subcommands with ArgChannel", func(t *testing.T) {
		cmd, ok := findServiceSubcommand("CHANSERV", "PURGE")
		if !ok {
			t.Fatal("PURGE not found")
		}
		if len(cmd.Subcommands) != 3 {
			t.Fatalf("PURGE subcommands=%d, want 3", len(cmd.Subcommands))
		}
		add, ok := findCommand(cmd.Subcommands, "ADD")
		if !ok {
			t.Fatal("ADD not found in PURGE subcommands")
		}
		if len(add.Args) != 1 || add.Args[0] != ArgChannel {
			t.Errorf("PURGE ADD args=%v, want [ArgChannel]", add.Args)
		}
	})

	t.Run("CERT has subcommands without args", func(t *testing.T) {
		cmd, ok := findServiceSubcommand("NICKSERV", "CERT")
		if !ok {
			t.Fatal("CERT not found")
		}
		if len(cmd.Subcommands) != 3 {
			t.Fatalf("CERT subcommands=%d, want 3", len(cmd.Subcommands))
		}
		for _, sc := range cmd.Subcommands {
			if len(sc.Args) != 0 {
				t.Errorf("CERT %s should have no args, got %v", sc.Name, sc.Args)
			}
		}
	})

	t.Run("CLIENTS has subcommands with optional ArgNick", func(t *testing.T) {
		cmd, ok := findServiceSubcommand("NICKSERV", "CLIENTS")
		if !ok {
			t.Fatal("CLIENTS not found")
		}
		if len(cmd.Subcommands) != 2 {
			t.Fatalf("CLIENTS subcommands=%d, want 2", len(cmd.Subcommands))
		}
		list, ok := findCommand(cmd.Subcommands, "LIST")
		if !ok {
			t.Fatal("LIST not found in CLIENTS subcommands")
		}
		if len(list.Args) != 1 || list.Args[0] != ArgNick {
			t.Errorf("CLIENTS LIST args=%v, want [ArgNick]", list.Args)
		}
	})

	t.Run("PUSH has subcommands", func(t *testing.T) {
		cmd, ok := findServiceSubcommand("NICKSERV", "PUSH")
		if !ok {
			t.Fatal("PUSH not found")
		}
		if len(cmd.Subcommands) != 2 {
			t.Fatalf("PUSH subcommands=%d, want 2", len(cmd.Subcommands))
		}
	})
}

func TestServiceNickFor(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{"NS alias", "NS IDENTIFY pass", "NickServ"},
		{"CS alias", "CS OP #ch nick", "ChanServ"},
		{"canonical NICKSERV", "NICKSERV INFO", "NickServ"},
		{"canonical HOSTSERV", "HOSTSERV ON", "HostServ"},
		{"canonical HISTSERV", "HISTSERV PLAY #ch", "HistServ"},
		{"case insensitive", "ns identify pass", "NickServ"},
		{"non-service", "PRIVMSG #ch :hi", ""},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serviceNickFor(tt.text)
			if got != tt.want {
				t.Errorf("serviceNickFor(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}
