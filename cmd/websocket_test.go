package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWriter(t *testing.T) {
	dir := t.TempDir()
	testFile, err := os.CreateTemp(dir, "markdown-preview-test")
	if err != nil {
		t.Fatalf("%v", err)
	}

	_, _ = testFile.Write([]byte("BEFORE.\n"))

	watcher, err := createWatcher(dir)

	if err != nil {
		t.Fatalf("%v", err)
	}

	s := httptest.NewServer(http.Handler(wsHandler(watcher)))

	u := "ws" + strings.TrimPrefix(s.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	<-time.After(50 * time.Millisecond) //XXX

	defer ws.Close()
	defer s.Close()

	_, err = testFile.Write([]byte("AFTER.\n"))
	if err != nil {
		t.Fatalf("%v", err)
	}

	_, p, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("%v", err)
	}

	actual := string(p)
	expected := "reload"
	if actual != expected {
		t.Errorf("got %v\n want %v", actual, expected)
	}
}
