package model

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleUser   Role = "user"
	RoleViewer Role = "viewer"
)

type User struct {
	UID        uint   `json:"uid,omitempty" gorm:"primaryKey;autoIncrement"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Department string `json:"department"`
	Role       Role   `json:"role"`
	Status     string `json:"status"`
	AccessKey  string `json:"accessKey" gorm:"unique"`
	SecretKey  string `json:"secretKey"`
}

type UserKey struct {
	KeyID    uint   `json:"keyId" gorm:"primaryKey;autoIncrement"`
	UID      uint   `json:"uid"`
	Key      string `json:"key" gorm:"unique"`
	Title    string `json:"title"`
	IsActive bool   `json:"isActive"`
}
