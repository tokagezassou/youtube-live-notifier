package config

import (
	"fmt"
	"os"
)

type Config struct {
	DiscordWebhookURL      string
	DiscordAdminWebhookURL string
	DiscordRoleID          string
	OAuthClientID          string
	OAuthClientSecret      string
	OAuthRefreshToken      string
}

func Load() (*Config, error) {
	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	if webhookURL == "" {
		return nil, fmt.Errorf("DISCORD_WEBHOOK_URL が設定されていません")
	}

	adminWebhookURL := os.Getenv("DISCORD_ADMIN_WEBHOOK_URL")
	if adminWebhookURL == "" {
		return nil, fmt.Errorf("DISCORD_ADMIN_WEBHOOK_URL が設定されていません")
	}

	roleID := os.Getenv("DISCORD_ROLE_ID")
	if roleID == "" {
		return nil, fmt.Errorf("DISCORD_ROLE_ID が設定されていません")
	}

	clientID := os.Getenv("OAUTH_CLIENT_ID")
	if clientID == "" {
		return nil, fmt.Errorf("OAUTH_CLIENT_ID が設定されていません")
	}

	clientSecret := os.Getenv("OAUTH_CLIENT_SECRET")
	if clientSecret == "" {
		return nil, fmt.Errorf("OAUTH_CLIENT_SECRET が設定されていません")
	}

	oAuthRefreshToken := os.Getenv("OAUTH_REFRESH_TOKEN")
	if oAuthRefreshToken == "" {
		return nil, fmt.Errorf("OAUTH_REFRESH_TOKEN が設定されていません")
	}

	return &Config{
		DiscordWebhookURL:      webhookURL,
		DiscordAdminWebhookURL: adminWebhookURL,
		DiscordRoleID:          roleID,
		OAuthClientID:          clientID,
		OAuthClientSecret:      clientSecret,
		OAuthRefreshToken:      oAuthRefreshToken,
	}, nil
}
