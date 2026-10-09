package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestConversationPreservesJapaneseAndExistingQuery(t *testing.T) {
	key := "お昼ではない<&?= 晴れる屋"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != key || r.URL.Query().Get("existing") != "keep" {
			t.Errorf("input changed: %v", r.URL.Query())
		}
		fmt.Fprint(w, `{"response":true,"text":"んまー"}`)
	}))
	defer server.Close()
	text, err := New().Conversation(context.Background(), server.URL+"?existing=keep", key)
	if err != nil || text != "んまー" {
		t.Fatalf("text=%q err=%v", text, err)
	}
}

func TestExternalFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"server error", 500, `{"response":true,"text":"wrong"}`},
		{"rate limit", 429, "retry later"},
		{"HTML", 200, "<!DOCTYPE html> login"},
		{"invalid JSON", 200, "secret response"},
		{"oversized", 200, strings.Repeat("x", maxBody+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); fmt.Fprint(w, tc.body) }))
			defer server.Close()
			text, err := New().Conversation(context.Background(), server.URL, "secret-query")
			if err == nil || text != "" {
				t.Fatalf("bad response accepted: text=%q err=%v", text, err)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatal("error contains private data")
			}
		})
	}
}

func TestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := New().Conversation(ctx, server.URL, "secret")
	if err == nil {
		t.Fatal("timeout ignored")
	}
}

func TestHareruyaDoesNotTreatCardNameAsQueryParameters(t *testing.T) {
	name := "晴れる屋の屋根 & stock=0 # 日本語"
	u, err := url.Parse(Hareruya(name))
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("product") != name || u.Query().Get("stock") != "1" || u.Fragment != "" {
		t.Fatalf("incorrect URL: %s", u)
	}
}

func TestWeatherAndDoubleFacedCard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/weather" {
			fmt.Fprint(w, `{"title":"東京","description":{"text":"晴れ"},"link":"https://example.com/weather"}`)
		} else {
			fmt.Fprint(w, `{"name":"English","printed_name":"日本語","card_faces":[{"image_uris":{"normal":"https://example.com/front.png"}}]}`)
		}
	}))
	defer server.Close()
	c := New()
	c.WeatherURL, c.RandomCardURL = server.URL+"/weather", server.URL+"/card"
	text, err := c.Weather(context.Background(), "130010")
	if err != nil || !strings.Contains(text, "晴れ") {
		t.Fatalf("weather=%q err=%v", text, err)
	}
	text, err = c.RandomCard(context.Background())
	if err != nil || text != "日本語\nhttps://example.com/front.png" {
		t.Fatalf("card=%q err=%v", text, err)
	}
}

func TestEmergencyDateAndEmptySchedule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
			t.Error("wrong method or content type")
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload["EventDate"] != "20260101" {
			t.Errorf("wrong date: %v", payload)
		}
		fmt.Fprint(w, `[]`)
	}))
	defer server.Close()
	text, err := New().Emergency(context.Background(), server.URL, "今日", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || !strings.Contains(text, "予定なし") {
		t.Fatalf("schedule=%q err=%v", text, err)
	}
}

func TestOthelloRejectsHTMLRoom(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "<html>login</html>") }))
	defer server.Close()
	c := New()
	c.OthelloAPI = server.URL
	if _, err := c.Othello(context.Background()); err == nil {
		t.Fatal("HTML response accepted as room ID")
	}
}
