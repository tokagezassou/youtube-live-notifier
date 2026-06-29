package repository

type MemoryDB struct {
	notifiedIDs map[string]bool
}

func NewMemoryDB() *MemoryDB {
	return &MemoryDB{
		notifiedIDs: make(map[string]bool),
	}
}

func (db *MemoryDB) IsNotified(id string) bool {
	return db.notifiedIDs[id]
}

func (db *MemoryDB) MarkAsNotified(id string) {
	db.notifiedIDs[id] = true
}
