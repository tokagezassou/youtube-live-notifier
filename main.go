package main

import (
	"context"
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

	h := handler.NewYouTubeHandler(notifierUsecase)

	http.HandleFunc("/check", h.Check)
	http.HandleFunc("/check_by_search", h.CheckBySearch)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("サーバーを起動します。ポート: %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("サーバー起動エラー: %v", err)
	}
}
