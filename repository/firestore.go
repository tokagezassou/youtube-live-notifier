package repository

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type StreamDocument struct {
	ID                 string
	Title              string
	URL                string
	ScheduledStartTime time.Time
	ShouldNotify       bool
	CreatedAt          time.Time
}

type FirestoreDB struct {
	client *firestore.Client
}

func NewFirestoreDB(ctx context.Context, projectID string, credPath string) (*FirestoreDB, error) {
	var client *firestore.Client
	var err error
	databaseID := "youtube-live-notifier-db"

	if credPath != "" {
		client, err = firestore.NewClientWithDatabase(ctx, projectID, databaseID,
			option.WithAuthCredentialsFile(option.ServiceAccount, credPath))
	} else {
		client, err = firestore.NewClientWithDatabase(ctx, projectID, databaseID)
	}

	if err != nil {
		return nil, err
	}
	return &FirestoreDB{client: client}, nil
}

func (db *FirestoreDB) Close() error {
	return db.client.Close()
}

func (db *FirestoreDB) Save(doc StreamDocument) {
	ctx := context.Background()
	_, err := db.client.Collection("streams").Doc(doc.ID).Set(ctx, doc)
	if err != nil {
		log.Printf("Firestore保存エラー (ID: %s): %v\n", doc.ID, err)
	}
}

func (db *FirestoreDB) GetLatest15IDs() []string {
	ctx := context.Background()
	var ids []string

	iter := db.client.Collection("streams").
		OrderBy("CreatedAt", firestore.Desc).
		Limit(15).
		Documents(ctx)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Firestore最新ID取得エラー: %v\n", err)
			return ids
		}
		ids = append(ids, doc.Ref.ID)
	}
	return ids
}

func (db *FirestoreDB) GetShouldNotifyStreams() []StreamDocument {
	ctx := context.Background()
	var targets []StreamDocument

	iter := db.client.Collection("streams").
		Where("ShouldNotify", "==", true).
		Documents(ctx)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Firestore通知対象取得エラー: %v\n", err)
			return targets
		}

		var s StreamDocument
		if err := doc.DataTo(&s); err == nil {
			targets = append(targets, s)
		} else {
			log.Printf("データ変換エラー (ID: %s): %v\n", doc.Ref.ID, err)
		}
	}
	return targets
}
