package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type WebhookClient struct {
	webhookURL string
	http       *http.Client
}

func NewWebhookClient(url string) *WebhookClient {
	return &WebhookClient{
		webhookURL: url,
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
