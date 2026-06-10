package model

import "time"

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleUser   Role = "user"
	RoleViewer Role = "viewer"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string    `gorm:"size:256;not null" json:"-"`
	Name         string    `gorm:"size:64" json:"name"`
	Phone        string    `gorm:"size:32" json:"phone"`
	Department   string    `gorm:"size:128" json:"department"`
	Role         Role      `gorm:"size:16;default:user" json:"role"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	AccessKey    string    `gorm:"size:64;uniqueIndex" json:"access_key"`
	SecretKey    string    `gorm:"size:128" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
