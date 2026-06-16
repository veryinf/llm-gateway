package model

type APIKey struct {
	KeyID    uint   `json:"keyId" gorm:"primaryKey;autoIncrement"`
	UID      uint   `json:"uid"`
	Key      string `json:"key" gorm:"unique"`
	Title    string `json:"title"`
	IsActive bool   `json:"isActive"`
}
