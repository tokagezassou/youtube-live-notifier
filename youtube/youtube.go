package youtube

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/tokagezassou/youtube-live-notifier/model"
)

type Client struct {
	channelID string
	apiKey    string
}

func NewClient(channelID, apiKey string) *Client {
	return &Client{
		channelID: channelID,
		apiKey:    apiKey,
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
	resp, err := http.Get(rssURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var f feed
	if err := xml.Unmarshal(body, &f); err != nil {
		return nil, err
	}

	var lives []model.LiveInfo
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
	} `json:"items"`
}

func (c *Client) FilterLiveVideos(videoIDs []string) ([]string, error) {
	if len(videoIDs) == 0 {
		return []string{}, nil
	}

	idsParam := strings.Join(videoIDs, ",")

	apiURL := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/videos?part=snippet&id=%s&key=%s",
		url.QueryEscape(idsParam),
		c.apiKey,
	)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("YouTube APIの送信に失敗しました: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("YouTube APIエラー: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp videoAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("JSONの解析に失敗しました: %w", err)
	}

	var liveIDs []string
	for _, item := range apiResp.Items {
		if item.Snippet.LiveBroadcastContent == "live" || item.Snippet.LiveBroadcastContent == "upcoming" {
			liveIDs = append(liveIDs, item.ID)
		}
	}

	return liveIDs, nil
}
