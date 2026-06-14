package model

type Config struct {
	Key         string `json:"key" gorm:"primaryKey;size:128"`
	Value       string `json:"value" gorm:"type:text"`
	Description string `json:"description"`
}
