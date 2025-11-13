package handlers

import "sync"

var (
	botTokenMu sync.RWMutex
	botToken   string
)

func setBotToken(token string) {
	botTokenMu.Lock()
	defer botTokenMu.Unlock()
	botToken = token
}

func getBotToken() string {
	botTokenMu.RLock()
	defer botTokenMu.RUnlock()
	return botToken
}

