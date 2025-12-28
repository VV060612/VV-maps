package model

// User 用户结构体 (用于登录认证)
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"` // 存储加密后的密码
	Email    string `json:"email,omitempty"`
}
