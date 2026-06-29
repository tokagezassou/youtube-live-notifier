package youtube

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"github.com/tokagezassou/youtube-live-notifier/model"
)

type YouTubeClient struct {
	channelID string
}

func NewRSSClient(channelID string) *YouTubeClient {
	return &YouTubeClient{
		channelID: channelID,
	}
}

const rssBaseURL = "https://www.youtube.com/feeds/videos.xml?channel_id="

func (c *YouTubeClient) FetchLatestLives() ([]model.LiveInfo, error) {
	rssURL := rssBaseURL + c.channelID

	resp, err := http.Get(rssURL)
	if err != nil {
		return nil, fmt.Errorf("RSSの取得に失敗しました: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTPエラー: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("レスポンスの読み込みに失敗しました: %w", err)
	}

	var f feed
	if err := xml.Unmarshal(body, &f); err != nil {
		return nil, fmt.Errorf("XMLの解析に失敗しました: %w", err)
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

type feed struct {
	XMLName xml.Name `xml:"feed"`
	Entries []entry  `xml:"entry"`
}
type entry struct {
	VideoID string `xml:"videoId"`
	Title   string `xml:"title"`
}
