package client

import "github.com/ergochat/irc-go/ircmsg"

// IRCMsg wraps a parsed IRC message for the TUI.
type IRCMsg struct {
	ircmsg.Message
}

// ConnectedMsg signals that the WebSocket connection is established.
type ConnectedMsg struct{}

// DisconnectedMsg signals that the connection was closed.
type DisconnectedMsg struct {
	Err error
}

// ErrorMsg carries a non-fatal error from the client.
type ErrorMsg struct {
	Err error
}
