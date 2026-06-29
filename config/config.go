package config

import (
	"fmt"
	"os"
)

type Config struct {
	YouTubeChannelID  string
	DiscordWebhookURL string
	DiscordRoleID     string
}

func Load() (*Config, error) {
	channelID := os.Getenv("YOUTUBE_CHANNEL_ID")
	if channelID == "" {
		return nil, fmt.Errorf("YOUTUBE_CHANNEL_ID が設定されていません")
	}

	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	if webhookURL == "" {
		return nil, fmt.Errorf("DISCORD_WEBHOOK_URL が設定されていません")
	}

	roleID := os.Getenv("DISCORD_ROLE_ID")
	if roleID == "" {
		return nil, fmt.Errorf("DISCORD_ROLE_ID が設定されていません")
	}

	return &Config{
		YouTubeChannelID:  channelID,
		DiscordWebhookURL: webhookURL,
		DiscordRoleID:     roleID,
	}, nil
}
