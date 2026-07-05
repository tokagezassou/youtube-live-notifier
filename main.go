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

	http.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		log.Println("定期チェックのリクエストを受信しました")

		msg, err := notifierUsecase.CheckAndNotify()
		if err != nil {
			log.Printf("処理エラー: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		log.Printf("処理完了: %s", msg)
		fmt.Fprintf(w, "Success: %s", msg)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("サーバーを起動します。ポート: %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("サーバー起動エラー: %v", err)
	}
}
