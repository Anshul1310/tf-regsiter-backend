package models

import (
	"time"

	"github.com/google/uuid"
)

type AdminRole string

const (
	RoleSuperAdmin AdminRole = "superadmin"
	RoleAdmin      AdminRole = "admin"
)

type AdminUser struct {
	AdminID           uuid.UUID `json:"admin_id" db:"admin_id"`
	Email             string    `json:"email" db:"email"`
	Name              string    `json:"name" db:"name"`
	Picture           *string   `json:"picture,omitempty" db:"picture"`
	Role              string    `json:"role" db:"role"` // 'superadmin' or 'admin'
	CanManageUsers    bool      `json:"can_manage_users" db:"can_manage_users"`
	CanManageTeams    bool      `json:"can_manage_teams" db:"can_manage_teams"`
	CanManagePayments bool      `json:"can_manage_payments" db:"can_manage_payments"`
	CanManageAdmins   bool      `json:"can_manage_admins" db:"can_manage_admins"`
	IsActive          bool      `json:"is_active" db:"is_active"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

type GoogleAuthRequest struct {
	Credential  string `json:"credential,omitempty"` // Google ID Token (from Google One Tap / GSI)
	AccessToken string `json:"access_token,omitempty"` // Google Access Token (from oauth2)
	Code        string `json:"code,omitempty"`
}

type GoogleTokenInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
}

type CreateAdminRequest struct {
	Email             string `json:"email"`
	Name              string `json:"name"`
	Role              string `json:"role"` // "superadmin" or "admin"
	CanManageUsers    bool   `json:"can_manage_users"`
	CanManageTeams    bool   `json:"can_manage_teams"`
	CanManagePayments bool   `json:"can_manage_payments"`
	CanManageAdmins   bool   `json:"can_manage_admins"`
}

type UpdateAdminRequest struct {
	Name              *string `json:"name,omitempty"`
	Role              *string `json:"role,omitempty"`
	CanManageUsers    *bool   `json:"can_manage_users,omitempty"`
	CanManageTeams    *bool   `json:"can_manage_teams,omitempty"`
	CanManagePayments *bool   `json:"can_manage_payments,omitempty"`
	CanManageAdmins   *bool   `json:"can_manage_admins,omitempty"`
	IsActive          *bool   `json:"is_active,omitempty"`
}

type AdminDashboardStats struct {
	TotalUsers          int                    `json:"total_users"`
	TotalTeams          int                    `json:"total_teams"`
	PaidTeams           int                    `json:"paid_teams"`
	PendingTeams        int                    `json:"pending_teams"`
	PublicTeams         int                    `json:"public_teams"`
	TotalRevenue        float64                `json:"total_revenue"`
	UsersWithTeam       int                    `json:"users_with_team"`
	UsersWithoutTeam    int                    `json:"users_without_team"`
	DomainStats         map[string]int         `json:"domain_stats"`
	PaymentStatusStats  map[string]int         `json:"payment_status_stats"`
	RecentRegistrations []User                 `json:"recent_registrations"`
	RecentPayments      []Payment              `json:"recent_payments"`
}

type Payment struct {
	PaymentID        string    `json:"payment_id" db:"payment_id"`
	OrderID          string    `json:"order_id" db:"order_id"`
	TeamID           string    `json:"team_id" db:"team_id"`
	TeamName         *string   `json:"team_name,omitempty" db:"team_name"`
	UserID           *uuid.UUID `json:"user_id,omitempty" db:"user_id"`
	UserName         *string   `json:"user_name,omitempty" db:"user_name"`
	UserEmail        *string   `json:"user_email,omitempty" db:"user_email"`
	Amount           float64   `json:"amount" db:"amount"`
	Currency         string    `json:"currency" db:"currency"`
	PaymentStatus    string    `json:"payment_status" db:"payment_status"`
	PaymentSessionID *string   `json:"payment_session_id,omitempty" db:"payment_session_id"`
	TransactionID    *string   `json:"transaction_id,omitempty" db:"transaction_id"`
	ScreenshotURL    *string   `json:"screenshot_url,omitempty" db:"screenshot_url"`
	RawWebhookData   *string   `json:"raw_webhook_data,omitempty" db:"raw_webhook_data"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type ManualPaymentVerifyRequest struct {
	TeamID        string  `json:"team_id"`
	Amount        float64 `json:"amount"`
	PaymentStatus string  `json:"payment_status"` // "SUCCESS", "PAID", "REJECTED"
	TransactionID string  `json:"transaction_id"`
	Notes         string  `json:"notes,omitempty"`
}

type UpdateUserAdminRequest struct {
	Name       string  `json:"name"`
	Email      string  `json:"email"`
	RollNumber *string `json:"roll_number,omitempty"`
	Hostel     *string `json:"hostel,omitempty"`
	Mess       *string `json:"mess,omitempty"`
	Gender     *string `json:"gender,omitempty"`
	TeamID     *string `json:"team_id,omitempty"`
}

type UpdateTeamAdminRequest struct {
	Name             string  `json:"name"`
	Domain           *string `json:"domain,omitempty"`
	ProblemStatement *string `json:"problem_statement,omitempty"`
	Contact          string  `json:"contact"`
	IsPublic         bool    `json:"ispublic"`
	PaymentStatus    string  `json:"payment_status"`
	LeaderUserID     *uuid.UUID `json:"leader_user_id,omitempty"`
	LeaderName       *string `json:"leader,omitempty"`
}

type AddTeamMemberRequest struct {
	UserID *uuid.UUID `json:"user_id,omitempty"`
	Email  *string    `json:"email,omitempty"`
}

type ChangeTeamLeaderRequest struct {
	NewLeaderUserID uuid.UUID `json:"new_leader_user_id"`
}
