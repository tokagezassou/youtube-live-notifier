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
}

func (c *WebhookClient) SendMessage(message string, roleID string) error {
	p := payload{
		Content: message,
		AllowedMentions: allowedMentions{
			Parse: []string{"everyone", "users"},
			Roles: []string{roleID},
		},
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
