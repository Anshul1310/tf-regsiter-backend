package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Anshul1310/tf-register/internals/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	databasePool *pgxpool.Pool
}

func NewUserRepository(databasePool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		databasePool: databasePool,
	}
}

func (userRepository *UserRepository) FindUserByID(requestContext context.Context, userID uuid.UUID) (*models.User, error) {
	selectQuery := `
		SELECT user_id, COALESCE(name, ''), COALESCE(email, ''), roll_number, hostel, mess, gender, pfp, team_id, created_at
		FROM users
		WHERE user_id = $1;
	`

	userRecord := &models.User{}
	queryError := userRepository.databasePool.QueryRow(requestContext, selectQuery, userID).Scan(
		&userRecord.UserID,
		&userRecord.Name,
		&userRecord.Email,
		&userRecord.RollNumber,
		&userRecord.Hostel,
		&userRecord.Mess,
		&userRecord.Gender,
		&userRecord.Pfp,
		&userRecord.TeamID,
		&userRecord.CreatedAt,
	)

	if queryError != nil {
		if errors.Is(queryError, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query user by id: %w", queryError)
	}

	return userRecord, nil
}

func (userRepository *UserRepository) FindUserByEmail(requestContext context.Context, emailAddress string) (*models.User, error) {
	selectQuery := `
		SELECT user_id, COALESCE(name, ''), COALESCE(email, ''), roll_number, hostel, mess, gender, pfp, team_id, created_at
		FROM users
		WHERE email = $1;
	`

	userRecord := &models.User{}
	queryError := userRepository.databasePool.QueryRow(requestContext, selectQuery, emailAddress).Scan(
		&userRecord.UserID,
		&userRecord.Name,
		&userRecord.Email,
		&userRecord.RollNumber,
		&userRecord.Hostel,
		&userRecord.Mess,
		&userRecord.Gender,
		&userRecord.Pfp,
		&userRecord.TeamID,
		&userRecord.CreatedAt,
	)

	if queryError != nil {
		if errors.Is(queryError, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query user by email: %w", queryError)
	}

	return userRecord, nil
}

func (userRepository *UserRepository) UpsertUser(requestContext context.Context, userToSave *models.User) (*models.User, error) {
	upsertQuery := `
		INSERT INTO users (user_id, name, email, roll_number, gender, pfp)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			name = CASE WHEN users.name IS NULL OR users.name = '' THEN EXCLUDED.name ELSE users.name END,
			email = CASE WHEN EXCLUDED.email IS NOT NULL AND EXCLUDED.email != '' THEN EXCLUDED.email ELSE users.email END,
			roll_number = CASE WHEN users.roll_number IS NULL OR users.roll_number = '' THEN EXCLUDED.roll_number ELSE users.roll_number END,
			gender = CASE WHEN users.gender IS NULL OR users.gender = '' THEN EXCLUDED.gender ELSE users.gender END,
			pfp = CASE WHEN users.pfp IS NULL OR users.pfp = '' THEN EXCLUDED.pfp ELSE users.pfp END
		RETURNING user_id, COALESCE(name, ''), COALESCE(email, ''), roll_number, hostel, mess, gender, pfp, team_id, created_at;
	`

	savedUser := &models.User{}
	queryError := userRepository.databasePool.QueryRow(
		requestContext,
		upsertQuery,
		userToSave.UserID,
		userToSave.Name,
		userToSave.Email,
		userToSave.RollNumber,
		userToSave.Gender,
		userToSave.Pfp,
	).Scan(
		&savedUser.UserID,
		&savedUser.Name,
		&savedUser.Email,
		&savedUser.RollNumber,
		&savedUser.Hostel,
		&savedUser.Mess,
		&savedUser.Gender,
		&savedUser.Pfp,
		&savedUser.TeamID,
		&savedUser.CreatedAt,
	)

	if queryError != nil {
		return nil, fmt.Errorf("failed to upsert user: %w", queryError)
	}

	return savedUser, nil
}

func (userRepository *UserRepository) UpdateUserProfile(requestContext context.Context, userProfile *models.User) (*models.User, error) {
	updateQuery := `
		UPDATE users
		SET
			name = COALESCE(NULLIF($2, ''), name),
			roll_number = $3,
			hostel = $4,
			mess = $5,
			gender = $6,
			email = COALESCE(NULLIF($7, ''), email)
		WHERE user_id = $1
		RETURNING user_id, COALESCE(name, ''), COALESCE(email, ''), roll_number, hostel, mess, gender, pfp, team_id, created_at;
	`

	updatedUser := &models.User{}
	queryError := userRepository.databasePool.QueryRow(
		requestContext,
		updateQuery,
		userProfile.UserID,
		userProfile.Name,
		userProfile.RollNumber,
		userProfile.Hostel,
		userProfile.Mess,
		userProfile.Gender,
		userProfile.Email,
	).Scan(
		&updatedUser.UserID,
		&updatedUser.Name,
		&updatedUser.Email,
		&updatedUser.RollNumber,
		&updatedUser.Hostel,
		&updatedUser.Mess,
		&updatedUser.Gender,
		&updatedUser.Pfp,
		&updatedUser.TeamID,
		&updatedUser.CreatedAt,
	)

	if queryError != nil {
		return nil, fmt.Errorf("failed to update user profile: %w", queryError)
	}

	return updatedUser, nil
}

func (userRepository *UserRepository) UpdateUserTeam(requestContext context.Context, targetUserID uuid.UUID, targetTeamID *string) error {
	updateQuery := `
		UPDATE users
		SET team_id = $2
		WHERE user_id = $1;
	`

	_, executionError := userRepository.databasePool.Exec(requestContext, updateQuery, targetUserID, targetTeamID)
	if executionError != nil {
		return fmt.Errorf("failed to update user team: %w", executionError)
	}

	return nil
}

func (userRepository *UserRepository) FindUsersByTeamID(requestContext context.Context, teamIdentifier string) ([]models.User, error) {
	selectQuery := `
		SELECT user_id, COALESCE(name, ''), COALESCE(email, ''), roll_number, hostel, mess, gender, pfp, team_id, created_at
		FROM users
		WHERE team_id = $1;
	`

	databaseRows, queryError := userRepository.databasePool.Query(requestContext, selectQuery, teamIdentifier)
	if queryError != nil {
		return nil, fmt.Errorf("failed to query users by team id: %w", queryError)
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
			return nil, fmt.Errorf("failed to scan member row: %w", scanError)
		}
		membersList = append(membersList, member)
	}

	iterationError := databaseRows.Err()
	if iterationError != nil {
		return nil, fmt.Errorf("error during member rows iteration: %w", iterationError)
	}

	return membersList, nil
}
