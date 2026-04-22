package users // Defines user-domain types shared across repository, service, and handler files.

import "time" // Uses time.Time for created/updated/login/password-change timestamps.

type User struct { // Represents one user row returned to API clients as JSON.
	ID                 int64      `json:"id"`                   // Primary key ID from database BIGSERIAL column.
	Name               string     `json:"name"`                 // User's display name used by API and DB.
	Email              string     `json:"email"`                // User's unique email used for login identity.
	Phone              string     `json:"phone"`                // User's phone used in contact and 2FA logs.
	MustChangePassword bool       `json:"must_change_password"` // Security flag telling client password must be rotated.
	LastLoginAt        *time.Time `json:"last_login_at"`        // Pointer allows null when user has never logged in.
	PasswordChangedAt  *time.Time `json:"password_changed_at"`  // Pointer allows null until password is changed once.
	TwoFactorEnabled   bool       `json:"two_factor_enabled"`   // Boolean indicates whether 2FA is enabled for this user.
	CreatedAt          time.Time  `json:"created_at"`           // Timestamp set by DB at insert time.
	UpdatedAt          time.Time  `json:"updated_at"`           // Timestamp updated on every write operation.
	
}

type CreateUserInput struct { // Input payload expected by handler on create endpoint.
	Name     string `json:"name"`     // Name sent from client JSON body.
	Email    string `json:"email"`    // Email sent from client JSON body.
	Phone    string `json:"phone"`    // Phone sent from client JSON body.
	Password string `json:"password"` // Raw password from client before hashing in service.
}

type UpdateUserInput struct { // Input payload expected by handler on update endpoint.
	Name  string `json:"name"`  // Updated name value from request body.
	Email string `json:"email"` // Updated email value from request body.
	Phone string `json:"phone"` // Updated phone value from request body.
}
