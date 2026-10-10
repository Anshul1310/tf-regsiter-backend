package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                string
	DatabaseURL         string
	JWTSecret           string
	AllowedOrigins      string
	FrontendURL         string
	TeamCap             int
	MinTeamMembers      int
	MaxTeamMembers      int
	PaymentAmount       float64
	CashfreeAppID       string
	CashfreeSecretKey   string
	CashfreeEnvironment string
	CashfreeApiVersion  string
	CashfreeNotifyURL   string
	DAuthClientID       string
	DAuthClientSecret   string
	DAuthRedirectURI    string
	AdminEmails         []string
	GoogleClientID      string
}

func LoadConfig() *Config {
	// Attempt to load .env file from common execution directories
	possibleEnvPaths := []string{
		".env",
		"backend/.env",
		"../.env",
		"../../.env",
		"../../backend/.env",
	}

	for _, envPath := range possibleEnvPaths {
		if err := godotenv.Load(envPath); err == nil {
			log.Printf("Loaded environment variables from: %s", envPath)
			break
		}
	}

	serverPort := os.Getenv("PORT")
	if serverPort == "" {
		serverPort = "8000"
	}

	databaseConnectionURL := os.Getenv("PG_URL")
	if databaseConnectionURL == "" {
		databaseConnectionURL = os.Getenv("DATABASE_URL")
	}

	jwtSecretKey := os.Getenv("JWT_SECRET")
	if jwtSecretKey == "" {
		jwtSecretKey = "tf-register-secret-key-2025"
	}

	frontendApplicationURL := "http://localhost:5173"
	if envFrontend := os.Getenv("FRONTEND_URL"); envFrontend != "" {
		frontendApplicationURL = envFrontend
	} else if envFrontend := os.Getenv("VITE_PROD_URL_FRONTEND"); envFrontend != "" {
		frontendApplicationURL = envFrontend
	}

	rawAllowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	originsMap := make(map[string]bool)

	// Hardcoded development and production origins
	defaultOrigins := []string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
		"http://localhost:5174",
		"http://127.0.0.1:5174",
		"http://localhost:5175",
		"http://127.0.0.1:5175",
		"http://localhost:5163",
		"http://127.0.0.1:5163",
		"http://localhost:3000",
		"http://127.0.0.1:3000",
		"http://localhost:3001",
		"http://127.0.0.1:3001",
		"http://localhost:4173",
		"http://127.0.0.1:4173",
		"http://localhost:4174",
		"http://127.0.0.1:4174",
		"https://tf-register-2025-rr5l.vercel.app",
		"https://tf-register-frontend.netlify.app",
	}
	for _, o := range defaultOrigins {
		trimmed := strings.TrimRight(strings.TrimSpace(o), "/")
		if trimmed != "" {
			originsMap[trimmed] = true
			originsMap[trimmed+"/"] = true
		}
	}

	if frontendApplicationURL != "" {
		trimmed := strings.TrimRight(strings.TrimSpace(frontendApplicationURL), "/")
		if trimmed != "" {
			originsMap[trimmed] = true
			originsMap[trimmed+"/"] = true
		}
	}

	if rawAllowedOrigins != "" {
		for _, part := range strings.Split(rawAllowedOrigins, ",") {
			trimmed := strings.TrimRight(strings.TrimSpace(part), "/")
			if trimmed != "" {
				originsMap[trimmed] = true
				originsMap[trimmed+"/"] = true
			}
		}
	}

	var combinedOrigins []string
	for origin := range originsMap {
		combinedOrigins = append(combinedOrigins, origin)
	}
	allowedOriginsList := strings.Join(combinedOrigins, ",")

	teamCapacityLimit := 50
	teamCapacityString := os.Getenv("TEAM_CAP")
	if teamCapacityString != "" {
		parsedCapacity, parseError := strconv.Atoi(strings.TrimSpace(teamCapacityString))
		if parseError == nil && parsedCapacity > 0 {
			teamCapacityLimit = parsedCapacity
		}
	}

	minMembersLimit := 1
	if minStr := os.Getenv("MIN_TEAM_MEMBERS"); minStr != "" {
		if parsedMin, err := strconv.Atoi(strings.TrimSpace(minStr)); err == nil && parsedMin > 0 {
			minMembersLimit = parsedMin
		}
	} else if minStr := os.Getenv("TEAM_MIN_MEMBERS"); minStr != "" {
		if parsedMin, err := strconv.Atoi(strings.TrimSpace(minStr)); err == nil && parsedMin > 0 {
			minMembersLimit = parsedMin
		}
	}

	maxMembersLimit := 5
	if maxStr := os.Getenv("MAX_TEAM_MEMBERS"); maxStr != "" {
		if parsedMax, err := strconv.Atoi(strings.TrimSpace(maxStr)); err == nil && parsedMax > 0 {
			maxMembersLimit = parsedMax
		}
	} else if maxStr := os.Getenv("TEAM_MAX_MEMBERS"); maxStr != "" {
		if parsedMax, err := strconv.Atoi(strings.TrimSpace(maxStr)); err == nil && parsedMax > 0 {
			maxMembersLimit = parsedMax
		}
	} else if maxStr := os.Getenv("TEAM_MEMBER_CAP"); maxStr != "" {
		if parsedMax, err := strconv.Atoi(strings.TrimSpace(maxStr)); err == nil && parsedMax > 0 {
			maxMembersLimit = parsedMax
		}
	}

	registrationPaymentAmount := 200.0
	paymentAmountString := os.Getenv("PAYMENT_AMOUNT")
	if paymentAmountString == "" {
		paymentAmountString = os.Getenv("REGISTRATION_FEE")
	}
	if paymentAmountString == "" {
		paymentAmountString = os.Getenv("TEAM_REGISTRATION_PRICE")
	}
	if paymentAmountString == "" {
		paymentAmountString = os.Getenv("REGISTRATION_PRICE")
	}
	if paymentAmountString != "" {
		parsedAmount, parseAmountError := strconv.ParseFloat(strings.TrimSpace(paymentAmountString), 64)
		if parseAmountError == nil && parsedAmount > 0 {
			registrationPaymentAmount = parsedAmount
		}
	}

	cashfreeApplicationIdentifier := strings.TrimSpace(os.Getenv("CASHFREE_APP_ID"))
	cashfreeSecretAPIKey := strings.TrimSpace(os.Getenv("CASHFREE_SECRET_KEY"))

	cashfreeEnvironmentMode := strings.ToUpper(strings.TrimSpace(os.Getenv("CASHFREE_ENV")))
	if cashfreeEnvironmentMode == "" {
		cashfreeEnvironmentMode = "SANDBOX"
	}

	cashfreeAPIVersionHeader := strings.TrimSpace(os.Getenv("CASHFREE_API_VERSION"))
	if cashfreeAPIVersionHeader == "" {
		cashfreeAPIVersionHeader = "2023-08-01"
	}

	cashfreeNotifyURL := strings.TrimSpace(os.Getenv("CASHFREE_NOTIFY_URL"))
	if cashfreeNotifyURL == "" {
		cashfreeNotifyURL = strings.TrimSpace(os.Getenv("WEBHOOK_URL"))
	}

	dauthClientID := strings.TrimSpace(os.Getenv("DAUTH_CLIENT_ID"))
	if dauthClientID == "" {
		dauthClientID = strings.TrimSpace(os.Getenv("VITE_DAUTH_CLIENT_ID"))
	}

	dauthClientSecret := strings.TrimSpace(os.Getenv("DAUTH_CLIENT_SECRET"))

	dauthRedirectURI := strings.TrimSpace(os.Getenv("DAUTH_REDIRECT_URI"))
	if dauthRedirectURI == "" {
		dauthRedirectURI = strings.TrimSpace(os.Getenv("VITE_DAUTH_REDIRECT_URI"))
	}

	adminEmailsRaw := os.Getenv("ADMIN_EMAILS")
	var adminEmailsList []string
	rootSuperadmins := []string{"transfinitte@gmail.com", "negi.anshulnegi17@gmail.com"}
	for _, root := range rootSuperadmins {
		adminEmailsList = append(adminEmailsList, strings.ToLower(strings.TrimSpace(root)))
	}

	if adminEmailsRaw != "" {
		for _, email := range strings.Split(adminEmailsRaw, ",") {
			trimmed := strings.ToLower(strings.TrimSpace(email))
			if trimmed != "" {
				exists := false
				for _, existing := range adminEmailsList {
					if existing == trimmed {
						exists = true
						break
					}
				}
				if !exists {
					adminEmailsList = append(adminEmailsList, trimmed)
				}
			}
		}
	}

	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	if googleClientID == "" {
		googleClientID = os.Getenv("VITE_GOOGLE_CLIENT_ID")
	}

	log.Printf("Loaded configuration with port: %s, cashfree mode: %s, dauth client: %s, admin emails: %v", serverPort, cashfreeEnvironmentMode, dauthClientID, adminEmailsList)

	return &Config{
		Port:                serverPort,
		DatabaseURL:         databaseConnectionURL,
		JWTSecret:           jwtSecretKey,
		AllowedOrigins:      allowedOriginsList,
		FrontendURL:         frontendApplicationURL,
		TeamCap:             teamCapacityLimit,
		MinTeamMembers:      minMembersLimit,
		MaxTeamMembers:      maxMembersLimit,
		PaymentAmount:       registrationPaymentAmount,
		CashfreeAppID:       cashfreeApplicationIdentifier,
		CashfreeSecretKey:   cashfreeSecretAPIKey,
		CashfreeEnvironment: cashfreeEnvironmentMode,
		CashfreeApiVersion:  cashfreeAPIVersionHeader,
		CashfreeNotifyURL:   cashfreeNotifyURL,
		DAuthClientID:       dauthClientID,
		DAuthClientSecret:   dauthClientSecret,
		DAuthRedirectURI:    dauthRedirectURI,
		AdminEmails:         adminEmailsList,
		GoogleClientID:      googleClientID,
	}
}
