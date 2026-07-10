package usecase

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tokagezassou/youtube-live-notifier/discord"
	"github.com/tokagezassou/youtube-live-notifier/model"
	"github.com/tokagezassou/youtube-live-notifier/repository"
	"github.com/tokagezassou/youtube-live-notifier/youtube"
)

type NotifierUsecase struct {
	youtubeClient *youtube.Client
	// db            *repository.MemoryDB
	db                 *repository.FirestoreDB
	discordClient      *discord.WebhookClient
	discordAdminClient *discord.WebhookClient
	roleID             string
}

func NewNotifierUsecase(
	yt *youtube.Client,
	// db *repository.MemoryDB,
	db *repository.FirestoreDB,
	dc *discord.WebhookClient,
	dac *discord.WebhookClient,
	roleID string,
) *NotifierUsecase {
	return &NotifierUsecase{
		youtubeClient:      yt,
		db:                 db,
		discordClient:      dc,
		discordAdminClient: dac,
		roleID:             roleID,
	}
}

func (u *NotifierUsecase) CheckAndNotify() (string, error) {
	var resultMessages []string

	newMsg, err := u.checkNewStreams()
	if err != nil {
		u.notifyIfAuthError(err)
		return "", fmt.Errorf("新着チェックエラー: %w", err)
	}
	resultMessages = append(resultMessages, "【新着チェック】 "+newMsg)

	startMsg, err := u.checkStreamStarted()
	if err != nil {
		u.notifyIfAuthError(err)
		return "", fmt.Errorf("開始チェックエラー: %w", err)
	}
	resultMessages = append(resultMessages, "【開始チェック】 "+startMsg)

	return strings.Join(resultMessages, "\n"), nil
}

func (u *NotifierUsecase) notifyIfAuthError(err error) {
	var authErr *youtube.AuthError
	if errors.As(err, &authErr) {
		msg := "⚠️ YouTube認証が切れました。リフレッシュトークンの再取得が必要です。"
		if sendErr := u.discordAdminClient.SendMessage(msg, ""); sendErr != nil {
			log.Printf("認証エラー通知の送信に失敗: %v", sendErr)
		}
	}
}

func (u *NotifierUsecase) checkNewStreams() (string, error) {
	lives, err := u.youtubeClient.FetchUpcomingLives()
	if err != nil {
		return "", err
	}

	ids := make([]string, 0, len(lives))
	for _, l := range lives {
		ids = append(ids, l.ID)
	}
	if len(ids) == 0 {
		return "新着なし", nil
	}
	existing := u.db.GetExistingIDs(ids)

	for _, info := range lives {
		if existing[info.ID] {
			continue
		}

		if info.PrivacyStatus != model.StatusPublic {
			continue
		}
		if info.ScheduledStartTime.IsZero() {
			continue
		}

		doc := repository.StreamDocument{
			ID:                 info.ID,
			Title:              info.Title,
			URL:                info.URL,
			ScheduledStartTime: info.ScheduledStartTime,
			ShouldNotify:       true,
			CreatedAt:          time.Now(),
		}

		if err := u.db.Save(doc); err != nil {
			log.Printf("保存失敗のため通知スキップ (ID: %s): %v", info.ID, err)
			continue
		}

		message := u.newStreamMessage(info)
		if err := u.discordClient.SendMessage(message, u.roleID); err != nil {
			fmt.Printf("Discord通知エラー (ID: %s): %v\n", info.ID, err)
		}
	}

	return "新着を確認しました", nil
}

func (u *NotifierUsecase) newStreamMessage(l model.LiveInfo) string {
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	japaneseWeekdays := []string{"日", "月", "火", "水", "木", "金", "土"}

	jstTime := l.ScheduledStartTime.In(jst)
	weekdayIdx := jstTime.Weekday()
	weekdayStr := japaneseWeekdays[weekdayIdx]

	now := time.Now().In(jst)
	var dateStr string
	if jstTime.Year() == now.Year() &&
		jstTime.Month() == now.Month() &&
		jstTime.Day() == now.Day() {
		dateStr = "今日"
	} else {
		dateStr = fmt.Sprintf("%d/%d(%s)", jstTime.Month(), jstTime.Day(), weekdayStr)
	}

	var timeStr string
	if jstTime.Minute() == 0 {
		timeStr = fmt.Sprintf("%sの%d時", dateStr, jstTime.Hour())
	} else {
		timeStr = fmt.Sprintf("%sの%d:%02d", dateStr, jstTime.Hour(), jstTime.Minute())
	}

	message := fmt.Sprintf("<@&%s>\n", u.roleID)
	message += timeStr + "から配信予定の枠立てました！高評価よろしくもーあちっ！\n"
	message += l.URL

	return message
}

func (u *NotifierUsecase) checkStreamStarted() (string, error) {
	targets := u.db.GetShouldNotifyStreams()
	if len(targets) == 0 {
		return "監視対象なし", nil
	}

	now := time.Now()
	var checkIDs []string
	for _, t := range targets {
		if t.ScheduledStartTime.IsZero() {
			if now.After(t.CreatedAt.Add(250 * time.Minute)) {
				t.ShouldNotify = false
				err := u.db.Save(t)
				if err != nil {
					log.Printf("保存失敗 (ID: %s): %v", t.ID, err)
					continue
				}
				continue
			}

			checkIDs = append(checkIDs, t.ID)
			continue
		}

		if now.After(t.ScheduledStartTime.Add(250 * time.Minute)) {
			t.ShouldNotify = false
			err := u.db.Save(t)
			if err != nil {
				log.Printf("保存失敗 (ID: %s): %v", t.ID, err)
				continue
			}
			continue
		}

		if now.After(t.ScheduledStartTime.Add(-130*time.Minute)) &&
			now.Before(t.ScheduledStartTime.Add(250*time.Minute)) {
			checkIDs = append(checkIDs, t.ID)
		}
	}

	if len(checkIDs) == 0 {
		return "時間内の監視対象なし", nil
	}

	activeLives, err := u.youtubeClient.FetchActiveLives()
	if err != nil {
		return "", err
	}
	activeMap := make(map[string]model.LiveInfo)
	for _, l := range activeLives {
		activeMap[l.ID] = l
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

		if _, isActive := activeMap[id]; isActive {
			doc.ShouldNotify = false
			if err := u.db.Save(doc); err != nil {
				log.Printf("保存失敗のため通知スキップ (ID: %s): %v", doc.ID, err)
				continue
			}

			message := fmt.Sprintf("<@&%s>\n", u.roleID)
			message += "配信開始しました！今日も「ただいまもあち」待っとるけーん💖\n"
			message += doc.URL
			if err := u.discordClient.SendMessage(message, u.roleID); err != nil {
				log.Printf("Discord通知エラー (ID: %s): %v", doc.ID, err)
			}
			notifiedCount++
		}
	}

	return fmt.Sprintf("%d件の開始状況をチェックしました（通知: %d件）", len(checkIDs), notifiedCount), nil
}
