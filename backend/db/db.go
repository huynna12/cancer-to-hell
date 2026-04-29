package db

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

type Session struct {
	ID         string    `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	CancerType string    `json:"cancer_type"`
	Stage      string    `json:"stage"`
	Evidence   string    `json:"evidence"`
	Guideline  string    `json:"guideline"`
	Safety     string    `json:"safety"`
}

var (
	mu       sync.Mutex
	filePath string
)

func Init(path string) {
	filePath = path
}

func Save(s Session) {
	mu.Lock()
	defer mu.Unlock()

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	json.NewEncoder(f).Encode(s) //nolint:errcheck
}

func List() []Session {
	mu.Lock()
	defer mu.Unlock()

	f, err := os.Open(filePath)
	if err != nil {
		return []Session{}
	}
	defer f.Close()

	var sessions []Session
	dec := json.NewDecoder(f)
	for dec.More() {
		var s Session
		if err := dec.Decode(&s); err != nil {
			continue
		}
		sessions = append(sessions, s)
	}
	return sessions
}
