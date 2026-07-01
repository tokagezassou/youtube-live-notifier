package repository

import (
	"sort"
	"time"
)

type StreamDocument struct {
	ID                 string
	Title              string
	URL                string
	ScheduledStartTime time.Time
	ShouldNotify       bool
	CreatedAt          time.Time
}

type MemoryDB struct {
	data map[string]StreamDocument
}

func NewMemoryDB() *MemoryDB {
	return &MemoryDB{
		data: make(map[string]StreamDocument),
	}
}

func (db *MemoryDB) Save(doc StreamDocument) {
	db.data[doc.ID] = doc
}

func (db *MemoryDB) GetLatest15IDs() []string {
	var docs []StreamDocument
	for _, doc := range db.data {
		docs = append(docs, doc)
	}

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].CreatedAt.After(docs[j].CreatedAt)
	})

	var ids []string
	limit := 15
	if len(docs) < limit {
		limit = len(docs)
	}
	for i := 0; i < limit; i++ {
		ids = append(ids, docs[i].ID)
	}
	return ids
}

func (db *MemoryDB) GetShouldNotifyStreams() []StreamDocument {
	var targets []StreamDocument
	for _, doc := range db.data {
		if doc.ShouldNotify {
			targets = append(targets, doc)
		}
	}
	return targets
}
