package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/tokagezassou/youtube-live-notifier/model"
)

const youtubeReadonlyScope = "https://www.googleapis.com/auth/youtube.readonly"

type Client struct {
	httpClient *http.Client
}

func NewClient(
	ctx context.Context,
	clientID string,
	clientSecret string,
	refreshToken string,
) *Client {
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{youtubeReadonlyScope},
	}
	token := &oauth2.Token{RefreshToken: refreshToken}
	return &Client{
		httpClient: cfg.Client(ctx, token),
	}
}

type broadcastListResponse struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			Title              string `json:"title"`
			ScheduledStartTime string `json:"scheduledStartTime"`
		} `json:"snippet"`
		Status struct {
			LifeCycleStatus string `json:"lifeCycleStatus"`
			PrivacyStatus   string `json:"privacyStatus"`
		} `json:"status"`
	} `json:"items"`
	NextPageToken string `json:"nextPageToken"`
}

func (c *Client) FetchUpcomingLives() ([]model.LiveInfo, error) {
	return c.fetchBroadcasts("upcoming")
}

func (c *Client) FetchActiveLives() ([]model.LiveInfo, error) {
	return c.fetchBroadcasts("active")
}

func (c *Client) fetchBroadcasts(broadcastStatus string) ([]model.LiveInfo, error) {
	apiURL := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/liveBroadcasts?part=snippet,status&broadcastStatus=%s&maxResults=50",
		broadcastStatus,
	)

	resp, err := c.httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("liveBroadcasts の取得に失敗しました: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("OAuth 認証に失敗しました（トークンの再取得が必要な可能性があります）: %s", resp.Status)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("YouTube API エラー (ステータス: %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp broadcastListResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("JSON の解析に失敗しました: %w", err)
	}

	var lives []model.LiveInfo
	for _, item := range apiResp.Items {
		live := model.LiveInfo{
			ID:              item.ID,
			Title:           item.Snippet.Title,
			URL:             "https://www.youtube.com/watch?v=" + item.ID,
			Status:          mapLifeCycleToStatus(item.Status.LifeCycleStatus),
			LifeCycleStatus: item.Status.LifeCycleStatus,
			PrivacyStatus:   item.Status.PrivacyStatus,
		}
		if item.Snippet.ScheduledStartTime != "" {
			if t, err := time.Parse(time.RFC3339, item.Snippet.ScheduledStartTime); err == nil {
				live.ScheduledStartTime = t
			}
		}
		lives = append(lives, live)
	}
	return lives, nil
}

func mapLifeCycleToStatus(lifeCycle string) string {
	switch lifeCycle {
	case "live", "liveStarting":
		return model.StatusLive
	case "complete", "revoked":
		return model.StatusCompleted
	default:
		return model.StatusUpcoming
	}
}
