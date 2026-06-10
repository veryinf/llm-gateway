package model

import "time"

type APIKey struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"index;not null" json:"user_id"`
	KeyHash      string    `gorm:"uniqueIndex;size:64;not null" json:"-"`
	KeyPrefix    string    `gorm:"size:32;not null" json:"key_prefix"`
	Name         string    `gorm:"size:128" json:"name"`
	QuotaLimit   int64     `gorm:"default:0" json:"quota_limit"`
	QuotaUsed    int64     `gorm:"default:0" json:"quota_used"`
	RateLimitQPM int       `gorm:"default:60" json:"rate_limit_qpm"`
	ExpiresAt    *time.Time `json:"expires_at"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	LastUsedAt   *time.Time `json:"last_used_at"`
	CreatedAt    time.Time `json:"created_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
