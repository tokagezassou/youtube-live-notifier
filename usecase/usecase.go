package usecase

import (
	"fmt"
	"strings"

	"github.com/tokagezassou/youtube-live-notifier/youtube"
)

type NotifierUsecase struct {
	youtubeClient *youtube.YouTubeClient
}

func NewNotifierUsecase(yt *youtube.YouTubeClient) *NotifierUsecase {
	return &NotifierUsecase{
		youtubeClient: yt,
	}
}

func (u *NotifierUsecase) CheckAndNotify() (string, error) {
	lives, err := u.youtubeClient.FetchLatestLives()
	if err != nil {
		return "", fmt.Errorf("配信枠情報の取得に失敗しました: %w", err)
	}

	var messages []string
	messages = append(messages, "📢 【最新の動画・配信枠一覧】")

	for _, v := range lives {
		msg := fmt.Sprintf("タイトル: %s\nURL: %s", v.Title, v.URL)
		messages = append(messages, msg)
	}

	return strings.Join(messages, "\n\n"), nil
}
