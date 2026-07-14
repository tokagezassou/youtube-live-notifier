package discord

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type WebhookClient struct {
	webhookURL string
	http       *http.Client
}

func NewWebhookClient(rawURL string) *WebhookClient {
	return &WebhookClient{
		webhookURL: strings.TrimSpace(rawURL),
		http:       &http.Client{Timeout: 10 * time.Second},
	}
}

type allowedMentions struct {
	Parse []string `json:"parse"`
	Roles []string `json:"roles"`
}

type payload struct {
	Content         string          `json:"content"`
	AllowedMentions allowedMentions `json:"allowed_mentions"`
	Username        string          `json:"username,omitempty"`
	AvatarURL       string          `json:"avatar_url,omitempty"`
}

type webhookMessage struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
}

func (c *WebhookClient) SendMessage(message string, roleID string) error {
	p := payload{
		Content: message,
		AllowedMentions: allowedMentions{
			Parse: []string{"everyone", "users"},
			Roles: []string{roleID},
		},
		Username:  "もあぼっとちゃん",
		AvatarURL: "https://raw.githubusercontent.com/tokagezassou/youtube-live-notifier/main/assets/bot_icon.jpg",
	}

	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("JSONの生成に失敗しました: %w", err)
	}

	resp, err := c.http.Post(c.webhookURL+"?wait=true", "application/json", bytes.NewBuffer(body))
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			return fmt.Errorf("Webhookの送信に失敗しました (url=%s): %v",
				maskWebhookURL(c.webhookURL), urlErr.Err)
		}
		return fmt.Errorf("Webhookの送信に失敗しました: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Discord APIエラー: %s (%s)", resp.Status, string(respBody))
	}

	var msg webhookMessage
	if err := json.Unmarshal(respBody, &msg); err != nil {
		log.Printf("[Discord] 送信成功（レスポンス解析失敗）: %v", err)
		return nil
	}

	log.Printf("[Discord] 送信成功 message_id=%s channel_id=%s", msg.ID, msg.ChannelID)
	return nil
}

func maskWebhookURL(raw string) string {
	i := strings.LastIndex(raw, "/")
	if i < 0 {
		return "(invalid)"
	}
	return raw[:i+1] + "***"
}
