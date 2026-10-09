package features

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aki-lua87/marumeshi_bot/internal/config"
	"github.com/aki-lua87/marumeshi_bot/internal/integrations"
)

func TestGuildSpecificConversation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"response":true,"text":%q}`, r.URL.Path)
	}))
	defer server.Close()
	e := Engine{API: integrations.New()}
	for _, path := range []string{"/guild-a", "/guild-b"} {
		text, err := e.Reply(context.Background(), config.Guild{SpreadsheetAPI: server.URL + path}, "こんにちは", time.Now())
		if err != nil || text != path {
			t.Fatalf("guild routing failed: text=%q err=%v", text, err)
		}
	}
}

func TestFallbackAndAPIError(t *testing.T) {
	e := Engine{API: integrations.New()}
	text, err := e.Reply(context.Background(), config.Guild{FallbackReply: "まるめし"}, "こんにちは", time.Now())
	if err != nil || text != "まるめし" {
		t.Fatalf("production fallback changed: %q %v", text, err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	defer server.Close()
	text, err = e.Reply(context.Background(), config.Guild{FallbackReply: "まるめし", SpreadsheetAPI: server.URL}, "こんにちは", time.Now())
	if err == nil || text != "まるめし" {
		t.Fatalf("API failure must retain fallback: %q %v", text, err)
	}
}

func TestNewYearUsesJapanDate(t *testing.T) {
	e := Engine{API: integrations.New()}
	g := config.Guild{FallbackReply: "通常"}
	for _, tc := range []struct {
		utc    string
		active bool
	}{
		{"2025-12-31T14:59:59Z", false},
		{"2025-12-31T15:00:00Z", true},
		{"2026-01-03T14:59:59Z", true},
		{"2026-01-03T15:00:00Z", false},
	} {
		now, _ := time.Parse(time.RFC3339, tc.utc)
		text, err := e.Reply(context.Background(), g, "あけおめ", now)
		if err != nil || (text == "あけおめし") != tc.active {
			t.Fatalf("UTC=%s reply=%q err=%v", tc.utc, text, err)
		}
	}
}

func TestHareruyaPrefixPreservesFirstCharacters(t *testing.T) {
	e := Engine{API: integrations.New()}
	name := "晴れる屋の屋根 & #"
	text, err := e.Reply(context.Background(), config.Guild{}, "晴れる屋 "+name, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(text)
	if u.Query().Get("product") != name {
		t.Fatalf("card name trimmed incorrectly: %q", u.Query().Get("product"))
	}
}

func TestMissingIntegrationIsExplicit(t *testing.T) {
	e := Engine{API: integrations.New()}
	for _, command := range []string{"今日の緊急", "明日の緊急", "覇者", "セリフ"} {
		text, err := e.Reply(context.Background(), config.Guild{}, command, time.Now())
		if err != nil || !strings.Contains(text, "設定") {
			t.Fatalf("command=%s text=%q err=%v", command, text, err)
		}
	}
}
