package cmd

import (
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

const (
	pongWait          = 60 * time.Second
	pingPeriod        = (pongWait * 9) / 10
	disconnectGrace   = 5 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var socket *websocket.Conn

var (
	connMu         sync.Mutex
	activeConns    int
	seenFirstConn  bool
	shutdownTimer  *time.Timer
)

func registerConn() {
	connMu.Lock()
	defer connMu.Unlock()
	activeConns++
	seenFirstConn = true
	if shutdownTimer != nil {
		shutdownTimer.Stop()
		shutdownTimer = nil
	}
}

func unregisterConn() {
	connMu.Lock()
	defer connMu.Unlock()
	if activeConns > 0 {
		activeConns--
	}
	if seenFirstConn && activeConns == 0 {
		shutdownTimer = time.AfterFunc(disconnectGrace, func() {
			logInfo("No clients connected for %s, shutting down\n", disconnectGrace)
			os.Exit(0)
		})
	}
}

func wsHandler(watcher *fsnotify.Watcher) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reload := make(chan bool, 1)
		errorChan := make(chan error)
		done := make(chan interface{})

		go watch(done, errorChan, reload, watcher)

		var err error
		socket, err = upgrader.Upgrade(w, r, nil)
		if err != nil {
			if _, ok := err.(websocket.HandshakeError); !ok {
				logDebug("Debug [handshake error]: %s", err)
			}
			return
		}
		registerConn()
		socket.SetReadDeadline(time.Now().Add(pongWait))
		socket.SetPongHandler(func(string) error { socket.SetReadDeadline(time.Now().Add(pongWait)); return nil })

		go wsReader(done, errorChan)
		go wsWriter(done, errorChan, reload)

		err = <-errorChan
		close(done)
		logInfo("Close WebSocket: %v\n", err)
		socket.Close()
		unregisterConn()
	})
}

func wsReader(done <-chan interface{}, errorChan chan<- error) {
	for {
		_, _, err := socket.ReadMessage()
		if err != nil {
			logDebug("Debug [read message]: %s", err)
			select {
			case errorChan <- err:
			case <-done:
			}
			return
		}
	}
}

func wsWriter(done <-chan interface{}, errChan chan<- error, reload <-chan bool) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-reload:
			err := socket.WriteMessage(websocket.TextMessage, []byte("reload"))
			if err != nil {
				logDebug("Debug [reload error]: %v", err)
				select {
				case errChan <- err:
				case <-done:
				}
				return
			}
		case <-ticker.C:
			logDebug("Debug [ping send]: ping to client")
			err := socket.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				logDebug("Debug [ping error]: %v", err)
				select {
				case errChan <- err:
				case <-done:
				}
				return
			}
		case <-done:
			return
		}
	}
}
