package youtube

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tokagezassou/youtube-live-notifier/model"
)

type Client struct {
	channelID string
	apiKey    string
	http      *http.Client
}

func NewClient(channelID, apiKey string) *Client {
	return &Client{
		channelID: strings.TrimSpace(channelID),
		apiKey:    strings.TrimSpace(apiKey),
		http:      &http.Client{Timeout: 10 * time.Second},
	}
}

type feed struct {
	XMLName xml.Name `xml:"feed"`
	Entries []entry  `xml:"entry"`
}
type entry struct {
	VideoID string `xml:"videoId"`
	Title   string `xml:"title"`
}

func (c *Client) FetchLatestLives() ([]model.LiveInfo, error) {
	rssURL := fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", c.channelID)

	resp, err := c.http.Get(rssURL)
	if err != nil {
		return nil, wrapHTTPErr("YouTube RSSの取得に失敗しました", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("YouTube RSSがエラーを返しました (ステータス: %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("RSSの読み込みに失敗しました: %w", err)
	}

	var f feed
	if err := xml.Unmarshal(body, &f); err != nil {
		return nil, fmt.Errorf("RSSの解析に失敗しました: %w", err)
	}

	lives := make([]model.LiveInfo, 0, len(f.Entries))
	for _, e := range f.Entries {
		lives = append(lives, model.LiveInfo{
			ID:    e.VideoID,
			Title: e.Title,
			URL:   "https://www.youtube.com/watch?v=" + e.VideoID,
		})
	}
	return lives, nil
}

type videoAPIResponse struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			LiveBroadcastContent string `json:"liveBroadcastContent"`
		} `json:"snippet"`
		LiveStreamingDetails struct {
			ScheduledStartTime string `json:"scheduledStartTime"`
		} `json:"liveStreamingDetails"`
	} `json:"items"`
}

func (c *Client) FetchStreamDetails(videoIDs []string) (map[string]model.LiveInfo, error) {
	if len(videoIDs) == 0 {
		return map[string]model.LiveInfo{}, nil
	}

	q := url.Values{}
	q.Set("part", "snippet,liveStreamingDetails")
	q.Set("id", strings.Join(videoIDs, ","))
	q.Set("key", c.apiKey)

	apiURL := "https://www.googleapis.com/youtube/v3/videos?" + q.Encode()

	resp, err := c.http.Get(apiURL)
	if err != nil {
		return nil, wrapHTTPErr("YouTube APIの送信に失敗しました", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("YouTube APIエラー: %s (%s)", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp videoAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("JSONの解析に失敗しました: %w", err)
	}

	result := make(map[string]model.LiveInfo)
	for _, item := range apiResp.Items {
		details := model.LiveInfo{
			ID:     item.ID,
			Status: item.Snippet.LiveBroadcastContent,
		}

		if item.LiveStreamingDetails.ScheduledStartTime != "" {
			t, err := time.Parse(time.RFC3339, item.LiveStreamingDetails.ScheduledStartTime)
			if err == nil {
				details.ScheduledStartTime = t
			}
		}

		result[item.ID] = details
	}

	return result, nil
}

type searchAPIResponse struct {
	Items []struct {
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
		Snippet struct {
			Title string `json:"title"`
		} `json:"snippet"`
	} `json:"items"`
}

func (c *Client) SearchUpcomingLives() ([]model.LiveInfo, error) {
	q := url.Values{}
	q.Set("part", "snippet")
	q.Set("channelId", c.channelID)
	q.Set("eventType", "upcoming")
	q.Set("type", "video")
	q.Set("maxResults", "10")
	q.Set("key", c.apiKey)

	apiURL := "https://www.googleapis.com/youtube/v3/search?" + q.Encode()

	resp, err := c.http.Get(apiURL)
	if err != nil {
		return nil, wrapHTTPErr("YouTube Search APIの送信に失敗しました", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("YouTube Search APIエラー: %s (%s)", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp searchAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("JSONの解析に失敗しました: %w", err)
	}

	lives := make([]model.LiveInfo, 0, len(apiResp.Items))
	for _, item := range apiResp.Items {
		if item.ID.VideoID == "" {
			continue
		}
		lives = append(lives, model.LiveInfo{
			ID:    item.ID.VideoID,
			Title: item.Snippet.Title,
			URL:   "https://www.youtube.com/watch?v=" + item.ID.VideoID,
		})
	}
	return lives, nil
}

func wrapHTTPErr(msg string, err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Errorf("%s: %v", msg, urlErr.Err)
	}
	return fmt.Errorf("%s: %w", msg, err)
}
