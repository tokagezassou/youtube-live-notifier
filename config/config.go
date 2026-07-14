package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	YouTubeChannelID  string
	YouTubeAPIKey     string
	DiscordWebhookURL string
	DiscordRoleID     string
}

func getEnv(key string) (string, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return "", fmt.Errorf("%s が設定されていません", key)
	}
	return v, nil
}

func Load() (*Config, error) {
	channelID, err := getEnv("YOUTUBE_CHANNEL_ID")
	if err != nil {
		return nil, err
	}

	apiKey, err := getEnv("YOUTUBE_API_KEY")
	if err != nil {
		return nil, err
	}

	webhookURL, err := getEnv("DISCORD_WEBHOOK_URL")
	if err != nil {
		return nil, err
	}

	roleID, err := getEnv("DISCORD_ROLE_ID")
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(webhookURL)
	if err != nil {
		return nil, fmt.Errorf("DISCORD_WEBHOOK_URL が不正です: %w", err)
	}
	if u.Scheme != "https" || u.Host != "discord.com" {
		return nil, fmt.Errorf(
			"DISCORD_WEBHOOK_URL は https://discord.com/... の形式である必要があります (scheme=%q host=%q)",
			u.Scheme, u.Host,
		)
	}

	return &Config{
		YouTubeChannelID:  channelID,
		YouTubeAPIKey:     apiKey,
		DiscordWebhookURL: webhookURL,
		DiscordRoleID:     roleID,
	}, nil
}
