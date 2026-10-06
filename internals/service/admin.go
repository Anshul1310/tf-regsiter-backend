package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Anshul1310/tf-register/config"
	"github.com/Anshul1310/tf-register/internals/models"
	"github.com/Anshul1310/tf-register/internals/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrAdminUnauthorized = errors.New("access denied: you do not have permission to access the admin portal")
	ErrAdminNotFound     = errors.New("admin user not found")
	ErrAdminInactive     = errors.New("admin account is deactivated")
)

type AdminService struct {
	adminRepo *repository.AdminRepository
	cfg       *config.Config
}

func NewAdminService(adminRepo *repository.AdminRepository, cfg *config.Config) *AdminService {
	return &AdminService{
		adminRepo: adminRepo,
		cfg:       cfg,
	}
}

// -------------------------------------------------------------
// Google OAuth & Authentication
// -------------------------------------------------------------

func (s *AdminService) VerifyGoogleToken(ctx context.Context, token string) (*models.GoogleTokenInfo, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("empty google token")
	}

	// 1. Try Google ID token validation endpoint
	tokenInfoURL := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var info models.GoogleTokenInfo
		if err := json.NewDecoder(resp.Body).Decode(&info); err == nil && info.Email != "" {
			return &info, nil
		}
	}
	if resp != nil {
		_ = resp.Body.Close()
	}

	// 2. Try Google UserInfo endpoint (if access_token was passed instead of id_token)
	userInfoURL := "https://www.googleapis.com/oauth2/v3/userinfo"
	reqUser, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
	if err == nil {
		reqUser.Header.Set("Authorization", "Bearer "+token)
		respUser, errUser := client.Do(reqUser)
		if errUser == nil && respUser.StatusCode == http.StatusOK {
			defer respUser.Body.Close()
			var info models.GoogleTokenInfo
			if err := json.NewDecoder(respUser.Body).Decode(&info); err == nil && info.Email != "" {
				return &info, nil
			}
		}
		if respUser != nil {
			_ = respUser.Body.Close()
		}
	}

	return nil, errors.New("invalid or expired google token")
}

func (s *AdminService) HandleGoogleLogin(ctx context.Context, googleToken string) (string, *models.AdminUser, error) {
	googleInfo, err := s.VerifyGoogleToken(ctx, googleToken)
	if err != nil {
		return "", nil, fmt.Errorf("google authentication failed: %w", err)
	}

	email := strings.ToLower(strings.TrimSpace(googleInfo.Email))
	if email == "" {
		return "", nil, errors.New("google account has no email")
	}

	// Check if this email is configured as an initial superadmin
	isConfiguredSuperadmin := false
	for _, superEmail := range s.cfg.AdminEmails {
		if strings.ToLower(strings.TrimSpace(superEmail)) == email {
			isConfiguredSuperadmin = true
			break
		}
	}

	// Query database for admin record
	adminUser, err := s.adminRepo.GetAdminByEmail(ctx, email)
	if err != nil {
		return "", nil, fmt.Errorf("database lookup error: %w", err)
	}

	if adminUser == nil {
		if !isConfiguredSuperadmin {
			return "", nil, ErrAdminUnauthorized
		}

		// Auto-provision initial superadmin
		adminUser = &models.AdminUser{
			AdminID:           uuid.New(),
			Email:             email,
			Name:              googleInfo.Name,
			Picture:           &googleInfo.Picture,
			Role:              string(models.RoleSuperAdmin),
			CanManageUsers:    true,
			CanManageTeams:    true,
			CanManagePayments: true,
			CanManageAdmins:   true,
			IsActive:          true,
		}
		if err := s.adminRepo.CreateAdmin(ctx, adminUser); err != nil {
			return "", nil, fmt.Errorf("failed to provision superadmin: %w", err)
		}
	} else {
		if !adminUser.IsActive {
			return "", nil, ErrAdminInactive
		}

		// Update name & picture if changed
		if googleInfo.Picture != "" && (adminUser.Picture == nil || *adminUser.Picture != googleInfo.Picture) {
			adminUser.Picture = &googleInfo.Picture
		}
		if googleInfo.Name != "" && adminUser.Name == "" {
			adminUser.Name = googleInfo.Name
		}
		_ = s.adminRepo.CreateAdmin(ctx, adminUser)
	}

	// Generate Admin JWT
	tokenString, err := s.GenerateAdminJWT(adminUser)
	if err != nil {
		return "", nil, fmt.Errorf("token generation failed: %w", err)
	}

	return tokenString, adminUser, nil
}

// Demo/Dev login for local setup or testing
func (s *AdminService) HandleDevLogin(ctx context.Context, email string) (string, *models.AdminUser, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", nil, errors.New("email is required")
	}

	adminUser, err := s.adminRepo.GetAdminByEmail(ctx, email)
	if err != nil {
		return "", nil, err
	}

	if adminUser == nil {
		// In dev/demo mode, if no admins exist or email is superadmin, provision one
		adminList, _ := s.adminRepo.ListAdmins(ctx)
		if len(adminList) == 0 || email == "admin@transfinitte.com" || email == "superadmin@transfinitte.com" {
			adminUser = &models.AdminUser{
				AdminID:           uuid.New(),
				Email:             email,
				Name:              "Super Administrator",
				Role:              string(models.RoleSuperAdmin),
				CanManageUsers:    true,
				CanManageTeams:    true,
				CanManagePayments: true,
				CanManageAdmins:   true,
				IsActive:          true,
			}
			_ = s.adminRepo.CreateAdmin(ctx, adminUser)
		} else {
			return "", nil, ErrAdminUnauthorized
		}
	}

	if !adminUser.IsActive {
		return "", nil, ErrAdminInactive
	}

	tokenString, err := s.GenerateAdminJWT(adminUser)
	if err != nil {
		return "", nil, err
	}

	return tokenString, adminUser, nil
}

func (s *AdminService) GenerateAdminJWT(admin *models.AdminUser) (string, error) {
	claims := jwt.MapClaims{
		"sub":                 admin.AdminID.String(),
		"email":               admin.Email,
		"name":                admin.Name,
		"role":                admin.Role,
		"can_manage_users":    admin.CanManageUsers,
		"can_manage_teams":    admin.CanManageTeams,
		"can_manage_payments": admin.CanManagePayments,
		"can_manage_admins":   admin.CanManageAdmins,
		"type":                "admin",
		"aud":                 "tf-register-admin",
		"iat":                 time.Now().Unix(),
		"exp":                 time.Now().Add(7 * 24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := []byte(s.cfg.JWTSecret)
	return token.SignedString(secret)
}

func (s *AdminService) GetAdminByID(ctx context.Context, adminID uuid.UUID) (*models.AdminUser, error) {
	return s.adminRepo.GetAdminByID(ctx, adminID)
}

// -------------------------------------------------------------
// Admin Management (RBAC)
// -------------------------------------------------------------

func (s *AdminService) ListAdmins(ctx context.Context) ([]models.AdminUser, error) {
	return s.adminRepo.ListAdmins(ctx)
}

func (s *AdminService) CreateAdmin(ctx context.Context, req *models.CreateAdminRequest) (*models.AdminUser, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		return nil, errors.New("email is required")
	}

	role := req.Role
	if role != string(models.RoleSuperAdmin) && role != string(models.RoleAdmin) {
		role = string(models.RoleAdmin)
	}

	// Superadmin automatically gets all permissions
	canUsers := req.CanManageUsers
	canTeams := req.CanManageTeams
	canPayments := req.CanManagePayments
	canAdmins := req.CanManageAdmins
	if role == string(models.RoleSuperAdmin) {
		canUsers = true
		canTeams = true
		canPayments = true
		canAdmins = true
	}

	admin := &models.AdminUser{
		AdminID:           uuid.New(),
		Email:             email,
		Name:              req.Name,
		Role:              role,
		CanManageUsers:    canUsers,
		CanManageTeams:    canTeams,
		CanManagePayments: canPayments,
		CanManageAdmins:   canAdmins,
		IsActive:          true,
	}

	if err := s.adminRepo.CreateAdmin(ctx, admin); err != nil {
		return nil, err
	}

	return admin, nil
}

// SeedRootAdmins ensures root superadmin accounts exist in database with full access
func (s *AdminService) SeedRootAdmins(ctx context.Context) error {
	rootAccounts := []struct {
		Email string
		Name  string
	}{
		{Email: "transfinitte@gmail.com", Name: "Transfinitte Root Admin"},
		{Email: "negi.anshulnegi17@gmail.com", Name: "Anshul Negi (Root)"},
	}

	for _, root := range rootAccounts {
		existing, err := s.adminRepo.GetAdminByEmail(ctx, root.Email)
		if err != nil {
			fmt.Printf("Notice: DB check for root admin %s: %v\n", root.Email, err)
			continue
		}

		if existing == nil {
			newRoot := &models.AdminUser{
				AdminID:           uuid.New(),
				Email:             root.Email,
				Name:              root.Name,
				Role:              string(models.RoleSuperAdmin),
				CanManageUsers:    true,
				CanManageTeams:    true,
				CanManagePayments: true,
				CanManageAdmins:   true,
				IsActive:          true,
			}
			if err := s.adminRepo.CreateAdmin(ctx, newRoot); err != nil {
				fmt.Printf("Warning: Failed to seed root admin %s: %v\n", root.Email, err)
			} else {
				fmt.Printf("Successfully seeded root superadmin: %s\n", root.Email)
			}
		} else {
			// Ensure root superadmin permissions and active status
			if existing.Role != string(models.RoleSuperAdmin) || !existing.IsActive || !existing.CanManageAdmins {
				existing.Role = string(models.RoleSuperAdmin)
				existing.CanManageUsers = true
				existing.CanManageTeams = true
				existing.CanManagePayments = true
				existing.CanManageAdmins = true
				existing.IsActive = true
				_ = s.adminRepo.CreateAdmin(ctx, existing)
				fmt.Printf("Re-verified and ensured root superadmin status: %s\n", root.Email)
			}
		}
	}

	// Also provision any additional configured superadmins in AdminEmails
	for _, adminEmail := range s.cfg.AdminEmails {
		existing, err := s.adminRepo.GetAdminByEmail(ctx, adminEmail)
		if err == nil && existing == nil {
			newAdmin := &models.AdminUser{
				AdminID:           uuid.New(),
				Email:             adminEmail,
				Name:              "Administrator",
				Role:              string(models.RoleSuperAdmin),
				CanManageUsers:    true,
				CanManageTeams:    true,
				CanManagePayments: true,
				CanManageAdmins:   true,
				IsActive:          true,
			}
			_ = s.adminRepo.CreateAdmin(ctx, newAdmin)
			fmt.Printf("Provisioned configured admin: %s\n", adminEmail)
		}
	}

	return nil
}

func (s *AdminService) UpdateAdmin(ctx context.Context, adminID uuid.UUID, req *models.UpdateAdminRequest) error {
	return s.adminRepo.UpdateAdmin(ctx, adminID, req)
}

func (s *AdminService) DeleteAdmin(ctx context.Context, adminID uuid.UUID) error {
	adm, err := s.adminRepo.GetAdminByID(ctx, adminID)
	if err != nil {
		return err
	}
	if adm != nil {
		cleanEmail := strings.ToLower(strings.TrimSpace(adm.Email))
		if cleanEmail == "transfinitte@gmail.com" || cleanEmail == "negi.anshulnegi17@gmail.com" {
			return errors.New("cannot delete root superadmin account")
		}
	}
	return s.adminRepo.DeleteAdmin(ctx, adminID)
}

// -------------------------------------------------------------
// Dashboard & Analytics
// -------------------------------------------------------------

func (s *AdminService) GetDashboardStats(ctx context.Context) (*models.AdminDashboardStats, error) {
	return s.adminRepo.GetDashboardStats(ctx)
}

// -------------------------------------------------------------
// User Management
// -------------------------------------------------------------

func (s *AdminService) ListUsers(ctx context.Context, search, hostel, mess, gender, hasTeam string, page, limit int) ([]models.User, int, error) {
	return s.adminRepo.ListUsers(ctx, search, hostel, mess, gender, hasTeam, page, limit)
}

func (s *AdminService) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, *models.Team, error) {
	return s.adminRepo.GetUserWithTeam(ctx, userID)
}

func (s *AdminService) UpdateUser(ctx context.Context, userID uuid.UUID, req *models.UpdateUserAdminRequest) error {
	return s.adminRepo.UpdateUser(ctx, userID, req)
}

func (s *AdminService) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	return s.adminRepo.DeleteUser(ctx, userID)
}

// -------------------------------------------------------------
// Team Management
// -------------------------------------------------------------

func (s *AdminService) ListTeams(ctx context.Context, search, domain, paymentStatus, isPublic string, page, limit int) ([]models.Team, int, error) {
	return s.adminRepo.ListTeams(ctx, search, domain, paymentStatus, isPublic, page, limit)
}

func (s *AdminService) GetTeamByID(ctx context.Context, teamID string) (*models.Team, error) {
	return s.adminRepo.GetTeamDetails(ctx, teamID)
}

func (s *AdminService) CreateTeam(ctx context.Context, team *models.Team) error {
	if team.TeamID == "" {
		team.TeamID = "TEAM-" + strings.ToUpper(uuid.New().String()[:8])
	}
	if team.PaymentStatus == "" {
		team.PaymentStatus = "Pending"
	}
	return s.adminRepo.CreateTeam(ctx, team)
}

func (s *AdminService) UpdateTeam(ctx context.Context, teamID string, req *models.UpdateTeamAdminRequest) error {
	return s.adminRepo.UpdateTeam(ctx, teamID, req)
}

func (s *AdminService) UpdateTeamPaymentStatus(ctx context.Context, teamID string, status string) error {
	return s.adminRepo.UpdateTeamPaymentStatus(ctx, teamID, status)
}

func (s *AdminService) DeleteTeam(ctx context.Context, teamID string) error {
	return s.adminRepo.DeleteTeam(ctx, teamID)
}

func (s *AdminService) AddTeamMember(ctx context.Context, teamID string, userID uuid.UUID) error {
	return s.adminRepo.AddTeamMember(ctx, teamID, userID)
}

func (s *AdminService) RemoveTeamMember(ctx context.Context, teamID string, userID uuid.UUID) error {
	return s.adminRepo.RemoveTeamMember(ctx, teamID, userID)
}

func (s *AdminService) ChangeTeamLeader(ctx context.Context, teamID string, newLeaderUserID uuid.UUID) error {
	return s.adminRepo.ChangeTeamLeader(ctx, teamID, newLeaderUserID)
}

// -------------------------------------------------------------
// Payment Management
// -------------------------------------------------------------

func (s *AdminService) ListPayments(ctx context.Context, search, status string, page, limit int) ([]models.Payment, int, error) {
	return s.adminRepo.ListPayments(ctx, search, status, page, limit)
}

func (s *AdminService) VerifyManualPayment(ctx context.Context, req *models.ManualPaymentVerifyRequest) error {
	if req.Amount <= 0 {
		req.Amount = s.cfg.PaymentAmount
	}
	return s.adminRepo.RecordManualPayment(ctx, req)
}

// -------------------------------------------------------------
// CSV Exports
// -------------------------------------------------------------

func (s *AdminService) ExportUsersCSV(ctx context.Context) ([]byte, error) {
	users, err := s.adminRepo.ExportAllUsers(ctx)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Write CSV Header
	_ = writer.Write([]string{
		"User ID", "Name", "Email", "Roll Number", "Hostel", "Mess", "Gender", "Team ID", "Created At",
	})

	for _, u := range users {
		roll := ""
		if u.RollNumber != nil {
			roll = *u.RollNumber
		}
		hostel := ""
		if u.Hostel != nil {
			hostel = *u.Hostel
		}
		mess := ""
		if u.Mess != nil {
			mess = *u.Mess
		}
		gender := ""
		if u.Gender != nil {
			gender = *u.Gender
		}
		teamID := ""
		if u.TeamID != nil {
			teamID = *u.TeamID
		}

		_ = writer.Write([]string{
			u.UserID.String(),
			u.Name,
			u.Email,
			roll,
			hostel,
			mess,
			gender,
			teamID,
			u.CreatedAt.Format(time.RFC3339),
		})
	}

	writer.Flush()
	return buf.Bytes(), nil
}

func (s *AdminService) ExportTeamsCSV(ctx context.Context) ([]byte, error) {
	teams, err := s.adminRepo.ExportAllTeams(ctx)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	_ = writer.Write([]string{
		"Team ID", "Team Name", "Leader Name", "Leader User ID", "Contact", "Domain", "Problem Statement", "Payment Status", "Is Public", "Created At",
	})

	for _, t := range teams {
		domain := ""
		if t.Domain != nil {
			domain = *t.Domain
		}
		ps := ""
		if t.ProblemStatement != nil {
			ps = *t.ProblemStatement
		}

		_ = writer.Write([]string{
			t.TeamID,
			t.Name,
			t.Leader,
			t.LeaderUserID.String(),
			t.Contact,
			domain,
			ps,
			t.PaymentStatus,
			fmt.Sprintf("%t", t.IsPublic),
			t.CreatedAt.Format(time.RFC3339),
		})
	}

	writer.Flush()
	return buf.Bytes(), nil
}

func (s *AdminService) ExportPaymentsCSV(ctx context.Context) ([]byte, error) {
	payments, err := s.adminRepo.ExportAllPayments(ctx)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	_ = writer.Write([]string{
		"Payment ID", "Order ID", "Team ID", "Team Name", "User Name", "User Email", "Amount", "Currency", "Status", "Transaction ID", "Created At",
	})

	for _, p := range payments {
		teamName := ""
		if p.TeamName != nil {
			teamName = *p.TeamName
		}
		userName := ""
		if p.UserName != nil {
			userName = *p.UserName
		}
		userEmail := ""
		if p.UserEmail != nil {
			userEmail = *p.UserEmail
		}
		txID := ""
		if p.TransactionID != nil {
			txID = *p.TransactionID
		}

		_ = writer.Write([]string{
			p.PaymentID,
			p.OrderID,
			p.TeamID,
			teamName,
			userName,
			userEmail,
			fmt.Sprintf("%.2f", p.Amount),
			p.Currency,
			p.PaymentStatus,
			txID,
			p.CreatedAt.Format(time.RFC3339),
		})
	}

	writer.Flush()
	return buf.Bytes(), nil
}
