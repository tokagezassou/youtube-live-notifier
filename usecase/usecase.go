package usecase

import (
	"fmt"
	"strings"

	"github.com/tokagezassou/youtube-live-notifier/discord"
	"github.com/tokagezassou/youtube-live-notifier/repository"
	"github.com/tokagezassou/youtube-live-notifier/youtube"
)

type NotifierUsecase struct {
	youtubeClient *youtube.YouTubeClient
	db            *repository.MemoryDB
	discordClient *discord.WebhookClient
	roleID        string
}

func NewNotifierUsecase(
	yt *youtube.YouTubeClient,
	db *repository.MemoryDB,
	dc *discord.WebhookClient,
	roleID string,
) *NotifierUsecase {
	return &NotifierUsecase{
		youtubeClient: yt,
		db:            db,
		discordClient: dc,
		roleID:        roleID,
	}
}

func (u *NotifierUsecase) CheckAndNotify() (string, error) {
	lives, err := u.youtubeClient.FetchLatestLives()
	if err != nil {
		return "", fmt.Errorf("配信枠情報の取得に失敗しました: %w", err)
	}

	var messages []string
	roleMention := fmt.Sprintf("<@&%s>", u.roleID)
	messages = append(messages, fmt.Sprintf("%s\n📢 【最新の動画・配信枠一覧】", roleMention))

	var newItemsCount int

	for _, v := range lives {
		if u.db.IsNotified(v.ID) {
			continue
		}

		msg := fmt.Sprintf("タイトル: %s\nURL: %s", v.Title, v.URL)
		messages = append(messages, msg)

		u.db.MarkAsNotified(v.ID)
		newItemsCount++
	}

	if newItemsCount == 0 {
		return "新着の配信枠はありませんでした。", nil
	}

	finalMessage := strings.Join(messages, "\n\n")
	err = u.discordClient.SendMessage(finalMessage, u.roleID)
	if err != nil {
		return "", fmt.Errorf("Discordへの通知に失敗しました: %w", err)
	}

	return finalMessage, nil
}
