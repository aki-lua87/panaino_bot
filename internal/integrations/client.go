package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxBody = 1 << 20

type Client struct {
	HTTP          *http.Client
	WeatherURL    string
	RandomCardURL string
	OthelloAPI    string
	OthelloURL    string
}

func New() *Client {
	return &Client{
		HTTP: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Scheme != "https" || len(via) >= 10 {
				return errors.New("安全でないまたは多すぎるリダイレクトです")
			}
			return nil
		}},
		WeatherURL:    "https://weather.tsukumijima.net/api/forecast",
		RandomCardURL: "https://api.scryfall.com/cards/random",
		OthelloAPI:    "https://el-ement.com/blog/wp-content/uploads/moonrev/api.php",
		OthelloURL:    "https://el-ement.com/blog/wp-content/uploads/moonrev/",
	}
}

func (c *Client) request(ctx context.Context, method, endpoint, contentType string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, errors.New("リクエストの作成に失敗しました")
	}
	req.Header.Set("User-Agent", "panaino-bot/2 (+https://github.com/aki-lua87/panaino_bot)")
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		// url.Error includes query strings, which can contain API credentials or chat text.
		return nil, errors.New("外部サービスとの通信に失敗しました")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("外部サービスがHTTP %dを返しました", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return nil, errors.New("外部サービスの応答を読み取れません")
	}
	if len(b) > maxBody {
		return nil, errors.New("外部サービスの応答が大きすぎます")
	}
	return b, nil
}

func decode(b []byte, out any) error {
	if err := json.Unmarshal(b, out); err != nil {
		return errors.New("外部サービスのJSONが不正です")
	}
	return nil
}

func withQuery(endpoint string, values url.Values) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", errors.New("API URLが不正です")
	}
	q := u.Query()
	for k, vs := range values {
		q.Del(k)
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (c *Client) Conversation(ctx context.Context, endpoint, key string) (string, error) {
	u, err := withQuery(endpoint, url.Values{"key": {key}})
	if err != nil {
		return "", err
	}
	b, err := c.request(ctx, http.MethodGet, u, "", nil)
	if err != nil {
		return "", err
	}
	var response struct {
		Result bool   `json:"response"`
		Text   string `json:"text"`
	}
	if err := decode(b, &response); err != nil {
		return "", err
	}
	if !response.Result {
		return "", nil
	}
	return response.Text, nil
}

func (c *Client) Weather(ctx context.Context, city string) (string, error) {
	u, err := withQuery(c.WeatherURL, url.Values{"city": {city}})
	if err != nil {
		return "", err
	}
	b, err := c.request(ctx, http.MethodGet, u, "", nil)
	if err != nil {
		return "", err
	}
	var data struct {
		Title       string `json:"title"`
		Link        string `json:"link"`
		Description struct {
			Text string `json:"text"`
		} `json:"description"`
	}
	if err := decode(b, &data); err != nil {
		return "", err
	}
	if strings.TrimSpace(data.Description.Text) == "" {
		return "", errors.New("天気情報がありません")
	}
	return data.Title + "\n" + data.Description.Text + "\n" + data.Link, nil
}

func (c *Client) RandomCard(ctx context.Context) (string, error) {
	u, err := withQuery(c.RandomCardURL, url.Values{"q": {"lang:ja"}})
	if err != nil {
		return "", err
	}
	b, err := c.request(ctx, http.MethodGet, u, "", nil)
	if err != nil {
		return "", err
	}
	var card struct {
		Name        string            `json:"name"`
		PrintedName string            `json:"printed_name"`
		URI         string            `json:"scryfall_uri"`
		Images      map[string]string `json:"image_uris"`
		Faces       []struct {
			Images map[string]string `json:"image_uris"`
		} `json:"card_faces"`
	}
	if err := decode(b, &card); err != nil {
		return "", err
	}
	name := card.PrintedName
	if name == "" {
		name = card.Name
	}
	image := card.Images["normal"]
	if image == "" && len(card.Faces) > 0 {
		image = card.Faces[0].Images["normal"]
	}
	if image == "" {
		image = card.URI
	}
	if name == "" || !httpsLink(image) {
		return "", errors.New("カード情報がありません")
	}
	return name + "\n" + image, nil
}

func httpsLink(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil
}

func (c *Client) Othello(ctx context.Context) (string, error) {
	b, err := c.request(ctx, http.MethodPost, c.OthelloAPI, "application/x-www-form-urlencoded", strings.NewReader("m=open"))
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(string(b))
	if id == "" || len(id) > 256 || strings.ContainsAny(id, "<>\r\n \t") {
		return "", errors.New("オセロの部屋情報が不正です")
	}
	u, err := url.Parse(c.OthelloURL)
	if err != nil {
		return "", errors.New("オセロのURLが不正です")
	}
	u.Fragment = id
	return u.String(), nil
}

func Hareruya(cardName string) string {
	u, _ := url.Parse("https://www.hareruyamtg.com/ja/products/search?sort=price&order=ASC&category=&cardset=&colorsType=0&cardtypesType=0&format=&illustrator=&stock=1")
	q := u.Query()
	q.Set("product", cardName)
	u.RawQuery = q.Encode()
	return u.String()
}

func (c *Client) CoatOfArms(ctx context.Context, endpoint string) (string, error) {
	b, err := c.request(ctx, http.MethodGet, endpoint, "", nil)
	if err != nil {
		return "", err
	}
	var data struct {
		UpdateTime string
		StringList []string
	}
	if err := decode(b, &data); err != nil {
		return "", err
	}
	if len(data.StringList) == 0 {
		return "", errors.New("紋章キャンペーンの情報がありません")
	}
	return "今週の覇者の紋章キャンペーンは以下のとおりです...\n\n" + strings.Join(data.StringList, "\n") + "\n(データ更新 : " + data.UpdateTime + ")", nil
}

func (c *Client) Emergency(ctx context.Context, endpoint, label string, date time.Time) (string, error) {
	data, _ := json.Marshal(struct {
		Date string `json:"EventDate"`
	}{date.Format("20060102")})
	b, err := c.request(ctx, http.MethodPost, endpoint, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return "", err
	}
	var quests []struct {
		EventName    string
		Hour, Minute int
	}
	if err := decode(b, &quests); err != nil {
		return "", err
	}
	lines := []string{label + "の緊急クエストは...."}
	if len(quests) == 0 {
		lines = append(lines, "予定なしです。")
	}
	for _, q := range quests {
		if q.Hour < 0 || q.Hour > 23 || q.Minute < 0 || q.Minute > 59 {
			return "", errors.New("緊急クエストの時刻が不正です")
		}
		lines = append(lines, fmt.Sprintf("%02d:%02d %s", q.Hour, q.Minute, q.EventName))
	}
	return strings.Join(lines, "\n"), nil
}
