// Package notify sends operational notices to the ops Telegram chat (App\Services\TelegramNotifier).
//
// Best-effort by design: a Telegram outage never fails the request that triggered the
// notice — every failure is logged and swallowed. The notifier is a no-op unless both
// TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID are set. Only ids and names are ever sent, never
// health data.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/platform/config"
)

// DefaultAPIBase is the Bot API endpoint.
const DefaultAPIBase = "https://api.telegram.org"

// Telegram is the notifier. The zero value and nil are disabled no-ops.
type Telegram struct {
	token   string
	chatID  string
	timeout time.Duration
	apiBase string
	client  *http.Client
	logger  *slog.Logger
}

// NewTelegram builds the notifier from TELEGRAM_* (config.Telegram). client nil → a default
// client; the per-message timeout is cfg.Timeout (TELEGRAM_TIMEOUT, default 5 s).
func NewTelegram(cfg config.Telegram, client *http.Client, logger *slog.Logger) *Telegram {
	if client == nil {
		client = &http.Client{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Telegram{
		token: strings.TrimSpace(cfg.BotToken), chatID: strings.TrimSpace(cfg.ChatID),
		timeout: timeout, apiBase: DefaultAPIBase, client: client, logger: logger,
	}
}

// WithAPIBase points the notifier at another Bot API host (tests).
func (t *Telegram) WithAPIBase(base string) *Telegram {
	t.apiBase = strings.TrimRight(base, "/")
	return t
}

// Enabled reports whether both the token and the chat id are configured.
func (t *Telegram) Enabled() bool { return t != nil && t.token != "" && t.chatID != "" }

// Send posts an HTML-formatted message and reports whether Telegram accepted it. It never
// returns an error: failures are logged (without the token) and false is returned.
func (t *Telegram) Send(ctx context.Context, text string) bool {
	if !t.Enabled() {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	body, _ := json.Marshal(map[string]any{
		"chat_id":                  t.chatID,
		"text":                     text,
		"parse_mode":               "HTML",
		"disable_web_page_preview": true,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/bot%s/sendMessage", t.apiBase, t.token), bytes.NewReader(body))
	if err != nil {
		t.logger.WarnContext(ctx, "Telegram notification error", slog.String("message", "build request failed"))
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := t.client.Do(req)
	if err != nil {
		// The URL carries the bot token; never log the raw error text.
		t.logger.WarnContext(ctx, "Telegram notification error", slog.String("message", "request failed"))
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true
	}
	t.logger.WarnContext(ctx, "Telegram notification failed", slog.Int("status", resp.StatusCode))
	return false
}

// SendAsync is Send in the background (detached from the request context), so a slow or
// unreachable Telegram never delays the API response. done, when non-nil, receives the result.
func (t *Telegram) SendAsync(text string, done func(bool)) {
	if !t.Enabled() {
		if done != nil {
			done(false)
		}
		return
	}
	go func() {
		ok := t.Send(context.Background(), text)
		if done != nil {
			done(ok)
		}
	}()
}

var htmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#039;")

// Escape is Laravel's e(): htmlspecialchars($s, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8', true).
func Escape(s string) string {
	return htmlEscaper.Replace(strings.ToValidUTF8(s, "�"))
}
