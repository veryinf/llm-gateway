package model

type UserModel struct {
	UserModelID uint   `json:"userModelId" gorm:"primaryKey;autoIncrement"`
	Name        string `json:"name" gorm:"unique"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	IsActive    bool   `json:"isActive"`
}

type UserModelRouter struct {
	RouterID        uint `json:"routerId" gorm:"primaryKey;autoIncrement"`
	UserModelID     uint `json:"userModelId"`
	ProviderModelID uint `json:"providerModelId"`
	Priority        uint `json:"priority"`
}
