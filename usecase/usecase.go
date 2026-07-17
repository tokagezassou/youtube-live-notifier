package usecase

import (
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
	db            *repository.FirestoreDB
	discordClient *discord.WebhookClient
	roleID        string
}

func NewNotifierUsecase(
	yt *youtube.Client,
	// db *repository.MemoryDB,
	db *repository.FirestoreDB,
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
	return u.processNewStreams(lives)
}

func (u *NotifierUsecase) processNewStreams(lives []model.LiveInfo) (string, error) {
	ids := make([]string, 0, len(lives))
	for _, l := range lives {
		ids = append(ids, l.ID)
	}

	existing, err := u.db.GetExistingIDs(ids)
	if err != nil {
		return "", fmt.Errorf("既存ID取得エラー: %w", err)
	}

	var newCandidateIDs []string
	candidateMap := make(map[string]model.LiveInfo)

	for _, l := range lives {
		if existing[l.ID] {
			continue
		}
		newCandidateIDs = append(newCandidateIDs, l.ID)
		candidateMap[l.ID] = l
	}

	if len(newCandidateIDs) == 0 {
		return "新着なし", nil
	}

	apiDetails, err := u.youtubeClient.FetchStreamDetails(newCandidateIDs)
	if err != nil {
		return "", err
	}

	for _, id := range newCandidateIDs {
		info := candidateMap[id]

		apiInfo, ok := apiDetails[id]
		if !ok {
			log.Printf("詳細が取得できなかったため新着処理をスキップ (ID: %s)", id)
			continue
		}

		info.Status = apiInfo.Status
		info.ScheduledStartTime = apiInfo.ScheduledStartTime

		if info.Status != model.StatusUpcoming && info.Status != model.StatusLive {
			continue
		}
		if info.Status == model.StatusUpcoming && info.ScheduledStartTime.IsZero() {
			continue
		}

		isStream := (info.Status == model.StatusUpcoming || info.Status == model.StatusLive)

		doc := repository.StreamDocument{
			ID:                 info.ID,
			Title:              info.Title,
			URL:                info.URL,
			ScheduledStartTime: info.ScheduledStartTime,
			ShouldNotify:       isStream,
			CreatedAt:          time.Now(),
		}

		if err := u.db.Save(doc); err != nil {
			log.Printf("保存失敗のため通知スキップ (ID: %s): %v", info.ID, err)
			continue
		}

		if info.Status == model.StatusUpcoming {
			message := u.newStreamMessage(info)
			if err := u.discordClient.SendMessage(message, u.roleID); err != nil {
				log.Printf("枠立て通知エラー (ID: %s): %v", info.ID, err)
			} else {
				log.Printf("枠立て通知を送信 (ID: %s, 予定: %s)", info.ID, info.ScheduledStartTime)
			}
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
	targets, err := u.db.GetShouldNotifyStreams()
	if err != nil {
		return "", fmt.Errorf("通知対象取得エラー: %w", err)
	}
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
					log.Printf("保存失敗のため通知スキップ (ID: %s): %v", t.ID, err)
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
				log.Printf("保存失敗のため通知スキップ (ID: %s): %v", t.ID, err)
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
		if !exists {
			log.Printf("警告: APIから動画データが返されませんでした (ID: %s)。次回の周期で再試行します。", id)
			continue
		}

		if apiInfo.Status == model.StatusNone {
			doc.ShouldNotify = false
			err := u.db.Save(doc)
			if err != nil {
				log.Printf("保存失敗のため通知スキップ (ID: %s): %v", doc.ID, err)
				continue
			}
			continue
		}

		if apiInfo.Status == model.StatusLive {
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

func (u *NotifierUsecase) SearchAndNotify() (string, error) {
	lives, err := u.youtubeClient.SearchUpcomingLives()
	if err != nil {
		return "", fmt.Errorf("search.list エラー: %w", err)
	}

	msg, err := u.processNewStreams(lives)
	if err != nil {
		return "", fmt.Errorf("search経由の新着チェックエラー: %w", err)
	}

	return "【search経由】 " + msg, nil
}
