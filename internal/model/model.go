package model

import "time"

type Model struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ProviderID uint      `gorm:"index;not null" json:"provider_id"`
	Name       string    `gorm:"size:128;not null" json:"name"`
	IsActive   bool      `gorm:"default:true" json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	Provider *Provider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}
