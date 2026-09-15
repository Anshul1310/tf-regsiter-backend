package models

import (
	"time"

	"github.com/google/uuid"
)

type Team struct {
	TeamID           string    `json:"team_id" db:"team_id"`
	Name             string    `json:"name" db:"name"`
	Leader           string    `json:"leader" db:"leader"`
	LeaderUserID     uuid.UUID `json:"leader_user_id" db:"leader_user_id"`
	Contact          string    `json:"contact" db:"contact"`
	Domain           *string   `json:"domain,omitempty" db:"domain"`
	ProblemStatement *string   `json:"problem_statement,omitempty" db:"problem_statement"`
	PaymentStatus    string    `json:"payment_status" db:"payment_status"`
	IsPublic         bool      `json:"ispublic" db:"ispublic"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`

	Members []User `json:"members,omitempty"`
}

type CreateTeamRequest struct {
	Name             string    `json:"name"`
	Contact          string    `json:"contactNumber"`
	Domain           *string   `json:"domain,omitempty"`
	ProblemStatement *string   `json:"problem_statement,omitempty"`
	LeaderEmail      string    `json:"leader_email,omitempty"`
	LeaderUserID     uuid.UUID `json:"leader_user_id,omitempty"`
}

type JoinTeamRequest struct {
	TeamCode string `json:"teamCode"`
}

type UpdateVisibilityRequest struct {
	IsPublic bool `json:"ispublic"`
}

type UpdateTeamNameRequest struct {
	Name string `json:"name"`
}

type UpdateTeamDetailsRequest struct {
	Domain           *string `json:"domain,omitempty"`
	ProblemStatement *string `json:"problem_statement,omitempty"`
}

type SubmitManualPaymentRequest struct {
	TransactionID string `json:"transactionId"`
	ScreenshotURL string `json:"screenshotUrl,omitempty"`
}

type CashfreeCheckoutRequest struct {
	TeamID   string `json:"teamId"`
	UserID   string `json:"userId"`
	TeamName string `json:"teamName"`
}

type CashfreeOrderResponse struct {
	PaymentSessionID string  `json:"payment_session_id"`
	OrderID          string  `json:"order_id"`
	OrderStatus      string  `json:"order_status"`
	OrderAmount      float64 `json:"order_amount"`
	OrderCurrency    string  `json:"order_currency"`
}

type CashfreeVerifyRequest struct {
	OrderID string `json:"order_id"`
	TeamID  string `json:"team_id"`
}
