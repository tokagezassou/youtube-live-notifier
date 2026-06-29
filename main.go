package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/tokagezassou/youtube-live-notifier/config"
	"github.com/tokagezassou/youtube-live-notifier/handler"
	"github.com/tokagezassou/youtube-live-notifier/usecase"
	"github.com/tokagezassou/youtube-live-notifier/youtube"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("[Warn] .env ファイルが見つかりません。OSの環境変数を使用します。")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("設定の読み込みに失敗しました: %v", err)
	}

	youtubeClient := youtube.NewRSSClient(cfg.ChannelID)
	notifierUsecase := usecase.NewNotifierUsecase(youtubeClient)
	youtubeHandler := handler.NewYouTubeHandler(notifierUsecase)

	http.HandleFunc("/check", youtubeHandler.Check)

	port := "8080"
	fmt.Printf("ローカルサーバーを起動しました: http://localhost:%s\n", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("サーバーの起動に失敗しました: %v", err)
	}
}
