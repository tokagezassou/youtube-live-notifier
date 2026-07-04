package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/tokagezassou/youtube-live-notifier/config"
	"github.com/tokagezassou/youtube-live-notifier/discord"
	"github.com/tokagezassou/youtube-live-notifier/handler"
	"github.com/tokagezassou/youtube-live-notifier/repository"
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

	youtubeClient := youtube.NewClient(cfg.YouTubeChannelID, cfg.YouTubeAPIKey)
	discordClient := discord.NewWebhookClient(cfg.DiscordWebhookURL)
	// memoryDB := repository.NewMemoryDB()
	ctx := context.Background()
	credPath := os.Getenv("FIRESTORE_CREDENTIALS_PATH")
	projectID := os.Getenv("GCP_PROJECT_ID")

	firestoreDB, err := repository.NewFirestoreDB(ctx, projectID, credPath)
	if err != nil {
		log.Fatalf("Firestore初期化エラー: %v", err)
	}
	defer firestoreDB.Close()

	notifierUsecase := usecase.NewNotifierUsecase(
		youtubeClient,
		firestoreDB,
		discordClient,
		cfg.DiscordRoleID,
	)
	youtubeHandler := handler.NewYouTubeHandler(notifierUsecase)

	http.HandleFunc("/check", youtubeHandler.Check)

	port := "8080"
	fmt.Printf("ローカルサーバーを起動しました: http://localhost:%s\n", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("サーバーの起動に失敗しました: %v", err)
	}
}
