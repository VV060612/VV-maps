package model

// User 用户结构体 (用于登录认证)
import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username string `json:"username" gorm:"uniqueIndex;not null"` // 用户名唯一且不为空
	Password string `json:"password" gorm:"not null"`             // 加密后的密码
	Email    string `json:"email"`
}
