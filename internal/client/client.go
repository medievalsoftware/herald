package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/coder/websocket"
	"github.com/ergochat/irc-go/ircmsg"
)

// Client manages a WebSocket connection to an IRC server.
type Client struct {
	conn     *websocket.Conn
	dispatch func(any)
	mu       sync.Mutex
	nick     string
}

// New creates a Client that will send tea.Msg values through dispatch.
func New(dispatch func(any)) *Client {
	return &Client{dispatch: dispatch}
}

// Connect dials the WebSocket server and performs IRC registration.
func (c *Client) Connect(ctx context.Context, addr, nick string) error {
	conn, _, err := websocket.Dial(ctx, addr, nil)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	c.conn = conn
	c.nick = nick

	// IRC registration.
	if err := c.Send(ctx, "NICK "+nick); err != nil {
		_ = conn.CloseNow()
		return err
	}
	if err := c.Send(ctx, "USER "+nick+" 0 * :Herald IRC Client"); err != nil {
		_ = conn.CloseNow()
		return err
	}

	c.dispatch(ConnectedMsg{})
	go c.readLoop(ctx)
	return nil
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
