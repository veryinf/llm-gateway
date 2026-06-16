package model

import "time"

type DownstreamModel struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Name            string    `gorm:"uniqueIndex;size:128;not null" json:"name"`
	DisplayName     string    `gorm:"size:128;default:''" json:"display_name"`
	UpstreamModelID uint      `gorm:"index;not null" json:"upstream_model_id"`
	Description     string    `gorm:"size:512;default:''" json:"description"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	UpstreamModel *ProviderModel `gorm:"foreignKey:UpstreamModelID" json:"upstream_model,omitempty"`
}
