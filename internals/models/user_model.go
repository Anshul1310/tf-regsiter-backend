package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	Name       string    `json:"name" db:"name"`
	Email      string    `json:"email" db:"email"`
	RollNumber *string   `json:"roll_number,omitempty" db:"roll_number"`
	Hostel     *string   `json:"hostel,omitempty" db:"hostel"`
	Mess       *string   `json:"mess,omitempty" db:"mess"`
	Gender     *string   `json:"gender,omitempty" db:"gender"`
	Pfp        *string   `json:"pfp,omitempty" db:"pfp"`
	TeamID     *string   `json:"team_id,omitempty" db:"team_id"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type SyncUserRequest struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Name   string    `json:"name"`
	Pfp    string    `json:"pfp"`
}

type EmailLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateUserProfileRequest struct {
	Name       string  `json:"name"`
	RollNumber string  `json:"roll_number"`
	Hostel     string  `json:"hostel"`
	Mess       string  `json:"mess"`
	Gender     string  `json:"gender"`
	Email      *string `json:"email,omitempty"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type DAuthLoginRequest struct {
	Code        string `json:"code"`
	RedirectURI string `json:"redirect_uri,omitempty"`
}

type DAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Error        string `json:"error,omitempty"`
	ErrorDesc    string `json:"error_description,omitempty"`
}

type DAuthUserResponse struct {
	ID          interface{} `json:"id"`
	Name        string      `json:"name"`
	Email       string      `json:"email"`
	Gender      string      `json:"gender,omitempty"`
	PhoneNumber string      `json:"phoneNumber,omitempty"`
	Batch       interface{} `json:"batch,omitempty"`
	Dept        string      `json:"dept,omitempty"`
	RollNo      string      `json:"rollNo,omitempty"`
}

