package users

import "time"

type User struct {
	ID                 int64      `json:"id"`
	Name               string     `json:"name"`
	Email              string     `json:"email"`
	Phone              string     `json:"phone"`
	MustChangePassword bool       `json:"must_change_password"`
	LastLoginAt        *time.Time `json:"last_login_at"`
	PasswordChangedAt  *time.Time `json:"password_changed_at"`
	TwoFactorEnabled   bool       `json:"two_factor_enabled"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type CreateUserInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type UpdateUserInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}
