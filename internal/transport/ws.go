// Package transport ws.go implemented websocket requests.
// Here is preparing and send websocket requests.
package transport

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/Votline/Gurl-cli/internal/config"
	"github.com/Votline/Gurl-cli/internal/parser"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// doWS sends websocket request.
// Update result by pointer.
// If config body not empty, send body.
// If wsID is parser.WSwhile, handle websocket while loop.
// If dp is true, print response.
func (t *Transport) doWS(c *config.HTTPConfig, resObj *Result, wsID int, dp bool) error {
	const op = "transport.doWS"

	dialer := websocket.DefaultDialer

	if c.GetIgnrCrt() != nil {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		t.log.Warn("InsecureSkipVerify is true",
			zap.String("op", op),
			zap.String("url", unsafe.String(unsafe.SliceData(c.URL), len(c.URL))))
	} else {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: false}
	}

	h := make(http.Header)

	parser.ParseHeaders(c.Headers, func(k, v []byte) {
		key := unsafe.String(unsafe.SliceData(k), len(k))
		val := unsafe.String(unsafe.SliceData(v), len(v))
		h.Set(key, val)
	})

	conn, resp, err := dialer.Dial(unsafe.String(unsafe.SliceData(c.URL), len(c.URL)), h)
	if err != nil {
		if resp != nil {
			resObj.Info = Status{
				Code:       resp.StatusCode,
				Message:    resp.Status,
				ConfigType: "http",
			}
		}
		return fmt.Errorf("%s: dial: %w", op, err)
	}
	defer conn.Close()

	if len(c.Body) > 0 {
		t.log.Debug("Sending body",
			zap.String("op", op),
			zap.String("name", c.GetName()),
			zap.Int("id", c.GetID()),
			zap.String("body", unsafe.String(unsafe.SliceData(c.Body), len(c.Body))))
		if err := conn.WriteMessage(websocket.TextMessage, c.Body); err != nil {
			return fmt.Errorf("%s: write: %w", op, err)
		}
	}
	if wsID == parser.WSwhile {
		t.handleWSWhile(conn, c.GetName(), c.GetID(), dp, t.log)
		return nil
	}

	dur := parser.ParseWait(c.Timeout)
	if dur != 0 {
		conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(dur)))
	} else {
		t.log.Warn("ReadWSWait is empty. Using default timeout of 2 seconds",
			zap.String("op", op),
			zap.String("name", c.GetName()),
			zap.Int("id", c.GetID()))
		conn.SetReadDeadline(time.Now().Add(time.Second * 2))
	}

	var msg []byte
	_, msg, err = conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("%s: read: %w", op, err)
	}

	resObj.Raw = msg
	resObj.IsJSON = false
	resObj.Cookie = nil
	resObj.Info = Status{
		Code:       101,
		Message:    "101 Switching Protocols",
		ConfigType: "ws",
	}

	return nil
}

// handleWSWhile handle websocket while loop.
// Used os.Stdin for input.
// If input is "exit" or "quit", close connection.
func (t *Transport) handleWSWhile(conn *websocket.Conn, cfgName string, cfgID int, dp bool, log *zap.Logger) {
	const op = "transport.handleWSWhile"

	log.Warn("Connection changed to websockets. While loop is enabled",
		zap.String("config name", cfgName),
		zap.Int("config id", cfgID),
		zap.String("op", op))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Go(func() {
		<-ctx.Done()
		conn.Close()
		cancel()
	})

	wg.Go(func() {
		defer cancel()
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Error("Failed to read message",
					zap.String("op", op),
					zap.Error(err))
				return
			}

			if !dp {
				prettyPrintWS(cfgID, msg)
			}
		}
	})

	wg.Go(func() {
		defer cancel()
		defer conn.Close()

		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			input := scanner.Text()
			if input == "exit" || input == "quit" {
				log.Warn("Exiting while WebSocket",
					zap.String("config name", cfgName),
					zap.Int("config id", cfgID),
					zap.String("op", op))
				cancel()
				return
			}

			if ctx.Err() != nil {
				return
			}

			inputBytes := unsafe.Slice(unsafe.StringData(input), len(input))
			t.log.Debug("Sending body",
				zap.String("op", op),
				zap.String("name", cfgName),
				zap.Int("id", cfgID),
				zap.String("body", input))
			if err := conn.WriteMessage(websocket.TextMessage, inputBytes); err != nil {
				log.Error("Failed to write message",
					zap.String("op", op),
					zap.Error(err))
				return
			}
		}
	})

	wg.Wait()

	log.Warn("WebSocket connection closed",
		zap.String("config name", cfgName),
		zap.Int("config id", cfgID),
		zap.String("op", op))
}

// prettyPrintWS prints response.
func prettyPrintWS(cfgID int, msg []byte) {
	const op = "transport.prettyPrintWS"

	if len(msg) == 0 {
		return
	}

	fmt.Println(strings.Repeat("-", 20))

	fmt.Printf("\n\033[90m[ID %d]\033[0m", cfgID)
	fmt.Printf("\n\033[90m[Message]\033[0m")
	fmt.Printf("\n%s\n", unsafe.String(unsafe.SliceData(msg), len(msg)))
}
