package models

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID        int64          `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"uniqueIndex;size:64" json:"username"`
	IsScammer bool           `json:"is_scammer"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Ad struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	UserID            int64          `json:"user_id"`
	ClientID          string         `gorm:"size:64;index" json:"client_id"`
	Username          string         `gorm:"size:64;index" json:"username"`
	Title             string         `gorm:"size:128" json:"title"`
	Desc              string         `gorm:"size:2048" json:"desc"`
	PhotoID           string         `gorm:"size:256" json:"-"`
	PhotoPath         string         `gorm:"size:512" json:"-"`
	Category          string         `gorm:"size:32;index" json:"category"`
	Mode              string         `gorm:"size:16;index" json:"mode"`
	Tag               string         `gorm:"size:64;index" json:"tag"`
	IsPremium         bool           `json:"is_premium"`
	Status            string         `gorm:"size:16;index" json:"status"`
	ExpiresAt         time.Time      `gorm:"index" json:"expires_at"`
	PreExpiryNotified bool           `json:"-"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

const (
	AdStatusActive   = "active"
	AdStatusExpired  = "expired"
	AdStatusInactive = "inactive"
)
