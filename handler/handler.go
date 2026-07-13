package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/tokagezassou/youtube-live-notifier/usecase"
)

type YouTubeHandler struct {
	usecase *usecase.NotifierUsecase
}

func NewYouTubeHandler(u *usecase.NotifierUsecase) *YouTubeHandler {
	return &YouTubeHandler{
		usecase: u,
	}
}

func (h *YouTubeHandler) Check(w http.ResponseWriter, r *http.Request) {
	log.Println("[rss] 定期チェックのリクエストを受信しました")

	msg, err := h.usecase.CheckAndNotify()
	if err != nil {
		log.Printf("[Error][rss] %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("[rss] 処理完了: %s", msg)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, msg)
}

func (h *YouTubeHandler) CheckBySearch(w http.ResponseWriter, r *http.Request) {
	log.Println("[search] 手動フォールバックのリクエストを受信しました")

	msg, err := h.usecase.SearchAndNotify()
	if err != nil {
		log.Printf("[Error][search] %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("[search] 処理完了: %s", msg)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, msg)
}
