package usecase

import (
	"fmt"
	"strings"
	"time"

	"github.com/tokagezassou/youtube-live-notifier/discord"
	"github.com/tokagezassou/youtube-live-notifier/model"
	"github.com/tokagezassou/youtube-live-notifier/repository"
	"github.com/tokagezassou/youtube-live-notifier/youtube"
)

type NotifierUsecase struct {
	youtubeClient *youtube.Client
	db            *repository.MemoryDB
	discordClient *discord.WebhookClient
	roleID        string
}

func NewNotifierUsecase(
	yt *youtube.Client,
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
	var resultMessages []string

	newMsg, err := u.checkNewStreams()
	if err != nil {
		return "", fmt.Errorf("新着チェックエラー: %w", err)
	}
	resultMessages = append(resultMessages, "【新着チェック】 "+newMsg)

	startMsg, err := u.checkStreamStarted()
	if err != nil {
		return "", fmt.Errorf("開始チェックエラー: %w", err)
	}
	resultMessages = append(resultMessages, "【開始チェック】 "+startMsg)

	return strings.Join(resultMessages, "\n"), nil
}

func (u *NotifierUsecase) checkNewStreams() (string, error) {
	lives, err := u.youtubeClient.FetchLatestLives()
	if err != nil {
		return "", err
	}

	latestDBIDs := u.db.GetLatest15IDs()
	dbIDMap := make(map[string]bool)
	for _, id := range latestDBIDs {
		dbIDMap[id] = true
	}

	var newCandidateIDs []string
	candidateMap := make(map[string]model.LiveInfo)

	for _, v := range lives {
		if dbIDMap[v.ID] {
			continue
		}
		newCandidateIDs = append(newCandidateIDs, v.ID)
		candidateMap[v.ID] = v
	}

	if len(newCandidateIDs) == 0 {
		return "新着なし", nil
	}

	apiDetails, err := u.youtubeClient.FetchStreamDetails(newCandidateIDs)
	if err != nil {
		return "", err
	}

	var messages []string
	roleMention := fmt.Sprintf("<@&%s>", u.roleID)
	messages = append(messages, fmt.Sprintf("%s 📢 【新しい配信枠が作成されました！】", roleMention))

	notifiedCount := 0

	for _, id := range newCandidateIDs {
		info := candidateMap[id]
		apiInfo := apiDetails[id]

		info.Status = apiInfo.Status
		info.ScheduledStartTime = apiInfo.ScheduledStartTime

		isStream := (info.Status == model.StatusUpcoming || info.Status == model.StatusLive)

		doc := repository.StreamDocument{
			ID:                 info.ID,
			Title:              info.Title,
			URL:                info.URL,
			ScheduledStartTime: info.ScheduledStartTime,
			ShouldNotify:       isStream,
			CreatedAt:          time.Now(),
		}
		u.db.Save(doc)

		if info.Status == model.StatusUpcoming {
			messages = append(messages, fmt.Sprintf("タイトル: %s\nURL: %s", info.Title, info.URL))
			notifiedCount++
		}
	}

	if notifiedCount > 0 {
		finalMessage := strings.Join(messages, "\n\n")
		_ = u.discordClient.SendMessage(finalMessage, u.roleID)
		return fmt.Sprintf("%d件の新しい配信枠を通知しました", notifiedCount), nil
	}

	return "新着は動画のみのため通知スキップ", nil
}

func (u *NotifierUsecase) checkStreamStarted() (string, error) {
	targets := u.db.GetShouldNotifyStreams()
	if len(targets) == 0 {
		return "監視対象なし", nil
	}

	var checkIDs []string
	now := time.Now()

	for _, t := range targets {
		if t.ScheduledStartTime.IsZero() {
			checkIDs = append(checkIDs, t.ID)
			continue
		}

		if now.After(t.ScheduledStartTime.Add(90 * time.Minute)) {
			t.ShouldNotify = false
			u.db.Save(t)
			continue
		}

		if now.After(t.ScheduledStartTime.Add(-5*time.Minute)) &&
			now.Before(t.ScheduledStartTime.Add(90*time.Minute)) {
			checkIDs = append(checkIDs, t.ID)
		}
	}

	if len(checkIDs) == 0 {
		return "時間内の監視対象なし", nil
	}

	apiDetails, err := u.youtubeClient.FetchStreamDetails(checkIDs)
	if err != nil {
		return "", err
	}

	notifiedCount := 0
	for _, id := range checkIDs {
		var doc repository.StreamDocument
		for _, t := range targets {
			if t.ID == id {
				doc = t
				break
			}
		}

		apiInfo, exists := apiDetails[id]

		if !exists || apiInfo.Status == model.StatusNone {
			doc.ShouldNotify = false
			u.db.Save(doc)
			continue
		}

		if apiInfo.Status == model.StatusLive {
			msg := fmt.Sprintf("<@&%s> 🎥 【配信が開始されました！】\nタイトル: %s\nURL: %s", u.roleID, doc.Title, doc.URL)
			u.discordClient.SendMessage(msg, u.roleID)

			doc.ShouldNotify = false
			u.db.Save(doc)
			notifiedCount++
		}
	}

	return fmt.Sprintf("%d件の開始状況をチェックしました（通知: %d件）", len(checkIDs), notifiedCount), nil
}
