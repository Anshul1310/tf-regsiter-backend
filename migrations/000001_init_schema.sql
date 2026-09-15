-- ==============================================================================
-- TransfiNITTe 2025 - PostgreSQL Database Schema
-- Compatible with PostgreSQL 13+, Neon, Supabase, AWS RDS, and Local Postgres
-- ==============================================================================

-- 1. Create Teams Table
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
);

-- 2. Create Users Table
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
);

-- 3. Create Index for Member Lookups
CREATE INDEX IF NOT EXISTS idx_users_team_id ON users(team_id);

-- 4. Create Payments Log Table (for Cashfree tracking & audits)
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
);

-- 5. Helper Function: count_team_members (optional backward compatibility)
CREATE OR REPLACE FUNCTION count_team_members(team_id_input text)
RETURNS integer
LANGUAGE sql
STABLE
AS $$
    SELECT count(*)::integer FROM users WHERE team_id = team_id_input;
$$;
