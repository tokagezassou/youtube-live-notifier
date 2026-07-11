package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type WebhookClient struct {
	webhookURL string
}

func NewWebhookClient(url string) *WebhookClient {
	return &WebhookClient{
		webhookURL: url,
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

func (c *WebhookClient) SendMessage(message string, roleID string) error {
	roles := []string{}
	if roleID != "" {
		roles = []string{roleID}
	}

	p := payload{
		Content: message,
		AllowedMentions: allowedMentions{
			Parse: []string{"everyone", "users"},
			Roles: roles,
		},
		Username:  "もあぼっとちゃん",
		AvatarURL: "https://raw.githubusercontent.com/tokagezassou/youtube-live-notifier/main/assets/bot_icon.jpg",
	}

	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("JSONの生成に失敗しました: %w", err)
	}

	resp, err := http.Post(c.webhookURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("Webhookの送信に失敗しました: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Discord APIエラー: %s", resp.Status)
	}

	return nil
}
