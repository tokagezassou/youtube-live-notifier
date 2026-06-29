package handler

import (
	"fmt"
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
	fmt.Println("[Info] リクエストを受け付けました")

	resultMessage, err := h.usecase.CheckAndNotify()
	if err != nil {
		fmt.Printf("[Error] %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, resultMessage)
}
