package proxy

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"

	"github.com/coder/websocket"
)

// Run starts a TCP listener that bridges connections to the WebSocket IRC server.
func Run(ctx context.Context, listenAddr, wsAddr string) error {
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", listenAddr, err)
	}
	defer func() { _ = ln.Close() }()

	log.Printf("proxy: listening on %s -> %s", listenAddr, wsAddr)

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("proxy: accept error: %v", err)
			continue
		}
		go handleConn(ctx, conn, wsAddr)
	}
}

func handleConn(ctx context.Context, tcpConn net.Conn, wsAddr string) {
	defer func() { _ = tcpConn.Close() }()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wsConn, _, err := websocket.Dial(ctx, wsAddr, nil)
	if err != nil {
		log.Printf("proxy: dial %s: %v", wsAddr, err)
		return
	}
	defer func() { _ = wsConn.CloseNow() }()

	log.Printf("proxy: new connection from %s", tcpConn.RemoteAddr())

	// TCP -> WS: read \r\n-delimited lines, send as WebSocket frames.
	go func() {
		defer cancel()
		scanner := bufio.NewScanner(tcpConn)
		for scanner.Scan() {
			line := scanner.Text()
			if err := wsConn.Write(ctx, websocket.MessageText, []byte(line)); err != nil {
				if ctx.Err() == nil {
					log.Printf("proxy: ws write error: %v", err)
				}
				return
			}
		}
		if err := scanner.Err(); err != nil && ctx.Err() == nil {
			log.Printf("proxy: tcp read error: %v", err)
		}
	}()

	// WS -> TCP: read WebSocket frames, write with \r\n to TCP.
	for {
		_, data, err := wsConn.Read(ctx)
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("proxy: ws read error: %v", err)
			}
			return
		}
		if _, err := fmt.Fprintf(tcpConn, "%s\r\n", data); err != nil {
			if ctx.Err() == nil {
				log.Printf("proxy: tcp write error: %v", err)
			}
			return
		}
	}
}
