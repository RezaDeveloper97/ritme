package notify_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/notify"
	"github.com/ritme/backend-go/internal/platform/config"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestTelegram_DisabledWithoutEnv(t *testing.T) {
	tg := notify.NewTelegram(config.Telegram{BotToken: "t"}, nil, quiet)
	assert.False(t, tg.Enabled())
	assert.False(t, tg.Send(context.Background(), "hi"))
	var nilTg *notify.Telegram
	assert.False(t, nilTg.Enabled())
	done := make(chan bool, 1)
	nilTg.SendAsync("x", func(ok bool) { done <- ok })
	assert.False(t, <-done)
}

func TestTelegram_Send(t *testing.T) {
	var got map[string]any
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tg := notify.NewTelegram(config.Telegram{BotToken: "tok", ChatID: "42", Timeout: time.Second}, srv.Client(), quiet).WithAPIBase(srv.URL)
	require.True(t, tg.Send(context.Background(), "<b>x</b>"))
	assert.Equal(t, "/bottok/sendMessage", path)
	assert.Equal(t, map[string]any{"chat_id": "42", "text": "<b>x</b>", "parse_mode": "HTML", "disable_web_page_preview": true}, got)
}

func TestTelegram_FailureIsSwallowed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	tg := notify.NewTelegram(config.Telegram{BotToken: "tok", ChatID: "42"}, srv.Client(), quiet).WithAPIBase(srv.URL)
	assert.False(t, tg.Send(context.Background(), "x"))

	done := make(chan bool, 1)
	tg.SendAsync("x", func(ok bool) { done <- ok })
	assert.False(t, <-done)
}

func TestEscape(t *testing.T) {
	assert.Equal(t, "&lt;a href=&quot;x&quot;&gt;Tom &amp; Jerry&#039;s&lt;/a&gt;", notify.Escape(`<a href="x">Tom & Jerry's</a>`))
}
