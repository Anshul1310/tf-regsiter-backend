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
	PaymentAmount       float64
	CashfreeAppID       string
	CashfreeSecretKey   string
	CashfreeEnvironment string
	CashfreeApiVersion  string
	DAuthClientID       string
	DAuthClientSecret   string
	DAuthRedirectURI    string
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

	frontendApplicationURL := "https://tf-register-2025-rr5l.vercel.app"
	if envFrontend := os.Getenv("VITE_PROD_URL_FRONTEND"); envFrontend != "" {
		frontendApplicationURL = envFrontend
	}

	rawAllowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	originsMap := make(map[string]bool)

	// Hardcoded development and production origins
	defaultOrigins := []string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
		"http://localhost:5163",
		"http://127.0.0.1:5163",
		"http://localhost:3000",
		"http://127.0.0.1:3000",
		"http://localhost:4173",
		"http://127.0.0.1:4173",
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
		parsedCapacity, parseError := strconv.Atoi(teamCapacityString)
		if parseError == nil && parsedCapacity > 0 {
			teamCapacityLimit = parsedCapacity
		}
	}

	registrationPaymentAmount := 200.0
	paymentAmountString := os.Getenv("PAYMENT_AMOUNT")
	if paymentAmountString != "" {
		parsedAmount, parseAmountError := strconv.ParseFloat(paymentAmountString, 64)
		if parseAmountError == nil && parsedAmount > 0 {
			registrationPaymentAmount = parsedAmount
		}
	}

	cashfreeApplicationIdentifier := os.Getenv("CASHFREE_APP_ID")
	cashfreeSecretAPIKey := os.Getenv("CASHFREE_SECRET_KEY")

	cashfreeEnvironmentMode := os.Getenv("CASHFREE_ENV")
	if cashfreeEnvironmentMode == "" {
		cashfreeEnvironmentMode = "PRODUCTION"
	}

	cashfreeAPIVersionHeader := os.Getenv("CASHFREE_API_VERSION")
	if cashfreeAPIVersionHeader == "" {
		cashfreeAPIVersionHeader = "2023-08-01"
	}

	dauthClientID := os.Getenv("DAUTH_CLIENT_ID")
	if dauthClientID == "" {
		dauthClientID = os.Getenv("VITE_DAUTH_CLIENT_ID")
	}

	dauthClientSecret := os.Getenv("DAUTH_CLIENT_SECRET")

	dauthRedirectURI := os.Getenv("DAUTH_REDIRECT_URI")
	if dauthRedirectURI == "" {
		dauthRedirectURI = os.Getenv("VITE_DAUTH_REDIRECT_URI")
	}

	log.Printf("Loaded configuration with port: %s, cashfree mode: %s", serverPort, cashfreeEnvironmentMode)

	return &Config{
		Port:                serverPort,
		DatabaseURL:         databaseConnectionURL,
		JWTSecret:           jwtSecretKey,
		AllowedOrigins:      allowedOriginsList,
		FrontendURL:         frontendApplicationURL,
		TeamCap:             teamCapacityLimit,
		PaymentAmount:       registrationPaymentAmount,
		CashfreeAppID:       cashfreeApplicationIdentifier,
		CashfreeSecretKey:   cashfreeSecretAPIKey,
		CashfreeEnvironment: cashfreeEnvironmentMode,
		CashfreeApiVersion:  cashfreeAPIVersionHeader,
		DAuthClientID:       dauthClientID,
		DAuthClientSecret:   dauthClientSecret,
		DAuthRedirectURI:    dauthRedirectURI,
	}
}
