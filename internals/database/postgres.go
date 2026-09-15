package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var GlobalDatabasePool *pgxpool.Pool

func InitializeDatabasePool(databaseConnectionString string) (*pgxpool.Pool, error) {
	poolConfiguration, configurationError := pgxpool.ParseConfig(databaseConnectionString)
	if configurationError != nil {
		return nil, fmt.Errorf("failed to parse database configuration: %w", configurationError)
	}

	poolConfiguration.MaxConns = 25
	poolConfiguration.MinConns = 5
	poolConfiguration.MaxConnIdleTime = 15 * time.Minute
	poolConfiguration.MaxConnLifetime = 1 * time.Hour

	connectionContext, cancelFunction := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunction()

	databasePool, connectionError := pgxpool.NewWithConfig(connectionContext, poolConfiguration)
	if connectionError != nil {
		return nil, fmt.Errorf("failed to create database connection pool: %w", connectionError)
	}

	pingError := databasePool.Ping(connectionContext)
	if pingError != nil {
		databasePool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", pingError)
	}

	log.Println("Database connection pool established successfully")

	migrationError := RunAutomaticMigrations(databasePool)
	if migrationError != nil {
		log.Printf("Warning: Database migration warning: %v", migrationError)
	}

	GlobalDatabasePool = databasePool
	return databasePool, nil
}

func RunAutomaticMigrations(databasePool *pgxpool.Pool) error {
	migrationContext, cancelFunction := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelFunction()

	// 1. Create teams table if not exists
	createTeamsTableQuery := `
	CREATE TABLE IF NOT EXISTS teams (
		team_id text PRIMARY KEY,
		name text NOT NULL UNIQUE,
		leader text,
		leader_user_id uuid,
		contact text,
		problem_statement text,
		domain text,
		payment_status text DEFAULT 'Pending',
		ispublic boolean DEFAULT false,
		created_at timestamptz DEFAULT now()
	);`

	_, teamsTableError := databasePool.Exec(migrationContext, createTeamsTableQuery)
	if teamsTableError != nil {
		return fmt.Errorf("failed to execute teams table migration: %w", teamsTableError)
	}

	// 2. Create users table if not exists
	createUsersTableQuery := `
	CREATE TABLE IF NOT EXISTS users (
		user_id uuid PRIMARY KEY,
		name text,
		email text,
		roll_number text,
		hostel text,
		mess text,
		gender text,
		pfp text,
		team_id text REFERENCES teams(team_id) ON DELETE SET NULL,
		created_at timestamptz DEFAULT now()
	);`

	_, usersTableError := databasePool.Exec(migrationContext, createUsersTableQuery)
	if usersTableError != nil {
		return fmt.Errorf("failed to execute users table migration: %w", usersTableError)
	}

	// 3. Create index on users team_id
	createIndexQuery := `CREATE INDEX IF NOT EXISTS idx_users_team_id ON users(team_id);`
	_, indexError := databasePool.Exec(migrationContext, createIndexQuery)
	if indexError != nil {
		log.Printf("Notice: Index creation note: %v", indexError)
	}

	// 4. Create payments log table for audit & Cashfree reconciliation
	createPaymentsTableQuery := `
	CREATE TABLE IF NOT EXISTS payments (
		payment_id text PRIMARY KEY,
		order_id text NOT NULL,
		team_id text REFERENCES teams(team_id) ON DELETE CASCADE,
		user_id uuid REFERENCES users(user_id) ON DELETE SET NULL,
		amount numeric(10, 2) NOT NULL,
		currency text DEFAULT 'INR',
		payment_status text NOT NULL,
		payment_session_id text,
		transaction_id text,
		screenshot_url text,
		raw_webhook_data text,
		created_at timestamptz DEFAULT now(),
		updated_at timestamptz DEFAULT now()
	);`

	_, paymentsTableError := databasePool.Exec(migrationContext, createPaymentsTableQuery)
	if paymentsTableError != nil {
		log.Printf("Notice: Payments table creation note: %v", paymentsTableError)
	}

	log.Println("Database automatic migrations finished")
	return nil
}

func CloseDatabasePool(databasePool *pgxpool.Pool) {
	if databasePool != nil {
		databasePool.Close()
		log.Println("Database connection pool closed successfully")
	}
}
