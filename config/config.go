package config

import (
	"fmt"
	"os"
)

type Config struct {
	ChannelID string
}

func Load() (*Config, error) {
	channelID := os.Getenv("YOUTUBE_CHANNEL_ID")
	if channelID == "" {
		return nil, fmt.Errorf("YOUTUBE_CHANNEL_ID が設定されていません")
	}

	return &Config{
		ChannelID: channelID,
	}, nil
}
