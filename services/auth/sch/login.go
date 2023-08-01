package sch

import "strings"

type LoginForm struct {
	Username string `json:"username" binding:"required"` // Login name
	Password string `json:"password" binding:"required"` // Login password (md5 hash)
}

func (a *LoginForm) Trim() *LoginForm {
	a.Username = strings.TrimSpace(a.Username)
	return a
}

type UpdateLoginPassword struct {
	OldPassword string `json:"old_password" binding:"required"` // Old password (md5 hash)
	NewPassword string `json:"new_password" binding:"required"` // New password (md5 hash)
}

type LoginToken struct {
	AccessToken string `json:"access_token"` // Access token (JWT)
	TokenType   string `json:"token_type"`   // Token type (Usage: Authorization=${token_type} ${access_token})
	ExpiresAt   int64  `json:"expires_at"`   // Expired time (Unit: second)
}
