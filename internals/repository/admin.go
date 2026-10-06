package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/Anshul1310/tf-register/internals/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminRepository struct {
	dbPool *pgxpool.Pool
}

func NewAdminRepository(dbPool *pgxpool.Pool) *AdminRepository {
	return &AdminRepository{
		dbPool: dbPool,
	}
}

// -------------------------------------------------------------
// Admin User & Permission Methods
// -------------------------------------------------------------

func (r *AdminRepository) GetAdminByEmail(ctx context.Context, email string) (*models.AdminUser, error) {
	query := `
		SELECT admin_id, email, name, picture, role, can_manage_users, can_manage_teams, 
		       can_manage_payments, can_manage_admins, is_active, created_at, updated_at
		FROM admin_users
		WHERE LOWER(email) = LOWER($1)
	`
	row := r.dbPool.QueryRow(ctx, query, strings.TrimSpace(email))
	var admin models.AdminUser
	err := row.Scan(
		&admin.AdminID,
		&admin.Email,
		&admin.Name,
		&admin.Picture,
		&admin.Role,
		&admin.CanManageUsers,
		&admin.CanManageTeams,
		&admin.CanManagePayments,
		&admin.CanManageAdmins,
		&admin.IsActive,
		&admin.CreatedAt,
		&admin.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get admin by email: %w", err)
	}
	return &admin, nil
}

func (r *AdminRepository) GetAdminByID(ctx context.Context, adminID uuid.UUID) (*models.AdminUser, error) {
	query := `
		SELECT admin_id, email, name, picture, role, can_manage_users, can_manage_teams, 
		       can_manage_payments, can_manage_admins, is_active, created_at, updated_at
		FROM admin_users
		WHERE admin_id = $1
	`
	row := r.dbPool.QueryRow(ctx, query, adminID)
	var admin models.AdminUser
	err := row.Scan(
		&admin.AdminID,
		&admin.Email,
		&admin.Name,
		&admin.Picture,
		&admin.Role,
		&admin.CanManageUsers,
		&admin.CanManageTeams,
		&admin.CanManagePayments,
		&admin.CanManageAdmins,
		&admin.IsActive,
		&admin.CreatedAt,
		&admin.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get admin by ID: %w", err)
	}
	return &admin, nil
}

func (r *AdminRepository) CreateAdmin(ctx context.Context, admin *models.AdminUser) error {
	if admin.AdminID == uuid.Nil {
		admin.AdminID = uuid.New()
	}
	query := `
		INSERT INTO admin_users (admin_id, email, name, picture, role, can_manage_users, can_manage_teams, can_manage_payments, can_manage_admins, is_active, created_at, updated_at)
		VALUES ($1, LOWER($2), $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		ON CONFLICT (email) DO UPDATE SET
			name = EXCLUDED.name,
			picture = COALESCE(EXCLUDED.picture, admin_users.picture),
			role = EXCLUDED.role,
			can_manage_users = EXCLUDED.can_manage_users,
			can_manage_teams = EXCLUDED.can_manage_teams,
			can_manage_payments = EXCLUDED.can_manage_payments,
			can_manage_admins = EXCLUDED.can_manage_admins,
			is_active = EXCLUDED.is_active,
			updated_at = NOW()
	`
	_, err := r.dbPool.Exec(ctx, query,
		admin.AdminID,
		admin.Email,
		admin.Name,
		admin.Picture,
		admin.Role,
		admin.CanManageUsers,
		admin.CanManageTeams,
		admin.CanManagePayments,
		admin.CanManageAdmins,
		admin.IsActive,
	)
	if err != nil {
		return fmt.Errorf("failed to create admin: %w", err)
	}
	return nil
}

func (r *AdminRepository) UpdateAdmin(ctx context.Context, adminID uuid.UUID, req *models.UpdateAdminRequest) error {
	existing, err := r.GetAdminByID(ctx, adminID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("admin not found")
	}

	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}
	role := existing.Role
	if req.Role != nil {
		role = *req.Role
	}
	canUsers := existing.CanManageUsers
	if req.CanManageUsers != nil {
		canUsers = *req.CanManageUsers
	}
	canTeams := existing.CanManageTeams
	if req.CanManageTeams != nil {
		canTeams = *req.CanManageTeams
	}
	canPayments := existing.CanManagePayments
	if req.CanManagePayments != nil {
		canPayments = *req.CanManagePayments
	}
	canAdmins := existing.CanManageAdmins
	if req.CanManageAdmins != nil {
		canAdmins = *req.CanManageAdmins
	}
	isActive := existing.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	query := `
		UPDATE admin_users
		SET name = $1, role = $2, can_manage_users = $3, can_manage_teams = $4, 
		    can_manage_payments = $5, can_manage_admins = $6, is_active = $7, updated_at = NOW()
		WHERE admin_id = $8
	`
	_, err = r.dbPool.Exec(ctx, query, name, role, canUsers, canTeams, canPayments, canAdmins, isActive, adminID)
	if err != nil {
		return fmt.Errorf("failed to update admin: %w", err)
	}
	return nil
}

func (r *AdminRepository) DeleteAdmin(ctx context.Context, adminID uuid.UUID) error {
	query := `DELETE FROM admin_users WHERE admin_id = $1`
	_, err := r.dbPool.Exec(ctx, query, adminID)
	if err != nil {
		return fmt.Errorf("failed to delete admin: %w", err)
	}
	return nil
}

func (r *AdminRepository) ListAdmins(ctx context.Context) ([]models.AdminUser, error) {
	query := `
		SELECT admin_id, email, name, picture, role, can_manage_users, can_manage_teams, 
		       can_manage_payments, can_manage_admins, is_active, created_at, updated_at
		FROM admin_users
		ORDER BY created_at ASC
	`
	rows, err := r.dbPool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list admins: %w", err)
	}
	defer rows.Close()

	admins := make([]models.AdminUser, 0)
	for rows.Next() {
		var a models.AdminUser
		if err := rows.Scan(
			&a.AdminID,
			&a.Email,
			&a.Name,
			&a.Picture,
			&a.Role,
			&a.CanManageUsers,
			&a.CanManageTeams,
			&a.CanManagePayments,
			&a.CanManageAdmins,
			&a.IsActive,
			&a.CreatedAt,
			&a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan admin: %w", err)
		}
		admins = append(admins, a)
	}
	return admins, nil
}

// -------------------------------------------------------------
// Dashboard Statistics
// -------------------------------------------------------------

func (r *AdminRepository) GetDashboardStats(ctx context.Context) (*models.AdminDashboardStats, error) {
	stats := &models.AdminDashboardStats{
		DomainStats:        make(map[string]int),
		PaymentStatusStats: make(map[string]int),
	}

	// 1. Total users
	err := r.dbPool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&stats.TotalUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get total users: %w", err)
	}

	// 2. Users with and without teams
	_ = r.dbPool.QueryRow(ctx, `SELECT count(*) FROM users WHERE team_id IS NOT NULL AND team_id != ''`).Scan(&stats.UsersWithTeam)
	stats.UsersWithoutTeam = stats.TotalUsers - stats.UsersWithTeam

	// 3. Total teams & payment breakups
	_ = r.dbPool.QueryRow(ctx, `SELECT count(*) FROM teams`).Scan(&stats.TotalTeams)
	_ = r.dbPool.QueryRow(ctx, `SELECT count(*) FROM teams WHERE LOWER(payment_status) IN ('paid', 'success')`).Scan(&stats.PaidTeams)
	_ = r.dbPool.QueryRow(ctx, `SELECT count(*) FROM teams WHERE LOWER(payment_status) NOT IN ('paid', 'success')`).Scan(&stats.PendingTeams)
	_ = r.dbPool.QueryRow(ctx, `SELECT count(*) FROM teams WHERE ispublic = true`).Scan(&stats.PublicTeams)

	// 4. Total revenue from successful payments or paid teams
	var revenue *float64
	_ = r.dbPool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) 
		FROM payments 
		WHERE LOWER(payment_status) IN ('paid', 'success')
	`).Scan(&revenue)
	if revenue != nil && *revenue > 0 {
		stats.TotalRevenue = *revenue
	} else {
		// Fallback: estimate from paid teams * 200 (default)
		stats.TotalRevenue = float64(stats.PaidTeams * 200)
	}

	// 5. Domain stats
	domainRows, err := r.dbPool.Query(ctx, `
		SELECT COALESCE(NULLIF(domain, ''), 'Unspecified') as dom, count(*) 
		FROM teams 
		GROUP BY dom
	`)
	if err == nil {
		defer domainRows.Close()
		for domainRows.Next() {
			var dom string
			var cnt int
			if err := domainRows.Scan(&dom, &cnt); err == nil {
				stats.DomainStats[dom] = cnt
			}
		}
	}

	// 6. Payment status stats
	paymentRows, err := r.dbPool.Query(ctx, `
		SELECT COALESCE(NULLIF(payment_status, ''), 'Pending') as pstatus, count(*) 
		FROM teams 
		GROUP BY pstatus
	`)
	if err == nil {
		defer paymentRows.Close()
		for paymentRows.Next() {
			var ps string
			var cnt int
			if err := paymentRows.Scan(&ps, &cnt); err == nil {
				stats.PaymentStatusStats[ps] = cnt
			}
		}
	}

	// 7. Recent registrations (last 10 users)
	recentUserRows, err := r.dbPool.Query(ctx, `
		SELECT user_id, name, email, roll_number, hostel, mess, gender, pfp, team_id, created_at
		FROM users
		ORDER BY created_at DESC
		LIMIT 10
	`)
	if err == nil {
		defer recentUserRows.Close()
		for recentUserRows.Next() {
			var u models.User
			if err := recentUserRows.Scan(
				&u.UserID, &u.Name, &u.Email, &u.RollNumber, &u.Hostel, &u.Mess, &u.Gender, &u.Pfp, &u.TeamID, &u.CreatedAt,
			); err == nil {
				stats.RecentRegistrations = append(stats.RecentRegistrations, u)
			}
		}
	}

	// 8. Recent payments (last 10)
	recentPaymentRows, err := r.dbPool.Query(ctx, `
		SELECT p.payment_id, p.order_id, p.team_id, t.name as team_name, p.user_id, u.name as user_name, u.email as user_email,
		       p.amount, p.currency, p.payment_status, p.payment_session_id, p.transaction_id, p.screenshot_url, p.created_at, p.updated_at
		FROM payments p
		LEFT JOIN teams t ON p.team_id = t.team_id
		LEFT JOIN users u ON p.user_id = u.user_id
		ORDER BY p.created_at DESC
		LIMIT 10
	`)
	if err == nil {
		defer recentPaymentRows.Close()
		for recentPaymentRows.Next() {
			var p models.Payment
			if err := recentPaymentRows.Scan(
				&p.PaymentID, &p.OrderID, &p.TeamID, &p.TeamName, &p.UserID, &p.UserName, &p.UserEmail,
				&p.Amount, &p.Currency, &p.PaymentStatus, &p.PaymentSessionID, &p.TransactionID, &p.ScreenshotURL,
				&p.CreatedAt, &p.UpdatedAt,
			); err == nil {
				stats.RecentPayments = append(stats.RecentPayments, p)
			}
		}
	}

	return stats, nil
}

// -------------------------------------------------------------
// User Management
// -------------------------------------------------------------

func (r *AdminRepository) ListUsers(ctx context.Context, search, hostel, mess, gender, hasTeam string, page, limit int) ([]models.User, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var whereClauses []string
	var args []interface{}
	argIdx := 1

	if search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(LOWER(name) LIKE LOWER($%d) OR LOWER(email) LIKE LOWER($%d) OR LOWER(COALESCE(roll_number, '')) LIKE LOWER($%d) OR LOWER(COALESCE(team_id, '')) LIKE LOWER($%d))", argIdx, argIdx, argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if hostel != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("LOWER(hostel) = LOWER($%d)", argIdx))
		args = append(args, hostel)
		argIdx++
	}
	if mess != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("LOWER(mess) = LOWER($%d)", argIdx))
		args = append(args, mess)
		argIdx++
	}
	if gender != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("LOWER(gender) = LOWER($%d)", argIdx))
		args = append(args, gender)
		argIdx++
	}
	if hasTeam == "true" {
		whereClauses = append(whereClauses, "team_id IS NOT NULL AND team_id != ''")
	} else if hasTeam == "false" {
		whereClauses = append(whereClauses, "(team_id IS NULL OR team_id = '')")
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count query
	countQuery := "SELECT count(*) FROM users" + whereSQL
	var totalCount int
	err := r.dbPool.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Data query
	dataQuery := fmt.Sprintf(`
		SELECT user_id, name, email, roll_number, hostel, mess, gender, pfp, team_id, created_at
		FROM users
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)
	rows, err := r.dbPool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(
			&u.UserID, &u.Name, &u.Email, &u.RollNumber, &u.Hostel, &u.Mess, &u.Gender, &u.Pfp, &u.TeamID, &u.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, totalCount, nil
}

func (r *AdminRepository) GetUserWithTeam(ctx context.Context, userID uuid.UUID) (*models.User, *models.Team, error) {
	userQuery := `
		SELECT user_id, name, email, roll_number, hostel, mess, gender, pfp, team_id, created_at
		FROM users
		WHERE user_id = $1
	`
	var u models.User
	err := r.dbPool.QueryRow(ctx, userQuery, userID).Scan(
		&u.UserID, &u.Name, &u.Email, &u.RollNumber, &u.Hostel, &u.Mess, &u.Gender, &u.Pfp, &u.TeamID, &u.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}

	var team *models.Team
	if u.TeamID != nil && *u.TeamID != "" {
		teamQuery := `
			SELECT team_id, name, leader, leader_user_id, contact, domain, problem_statement, payment_status, ispublic, created_at
			FROM teams
			WHERE team_id = $1
		`
		var t models.Team
		err := r.dbPool.QueryRow(ctx, teamQuery, *u.TeamID).Scan(
			&t.TeamID, &t.Name, &t.Leader, &t.LeaderUserID, &t.Contact, &t.Domain, &t.ProblemStatement, &t.PaymentStatus, &t.IsPublic, &t.CreatedAt,
		)
		if err == nil {
			team = &t
		}
	}

	return &u, team, nil
}

func (r *AdminRepository) UpdateUser(ctx context.Context, userID uuid.UUID, req *models.UpdateUserAdminRequest) error {
	query := `
		UPDATE users
		SET name = $1, email = $2, roll_number = $3, hostel = $4, mess = $5, gender = $6, team_id = $7
		WHERE user_id = $8
	`
	_, err := r.dbPool.Exec(ctx, query, req.Name, req.Email, req.RollNumber, req.Hostel, req.Mess, req.Gender, req.TeamID, userID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (r *AdminRepository) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	// If user is leader of a team, disassociate or handle
	_, _ = r.dbPool.Exec(ctx, `UPDATE teams SET leader_user_id = NULL WHERE leader_user_id = $1`, userID)
	_, err := r.dbPool.Exec(ctx, `DELETE FROM users WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// -------------------------------------------------------------
// Team Management
// -------------------------------------------------------------

func (r *AdminRepository) ListTeams(ctx context.Context, search, domain, paymentStatus, isPublic string, page, limit int) ([]models.Team, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var whereClauses []string
	var args []interface{}
	argIdx := 1

	if search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(LOWER(name) LIKE LOWER($%d) OR LOWER(team_id) LIKE LOWER($%d) OR LOWER(leader) LIKE LOWER($%d))", argIdx, argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if domain != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("LOWER(domain) = LOWER($%d)", argIdx))
		args = append(args, domain)
		argIdx++
	}
	if paymentStatus != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("LOWER(payment_status) = LOWER($%d)", argIdx))
		args = append(args, paymentStatus)
		argIdx++
	}
	if isPublic == "true" {
		whereClauses = append(whereClauses, "ispublic = true")
	} else if isPublic == "false" {
		whereClauses = append(whereClauses, "ispublic = false")
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	countQuery := "SELECT count(*) FROM teams" + whereSQL
	var totalCount int
	err := r.dbPool.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count teams: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT team_id, name, leader, leader_user_id, contact, domain, problem_statement, payment_status, ispublic, created_at
		FROM teams
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)
	rows, err := r.dbPool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query teams: %w", err)
	}
	defer rows.Close()

	teams := make([]models.Team, 0)
	for rows.Next() {
		var t models.Team
		if err := rows.Scan(
			&t.TeamID, &t.Name, &t.Leader, &t.LeaderUserID, &t.Contact, &t.Domain, &t.ProblemStatement, &t.PaymentStatus, &t.IsPublic, &t.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan team: %w", err)
		}

		// Fetch members for this team
		memberRows, err := r.dbPool.Query(ctx, `
			SELECT user_id, name, email, roll_number, hostel, mess, gender, pfp, team_id, created_at
			FROM users
			WHERE team_id = $1
		`, t.TeamID)
		if err == nil {
			for memberRows.Next() {
				var u models.User
				if err := memberRows.Scan(&u.UserID, &u.Name, &u.Email, &u.RollNumber, &u.Hostel, &u.Mess, &u.Gender, &u.Pfp, &u.TeamID, &u.CreatedAt); err == nil {
					t.Members = append(t.Members, u)
				}
			}
			memberRows.Close()
		}

		teams = append(teams, t)
	}
	return teams, totalCount, nil
}

func (r *AdminRepository) GetTeamDetails(ctx context.Context, teamID string) (*models.Team, error) {
	teamQuery := `
		SELECT team_id, name, leader, leader_user_id, contact, domain, problem_statement, payment_status, ispublic, created_at
		FROM teams
		WHERE team_id = $1 OR LOWER(name) = LOWER($1)
	`
	var t models.Team
	err := r.dbPool.QueryRow(ctx, teamQuery, teamID).Scan(
		&t.TeamID, &t.Name, &t.Leader, &t.LeaderUserID, &t.Contact, &t.Domain, &t.ProblemStatement, &t.PaymentStatus, &t.IsPublic, &t.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	memberRows, err := r.dbPool.Query(ctx, `
		SELECT user_id, name, email, roll_number, hostel, mess, gender, pfp, team_id, created_at
		FROM users
		WHERE team_id = $1
		ORDER BY created_at ASC
	`, t.TeamID)
	if err == nil {
		defer memberRows.Close()
		for memberRows.Next() {
			var u models.User
			if err := memberRows.Scan(&u.UserID, &u.Name, &u.Email, &u.RollNumber, &u.Hostel, &u.Mess, &u.Gender, &u.Pfp, &u.TeamID, &u.CreatedAt); err == nil {
				t.Members = append(t.Members, u)
			}
		}
	}

	return &t, nil
}

func (r *AdminRepository) CreateTeam(ctx context.Context, team *models.Team) error {
	query := `
		INSERT INTO teams (team_id, name, leader, leader_user_id, contact, domain, problem_statement, payment_status, ispublic, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`
	_, err := r.dbPool.Exec(ctx, query,
		team.TeamID, team.Name, team.Leader, team.LeaderUserID, team.Contact, team.Domain, team.ProblemStatement, team.PaymentStatus, team.IsPublic,
	)
	if err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}
	return nil
}

func (r *AdminRepository) UpdateTeam(ctx context.Context, teamID string, req *models.UpdateTeamAdminRequest) error {
	existing, err := r.GetTeamDetails(ctx, teamID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("team not found")
	}

	name := existing.Name
	if req.Name != "" {
		name = req.Name
	}
	contact := existing.Contact
	if req.Contact != "" {
		contact = req.Contact
	}
	paymentStatus := existing.PaymentStatus
	if req.PaymentStatus != "" {
		paymentStatus = req.PaymentStatus
	}
	leader := existing.Leader
	if req.LeaderName != nil && *req.LeaderName != "" {
		leader = *req.LeaderName
	}
	leaderUserID := existing.LeaderUserID
	if req.LeaderUserID != nil && *req.LeaderUserID != uuid.Nil {
		leaderUserID = *req.LeaderUserID
	}

	query := `
		UPDATE teams
		SET name = $1, contact = $2, domain = $3, problem_statement = $4, ispublic = $5, payment_status = $6, leader = $7, leader_user_id = $8
		WHERE team_id = $9
	`
	_, err = r.dbPool.Exec(ctx, query, name, contact, req.Domain, req.ProblemStatement, req.IsPublic, paymentStatus, leader, leaderUserID, existing.TeamID)
	if err != nil {
		return fmt.Errorf("failed to update team: %w", err)
	}
	return nil
}

func (r *AdminRepository) UpdateTeamPaymentStatus(ctx context.Context, teamID string, status string) error {
	query := `UPDATE teams SET payment_status = $1 WHERE team_id = $2`
	_, err := r.dbPool.Exec(ctx, query, status, teamID)
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}
	return nil
}

func (r *AdminRepository) DeleteTeam(ctx context.Context, teamID string) error {
	// Remove all members first
	_, _ = r.dbPool.Exec(ctx, `UPDATE users SET team_id = NULL WHERE team_id = $1`, teamID)
	// Delete payments and join requests
	_, _ = r.dbPool.Exec(ctx, `DELETE FROM payments WHERE team_id = $1`, teamID)
	_, _ = r.dbPool.Exec(ctx, `DELETE FROM team_join_requests WHERE team_id = $1`, teamID)
	// Delete team
	_, err := r.dbPool.Exec(ctx, `DELETE FROM teams WHERE team_id = $1`, teamID)
	if err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}
	return nil
}

func (r *AdminRepository) AddTeamMember(ctx context.Context, teamID string, userID uuid.UUID) error {
	query := `UPDATE users SET team_id = $1 WHERE user_id = $2`
	_, err := r.dbPool.Exec(ctx, query, teamID, userID)
	if err != nil {
		return fmt.Errorf("failed to add team member: %w", err)
	}
	return nil
}

func (r *AdminRepository) RemoveTeamMember(ctx context.Context, teamID string, userID uuid.UUID) error {
	// If the user being removed is leader, clear leader
	_, _ = r.dbPool.Exec(ctx, `UPDATE teams SET leader_user_id = NULL WHERE team_id = $1 AND leader_user_id = $2`, teamID, userID)
	query := `UPDATE users SET team_id = NULL WHERE user_id = $1 AND team_id = $2`
	_, err := r.dbPool.Exec(ctx, query, userID, teamID)
	if err != nil {
		return fmt.Errorf("failed to remove team member: %w", err)
	}
	return nil
}

func (r *AdminRepository) ChangeTeamLeader(ctx context.Context, teamID string, newLeaderUserID uuid.UUID) error {
	// Fetch new leader user info
	var name string
	err := r.dbPool.QueryRow(ctx, `SELECT name FROM users WHERE user_id = $1`, newLeaderUserID).Scan(&name)
	if err != nil {
		return fmt.Errorf("failed to fetch new leader name: %w", err)
	}

	// Update team leader and ensure leader is member of team
	_, err = r.dbPool.Exec(ctx, `
		UPDATE teams 
		SET leader_user_id = $1, leader = $2 
		WHERE team_id = $3
	`, newLeaderUserID, name, teamID)
	if err != nil {
		return fmt.Errorf("failed to update team leader: %w", err)
	}

	_, _ = r.dbPool.Exec(ctx, `UPDATE users SET team_id = $1 WHERE user_id = $2`, teamID, newLeaderUserID)
	return nil
}

// -------------------------------------------------------------
// Payment Management
// -------------------------------------------------------------

func (r *AdminRepository) ListPayments(ctx context.Context, search, status string, page, limit int) ([]models.Payment, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var whereClauses []string
	var args []interface{}
	argIdx := 1

	if search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(LOWER(p.order_id) LIKE LOWER($%d) OR LOWER(p.payment_id) LIKE LOWER($%d) OR LOWER(p.team_id) LIKE LOWER($%d) OR LOWER(COALESCE(p.transaction_id, '')) LIKE LOWER($%d) OR LOWER(COALESCE(t.name, '')) LIKE LOWER($%d))", argIdx, argIdx, argIdx, argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("LOWER(p.payment_status) = LOWER($%d)", argIdx))
		args = append(args, status)
		argIdx++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	countQuery := fmt.Sprintf(`
		SELECT count(*)
		FROM payments p
		LEFT JOIN teams t ON p.team_id = t.team_id
		%s
	`, whereSQL)

	var totalCount int
	err := r.dbPool.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count payments: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT p.payment_id, p.order_id, p.team_id, t.name as team_name, p.user_id, u.name as user_name, u.email as user_email,
		       p.amount, p.currency, p.payment_status, p.payment_session_id, p.transaction_id, p.screenshot_url, p.raw_webhook_data,
		       p.created_at, p.updated_at
		FROM payments p
		LEFT JOIN teams t ON p.team_id = t.team_id
		LEFT JOIN users u ON p.user_id = u.user_id
		%s
		ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)
	rows, err := r.dbPool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query payments: %w", err)
	}
	defer rows.Close()

	payments := make([]models.Payment, 0)
	for rows.Next() {
		var p models.Payment
		if err := rows.Scan(
			&p.PaymentID, &p.OrderID, &p.TeamID, &p.TeamName, &p.UserID, &p.UserName, &p.UserEmail,
			&p.Amount, &p.Currency, &p.PaymentStatus, &p.PaymentSessionID, &p.TransactionID, &p.ScreenshotURL, &p.RawWebhookData,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan payment: %w", err)
		}
		payments = append(payments, p)
	}
	return payments, totalCount, nil
}

func (r *AdminRepository) RecordManualPayment(ctx context.Context, req *models.ManualPaymentVerifyRequest) error {
	paymentID := "MANUAL_" + uuid.New().String()[:12]
	orderID := "ORDER_MANUAL_" + strings.ReplaceAll(req.TeamID, "-", "_")

	query := `
		INSERT INTO payments (payment_id, order_id, team_id, amount, currency, payment_status, transaction_id, raw_webhook_data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'INR', $5, $6, $7, NOW(), NOW())
		ON CONFLICT (payment_id) DO UPDATE SET
			payment_status = EXCLUDED.payment_status,
			transaction_id = EXCLUDED.transaction_id,
			raw_webhook_data = EXCLUDED.raw_webhook_data,
			updated_at = NOW()
	`
	notes := "Admin manual verification: " + req.Notes
	_, err := r.dbPool.Exec(ctx, query, paymentID, orderID, req.TeamID, req.Amount, req.PaymentStatus, req.TransactionID, notes)
	if err != nil {
		return fmt.Errorf("failed to record manual payment: %w", err)
	}

	// Also update team payment status
	var teamStatus string
	if strings.ToUpper(req.PaymentStatus) == "SUCCESS" || strings.ToUpper(req.PaymentStatus) == "PAID" {
		teamStatus = "Paid"
	} else {
		teamStatus = req.PaymentStatus
	}
	_ = r.UpdateTeamPaymentStatus(ctx, req.TeamID, teamStatus)

	return nil
}

// -------------------------------------------------------------
// Exports
// -------------------------------------------------------------

func (r *AdminRepository) ExportAllUsers(ctx context.Context) ([]models.User, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT user_id, name, email, roll_number, hostel, mess, gender, pfp, team_id, created_at
		FROM users
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to export users: %w", err)
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.UserID, &u.Name, &u.Email, &u.RollNumber, &u.Hostel, &u.Mess, &u.Gender, &u.Pfp, &u.TeamID, &u.CreatedAt); err == nil {
			users = append(users, u)
		}
	}
	return users, nil
}

func (r *AdminRepository) ExportAllTeams(ctx context.Context) ([]models.Team, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT team_id, name, leader, leader_user_id, contact, domain, problem_statement, payment_status, ispublic, created_at
		FROM teams
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to export teams: %w", err)
	}
	defer rows.Close()

	teams := make([]models.Team, 0)
	for rows.Next() {
		var t models.Team
		if err := rows.Scan(
			&t.TeamID, &t.Name, &t.Leader, &t.LeaderUserID, &t.Contact, &t.Domain, &t.ProblemStatement, &t.PaymentStatus, &t.IsPublic, &t.CreatedAt,
		); err == nil {
			teams = append(teams, t)
		}
	}
	return teams, nil
}

func (r *AdminRepository) ExportAllPayments(ctx context.Context) ([]models.Payment, error) {
	rows, err := r.dbPool.Query(ctx, `
		SELECT p.payment_id, p.order_id, p.team_id, t.name as team_name, p.user_id, u.name as user_name, u.email as user_email,
		       p.amount, p.currency, p.payment_status, p.payment_session_id, p.transaction_id, p.screenshot_url, p.raw_webhook_data,
		       p.created_at, p.updated_at
		FROM payments p
		LEFT JOIN teams t ON p.team_id = t.team_id
		LEFT JOIN users u ON p.user_id = u.user_id
		ORDER BY p.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to export payments: %w", err)
	}
	defer rows.Close()

	payments := make([]models.Payment, 0)
	for rows.Next() {
		var p models.Payment
		if err := rows.Scan(
			&p.PaymentID, &p.OrderID, &p.TeamID, &p.TeamName, &p.UserID, &p.UserName, &p.UserEmail,
			&p.Amount, &p.Currency, &p.PaymentStatus, &p.PaymentSessionID, &p.TransactionID, &p.ScreenshotURL, &p.RawWebhookData,
			&p.CreatedAt, &p.UpdatedAt,
		); err == nil {
			payments = append(payments, p)
		}
	}
	return payments, nil
}
