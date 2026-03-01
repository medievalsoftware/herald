package client

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/coder/websocket"
	"github.com/ergochat/irc-go/ircmsg"
)

// desiredCaps lists the IRCv3 capabilities Herald will request.
var desiredCaps = []string{"batch", "server-time", "chathistory", "draft/chathistory"}

// Client manages a WebSocket connection to an IRC server.
type Client struct {
	conn     *websocket.Conn
	dispatch func(any)
	mu       sync.Mutex
	nick     string
	caps     map[string]string
}

// New creates a Client that will send tea.Msg values through dispatch.
func New(dispatch func(any)) *Client {
	return &Client{dispatch: dispatch}
}

// Connect dials the WebSocket server and performs IRC registration with
// CAP negotiation.
func (c *Client) Connect(ctx context.Context, addr, nick string) error {
	conn, _, err := websocket.Dial(ctx, addr, nil)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	c.conn = conn
	c.nick = nick

	// Begin CAP negotiation.
	if err := c.Send(ctx, "CAP LS 302"); err != nil {
		_ = conn.CloseNow()
		return err
	}

	if err := c.negotiateCAPs(ctx); err != nil {
		_ = conn.CloseNow()
		return err
	}

	// IRC registration.
	if err := c.Send(ctx, "NICK "+nick); err != nil {
		_ = conn.CloseNow()
		return err
	}
	if err := c.Send(ctx, "USER "+nick+" 0 * :Herald IRC Client"); err != nil {
		_ = conn.CloseNow()
		return err
	}
	if err := c.Send(ctx, "CAP END"); err != nil {
		_ = conn.CloseNow()
		return err
	}

	c.dispatch(ConnectedMsg{})
	go c.readLoop(ctx)
	return nil
}

// HasCap reports whether the server acknowledged the named capability.
func (c *Client) HasCap(name string) bool {
	_, ok := c.caps[name]
	return ok
}

// negotiateCAPs reads the CAP LS response, requests desired caps, and
// processes the ACK/NAK. Non-CAP messages received during negotiation are
// dispatched normally. If the server doesn't support CAP (421), negotiation
// is silently skipped.
func (c *Client) negotiateCAPs(ctx context.Context) error {
	// Collect advertised caps, handling multiline (CAP * LS *).
	advertised := make(map[string]string)
	for {
		msg, err := c.readMessage(ctx)
		if err != nil {
			return err
		}

		// 421 ERR_UNKNOWNCOMMAND for CAP — server doesn't support it.
		if msg.Command == "421" && len(msg.Params) >= 2 && strings.EqualFold(msg.Params[1], "CAP") {
			return nil
		}

		if msg.Command != "CAP" {
			// Dispatch non-CAP messages (e.g. NOTICE from server).
			c.dispatch(IRCMsg{msg})
			continue
		}

		// CAP LS response: params are [nick, "LS", capList] or [nick, "LS", "*", capList]
		if len(msg.Params) < 3 {
			continue
		}
		subcommand := msg.Params[1]
		if !strings.EqualFold(subcommand, "LS") {
			continue
		}

		multiline := false
		capStr := msg.Params[len(msg.Params)-1]
		if len(msg.Params) >= 4 && msg.Params[2] == "*" {
			multiline = true
		}

		for _, token := range strings.Fields(capStr) {
			name, value, _ := strings.Cut(token, "=")
			advertised[name] = value
		}

		if !multiline {
			break
		}
	}

	// Determine overlap with desired caps.
	var want []string
	for _, cap := range desiredCaps {
		if _, ok := advertised[cap]; ok {
			want = append(want, cap)
		}
	}
	if len(want) == 0 {
		return nil
	}

	// Request desired caps.
	if err := c.Send(ctx, "CAP REQ :"+strings.Join(want, " ")); err != nil {
		return err
	}

	// Read ACK/NAK.
	for {
		msg, err := c.readMessage(ctx)
		if err != nil {
			return err
		}
		if msg.Command != "CAP" {
			c.dispatch(IRCMsg{msg})
			continue
		}
		if len(msg.Params) < 3 {
			continue
		}
		subcommand := msg.Params[1]
		if strings.EqualFold(subcommand, "ACK") {
			c.caps = make(map[string]string)
			for _, cap := range strings.Fields(msg.Params[len(msg.Params)-1]) {
				c.caps[cap] = advertised[cap]
			}
			return nil
		}
		if strings.EqualFold(subcommand, "NAK") {
			// Server refused; proceed without caps.
			return nil
		}
	}
}

// readMessage reads and parses a single IRC message from the WebSocket.
func (c *Client) readMessage(ctx context.Context) (ircmsg.Message, error) {
	_, data, err := c.conn.Read(ctx)
	if err != nil {
		return ircmsg.Message{}, fmt.Errorf("read: %w", err)
	}
	msg, err := ircmsg.ParseLine(string(data))
	if err != nil {
		return ircmsg.Message{}, fmt.Errorf("parse IRC: %w", err)
	}
	return msg, nil
}

// Send writes a raw IRC line to the WebSocket.
func (c *Client) Send(ctx context.Context, line string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.Write(ctx, websocket.MessageText, []byte(line))
}

// Close cleanly shuts down the connection.
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close(websocket.StatusNormalClosure, "bye")
}

// Nick returns the current nickname.
func (c *Client) Nick() string {
	return c.nick
}

// SetNick updates the stored nickname (called when server confirms a nick change).
func (c *Client) SetNick(nick string) {
	c.nick = nick
}

func (c *Client) readLoop(ctx context.Context) {
	defer func() {
		c.dispatch(DisconnectedMsg{})
	}()

	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.dispatch(DisconnectedMsg{Err: err})
			return
		}

		msg, err := ircmsg.ParseLine(string(data))
		if err != nil {
			c.dispatch(ErrorMsg{Err: fmt.Errorf("parse IRC: %w", err)})
			continue
		}

		c.dispatch(IRCMsg{msg})
	}
}
