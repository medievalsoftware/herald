package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/medievalsoftware/herald/internal/config"
	"github.com/medievalsoftware/herald/internal/proxy"
	"github.com/medievalsoftware/herald/internal/tui"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "connect":
		if err := runConnect(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "herald: %v\n", err)
			os.Exit(1)
		}
	case "proxy":
		if err := runProxy(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "herald: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: herald <command> [flags] <server>

Commands:
  connect    Connect to an IRC server over WebSocket (TUI)
  proxy      TCP-to-WebSocket proxy for traditional IRC clients

Run 'herald <command> -help' for command-specific flags.
`)
}

func runConnect(args []string) error {
	fs := flag.NewFlagSet("connect", flag.ExitOnError)
	nick := fs.String("nick", "herald", "IRC nickname")
	pass := fs.String("pass", "", "Server password (sent via PASS during registration)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: herald connect [flags] wss://server\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		fs.Usage()
		return fmt.Errorf("missing server address")
	}
	addr := fs.Arg(0)

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	theme, err := config.LoadTheme(cfg.Theme)
	if err != nil {
		fmt.Fprintf(os.Stderr, "herald: %v, using defaults\n", err)
		theme = config.DefaultTheme()
	}
	tui.ApplyTheme(theme)
	m := tui.New(addr, *nick, *pass, cfg)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	m.SetProgram(p)

	_, err = p.Run()
	return err
}

func runProxy(args []string) error {
	fs := flag.NewFlagSet("proxy", flag.ExitOnError)
	listen := fs.String("listen", "127.0.0.1:6667", "TCP listen address")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: herald proxy [flags] wss://server\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		fs.Usage()
		return fmt.Errorf("missing server address")
	}
	addr := fs.Arg(0)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	fmt.Printf("Herald proxy: %s -> %s\n", *listen, addr)
	return proxy.Run(ctx, *listen, addr)
}
