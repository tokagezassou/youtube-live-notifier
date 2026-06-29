package usecase

import (
	"fmt"
	"strings"

	"github.com/tokagezassou/youtube-live-notifier/repository"
	"github.com/tokagezassou/youtube-live-notifier/youtube"
)

type NotifierUsecase struct {
	youtubeClient *youtube.YouTubeClient
	db            *repository.MemoryDB // データベースを保持
}

func NewNotifierUsecase(
	yt *youtube.YouTubeClient,
	db *repository.MemoryDB,
) *NotifierUsecase {
	return &NotifierUsecase{
		youtubeClient: yt,
		db:            db,
	}
}

func (u *NotifierUsecase) CheckAndNotify() (string, error) {
	lives, err := u.youtubeClient.FetchLatestLives()
	if err != nil {
		return "", fmt.Errorf("配信枠情報の取得に失敗しました: %w", err)
	}

	var messages []string
	messages = append(messages, "📢 【最新の動画・配信枠一覧】")

	var newItemsCount int // 新着の件数をカウント

	// 取得した15件をループで回し、差分チェックを行う
	for _, v := range lives {
		// すでに通知済みのIDならスキップ
		if u.db.IsNotified(v.ID) {
			continue
		}

		// 新しいIDだったので、メッセージを作成してDBに記憶させる
		msg := fmt.Sprintf("タイトル: %s\nURL: %s", v.Title, v.URL)
		messages = append(messages, msg)

		u.db.MarkAsNotified(v.ID)
		newItemsCount++
	}

	// もし新着が1件もなければ、その旨を返す
	if newItemsCount == 0 {
		return "新着の配信枠はありませんでした。", nil
	}

	return strings.Join(messages, "\n\n"), nil
}
