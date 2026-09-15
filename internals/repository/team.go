package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Anshul1310/tf-register/internals/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TeamRepository struct {
	databasePool *pgxpool.Pool
}

func NewTeamRepository(databasePool *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{
		databasePool: databasePool,
	}
}

func (teamRepository *TeamRepository) CreateTeamWithLeader(requestContext context.Context, teamToCreate *models.Team) error {
	databaseTransaction, transactionBeginError := teamRepository.databasePool.Begin(requestContext)
	if transactionBeginError != nil {
		return fmt.Errorf("failed to begin transaction: %w", transactionBeginError)
	}
	defer databaseTransaction.Rollback(requestContext)

	insertTeamQuery := `
		INSERT INTO teams (
			team_id,
			name,
			leader,
			leader_user_id,
			contact,
			problem_statement,
			domain,
			payment_status,
			ispublic
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);
	`

	_, insertError := databaseTransaction.Exec(
		requestContext,
		insertTeamQuery,
		teamToCreate.TeamID,
		teamToCreate.Name,
		teamToCreate.Leader,
		teamToCreate.LeaderUserID,
		teamToCreate.Contact,
		teamToCreate.ProblemStatement,
		teamToCreate.Domain,
		teamToCreate.PaymentStatus,
		teamToCreate.IsPublic,
	)
	if insertError != nil {
		return fmt.Errorf("failed to insert team: %w", insertError)
	}

	updateLeaderQuery := `
		UPDATE users
		SET team_id = $1
		WHERE user_id = $2;
	`

	_, updateError := databaseTransaction.Exec(
		requestContext,
		updateLeaderQuery,
		teamToCreate.TeamID,
		teamToCreate.LeaderUserID,
	)
	if updateError != nil {
		return fmt.Errorf("failed to update leader team id: %w", updateError)
	}

	commitError := databaseTransaction.Commit(requestContext)
	if commitError != nil {
		return fmt.Errorf("failed to commit team creation transaction: %w", commitError)
	}

	return nil
}

func (teamRepository *TeamRepository) FindTeamByID(requestContext context.Context, teamIdentifier string) (*models.Team, error) {
	selectQuery := `
		SELECT team_id, name, leader, leader_user_id, contact, domain, problem_statement, payment_status, ispublic, created_at
		FROM teams
		WHERE team_id = $1;
	`

	teamRecord := &models.Team{}
	queryError := teamRepository.databasePool.QueryRow(requestContext, selectQuery, teamIdentifier).Scan(
		&teamRecord.TeamID,
		&teamRecord.Name,
		&teamRecord.Leader,
		&teamRecord.LeaderUserID,
		&teamRecord.Contact,
		&teamRecord.Domain,
		&teamRecord.ProblemStatement,
		&teamRecord.PaymentStatus,
		&teamRecord.IsPublic,
		&teamRecord.CreatedAt,
	)

	if queryError != nil {
		if errors.Is(queryError, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find team by id: %w", queryError)
	}

	return teamRecord, nil
}

func (teamRepository *TeamRepository) FindTeamByIDWithMembers(requestContext context.Context, teamIdentifier string) (*models.Team, error) {
	teamRecord, findTeamError := teamRepository.FindTeamByID(requestContext, teamIdentifier)
	if findTeamError != nil {
		return nil, findTeamError
	}
	if teamRecord == nil {
		return nil, nil
	}

	selectMembersQuery := `
		SELECT user_id, name, email, roll_number, hostel, mess, gender, pfp, team_id, created_at
		FROM users
		WHERE team_id = $1;
	`

	databaseRows, queryError := teamRepository.databasePool.Query(requestContext, selectMembersQuery, teamIdentifier)
	if queryError != nil {
		return nil, fmt.Errorf("failed to query team members: %w", queryError)
	}
	defer databaseRows.Close()

	membersList := make([]models.User, 0)
	for databaseRows.Next() {
		member := models.User{}
		scanError := databaseRows.Scan(
			&member.UserID,
			&member.Name,
			&member.Email,
			&member.RollNumber,
			&member.Hostel,
			&member.Mess,
			&member.Gender,
			&member.Pfp,
			&member.TeamID,
			&member.CreatedAt,
		)
		if scanError != nil {
			return nil, fmt.Errorf("failed to scan team member row: %w", scanError)
		}
		membersList = append(membersList, member)
	}

	teamRecord.Members = membersList
	return teamRecord, nil
}

func (teamRepository *TeamRepository) FindTeamByName(requestContext context.Context, teamName string) (*models.Team, error) {
	selectQuery := `
		SELECT team_id, name, leader, leader_user_id, contact, domain, problem_statement, payment_status, ispublic, created_at
		FROM teams
		WHERE LOWER(name) = LOWER($1);
	`

	teamRecord := &models.Team{}
	queryError := teamRepository.databasePool.QueryRow(requestContext, selectQuery, teamName).Scan(
		&teamRecord.TeamID,
		&teamRecord.Name,
		&teamRecord.Leader,
		&teamRecord.LeaderUserID,
		&teamRecord.Contact,
		&teamRecord.Domain,
		&teamRecord.ProblemStatement,
		&teamRecord.PaymentStatus,
		&teamRecord.IsPublic,
		&teamRecord.CreatedAt,
	)

	if queryError != nil {
		if errors.Is(queryError, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find team by name: %w", queryError)
	}

	return teamRecord, nil
}

func (teamRepository *TeamRepository) FindPublicTeams(requestContext context.Context, domainFilter string, searchQuery string) ([]models.Team, error) {
	selectQuery := `
		SELECT team_id, name, leader, leader_user_id, contact, domain, problem_statement, payment_status, ispublic, created_at
		FROM teams
		WHERE ispublic = true
		  AND ($1 = '' OR domain = $1)
		  AND ($2 = '' OR LOWER(name) LIKE LOWER('%' || $2 || '%'))
		ORDER BY created_at DESC;
	`

	databaseRows, queryError := teamRepository.databasePool.Query(requestContext, selectQuery, domainFilter, searchQuery)
	if queryError != nil {
		return nil, fmt.Errorf("failed to query public teams: %w", queryError)
	}
	defer databaseRows.Close()

	publicTeamsList := make([]models.Team, 0)
	for databaseRows.Next() {
		team := models.Team{}
		scanError := databaseRows.Scan(
			&team.TeamID,
			&team.Name,
			&team.Leader,
			&team.LeaderUserID,
			&team.Contact,
			&team.Domain,
			&team.ProblemStatement,
			&team.PaymentStatus,
			&team.IsPublic,
			&team.CreatedAt,
		)
		if scanError != nil {
			return nil, fmt.Errorf("failed to scan public team row: %w", scanError)
		}
		publicTeamsList = append(publicTeamsList, team)
	}

	return publicTeamsList, nil
}

func (teamRepository *TeamRepository) CountTeamMembers(requestContext context.Context, teamIdentifier string) (int, error) {
	countQuery := `
		SELECT count(*)::integer
		FROM users
		WHERE team_id = $1;
	`

	memberCount := 0
	queryError := teamRepository.databasePool.QueryRow(requestContext, countQuery, teamIdentifier).Scan(&memberCount)
	if queryError != nil {
		return 0, fmt.Errorf("failed to count team members: %w", queryError)
	}

	return memberCount, nil
}

func (teamRepository *TeamRepository) CountPaidTeams(requestContext context.Context) (int, error) {
	countQuery := `
		SELECT count(*)::integer
		FROM teams
		WHERE payment_status = 'PAID';
	`

	paidTeamCount := 0
	queryError := teamRepository.databasePool.QueryRow(requestContext, countQuery).Scan(&paidTeamCount)
	if queryError != nil {
		return 0, fmt.Errorf("failed to count paid teams: %w", queryError)
	}

	return paidTeamCount, nil
}

func (teamRepository *TeamRepository) UpdateTeamVisibility(requestContext context.Context, teamIdentifier string, isPublic bool) error {
	updateQuery := `
		UPDATE teams
		SET ispublic = $2
		WHERE team_id = $1;
	`

	_, executionError := teamRepository.databasePool.Exec(requestContext, updateQuery, teamIdentifier, isPublic)
	if executionError != nil {
		return fmt.Errorf("failed to update team visibility: %w", executionError)
	}

	return nil
}

func (teamRepository *TeamRepository) UpdateTeamName(requestContext context.Context, teamIdentifier string, newTeamName string) error {
	updateQuery := `
		UPDATE teams
		SET name = $2
		WHERE team_id = $1;
	`

	_, executionError := teamRepository.databasePool.Exec(requestContext, updateQuery, teamIdentifier, newTeamName)
	if executionError != nil {
		return fmt.Errorf("failed to update team name: %w", executionError)
	}

	return nil
}

func (teamRepository *TeamRepository) UpdateTeamDetails(requestContext context.Context, teamIdentifier string, domain *string, problemStatement *string) error {
	updateQuery := `
		UPDATE teams
		SET
			domain = COALESCE($2, domain),
			problem_statement = COALESCE($3, problem_statement)
		WHERE team_id = $1;
	`

	_, executionError := teamRepository.databasePool.Exec(requestContext, updateQuery, teamIdentifier, domain, problemStatement)
	if executionError != nil {
		return fmt.Errorf("failed to update team details: %w", executionError)
	}

	return nil
}

func (teamRepository *TeamRepository) UpdateTeamID(requestContext context.Context, oldTeamID string, newTeamID string) error {
	databaseTransaction, transactionBeginError := teamRepository.databasePool.Begin(requestContext)
	if transactionBeginError != nil {
		return fmt.Errorf("failed to begin transaction: %w", transactionBeginError)
	}
	defer databaseTransaction.Rollback(requestContext)

	// Step 1: Find all members belonging to old team
	findMembersQuery := `SELECT user_id FROM users WHERE team_id = $1;`
	memberRows, queryMembersError := databaseTransaction.Query(requestContext, findMembersQuery, oldTeamID)
	if queryMembersError != nil {
		return fmt.Errorf("failed to query team members: %w", queryMembersError)
	}

	memberUserIDs := make([]interface{}, 0)
	for memberRows.Next() {
		var memberUUID interface{}
		if scanError := memberRows.Scan(&memberUUID); scanError == nil {
			memberUserIDs = append(memberUserIDs, memberUUID)
		}
	}
	memberRows.Close()

	// Step 2: Unlink users temporarily to prevent foreign key violation
	unlinkUsersQuery := `UPDATE users SET team_id = NULL WHERE team_id = $1;`
	_, unlinkError := databaseTransaction.Exec(requestContext, unlinkUsersQuery, oldTeamID)
	if unlinkError != nil {
		return fmt.Errorf("failed to unlink users from old team id: %w", unlinkError)
	}

	// Step 3: Update team_id in teams table
	updateTeamQuery := `
		UPDATE teams
		SET team_id = $2
		WHERE team_id = $1;
	`
	_, updateTeamError := databaseTransaction.Exec(requestContext, updateTeamQuery, oldTeamID, newTeamID)
	if updateTeamError != nil {
		return fmt.Errorf("failed to update team id: %w", updateTeamError)
	}

	// Step 4: Re-link all previous members with the new team_id
	relinkUsersQuery := `UPDATE users SET team_id = $1 WHERE user_id = $2;`
	for _, memberID := range memberUserIDs {
		_, relinkError := databaseTransaction.Exec(requestContext, relinkUsersQuery, newTeamID, memberID)
		if relinkError != nil {
			return fmt.Errorf("failed to relink member to new team id: %w", relinkError)
		}
	}

	commitError := databaseTransaction.Commit(requestContext)
	if commitError != nil {
		return fmt.Errorf("failed to commit team id update: %w", commitError)
	}

	return nil
}

func (teamRepository *TeamRepository) UpdatePaymentStatus(requestContext context.Context, teamIdentifier string, paymentStatus string) error {
	updateQuery := `
		UPDATE teams
		SET payment_status = $2
		WHERE team_id = $1;
	`

	_, executionError := teamRepository.databasePool.Exec(requestContext, updateQuery, teamIdentifier, paymentStatus)
	if executionError != nil {
		return fmt.Errorf("failed to update team payment status: %w", executionError)
	}

	return nil
}

func (teamRepository *TeamRepository) DeleteTeam(requestContext context.Context, teamIdentifier string) error {
	databaseTransaction, transactionBeginError := teamRepository.databasePool.Begin(requestContext)
	if transactionBeginError != nil {
		return fmt.Errorf("failed to begin transaction: %w", transactionBeginError)
	}
	defer databaseTransaction.Rollback(requestContext)

	// Step 1: Remove team association from members
	unlinkUsersQuery := `
		UPDATE users
		SET team_id = NULL
		WHERE team_id = $1;
	`
	_, unlinkError := databaseTransaction.Exec(requestContext, unlinkUsersQuery, teamIdentifier)
	if unlinkError != nil {
		return fmt.Errorf("failed to unlink users before deleting team: %w", unlinkError)
	}

	// Step 2: Delete team record
	deleteTeamQuery := `
		DELETE FROM teams
		WHERE team_id = $1;
	`
	_, deleteError := databaseTransaction.Exec(requestContext, deleteTeamQuery, teamIdentifier)
	if deleteError != nil {
		return fmt.Errorf("failed to delete team record: %w", deleteError)
	}

	commitError := databaseTransaction.Commit(requestContext)
	if commitError != nil {
		return fmt.Errorf("failed to commit team deletion: %w", commitError)
	}

	return nil
}

func (teamRepository *TeamRepository) RecordPaymentLog(
	requestContext context.Context,
	paymentIdentifier string,
	orderIdentifier string,
	teamIdentifier string,
	amount float64,
	paymentStatus string,
	paymentSessionID string,
	transactionID string,
	screenshotURL string,
	rawWebhookData string,
) error {
	insertPaymentQuery := `
		INSERT INTO payments (
			payment_id,
			order_id,
			team_id,
			amount,
			payment_status,
			payment_session_id,
			transaction_id,
			screenshot_url,
			raw_webhook_data,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
		ON CONFLICT (payment_id) DO UPDATE SET
			payment_status = EXCLUDED.payment_status,
			transaction_id = COALESCE(EXCLUDED.transaction_id, payments.transaction_id),
			raw_webhook_data = COALESCE(EXCLUDED.raw_webhook_data, payments.raw_webhook_data),
			updated_at = now();
	`

	_, executionError := teamRepository.databasePool.Exec(
		requestContext,
		insertPaymentQuery,
		paymentIdentifier,
		orderIdentifier,
		teamIdentifier,
		amount,
		paymentStatus,
		paymentSessionID,
		transactionID,
		screenshotURL,
		rawWebhookData,
	)
	if executionError != nil {
		return fmt.Errorf("failed to record payment log: %w", executionError)
	}

	return nil
}
