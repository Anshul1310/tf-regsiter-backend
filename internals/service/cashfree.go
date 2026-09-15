package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Anshul1310/tf-register/config"
	"github.com/Anshul1310/tf-register/internals/models"
	"github.com/Anshul1310/tf-register/internals/repository"
)

type CashfreeService struct {
	configuration  *config.Config
	teamRepository *repository.TeamRepository
	userRepository *repository.UserRepository
	httpClient     *http.Client
}

func NewCashfreeService(
	configuration *config.Config,
	teamRepository *repository.TeamRepository,
	userRepository *repository.UserRepository,
) *CashfreeService {
	return &CashfreeService{
		configuration:  configuration,
		teamRepository: teamRepository,
		userRepository: userRepository,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (cashfreeService *CashfreeService) getCashfreeBaseURL() string {
	if strings.ToUpper(cashfreeService.configuration.CashfreeEnvironment) == "SANDBOX" {
		return "https://sandbox.cashfree.com/pg"
	}
	return "https://api.cashfree.com/pg"
}

func (cashfreeService *CashfreeService) CreatePaymentOrder(
	requestContext context.Context,
	teamIdentifier string,
	userIdentifier string,
	teamName string,
) (*models.CashfreeOrderResponse, error) {
	teamRecord, findTeamError := cashfreeService.teamRepository.FindTeamByID(requestContext, teamIdentifier)
	if findTeamError != nil {
		return nil, fmt.Errorf("failed to verify team: %w", findTeamError)
	}
	if teamRecord == nil {
		return nil, errors.New("team not found")
	}

	memberCount, countError := cashfreeService.teamRepository.CountTeamMembers(requestContext, teamIdentifier)
	if countError != nil {
		return nil, fmt.Errorf("failed to count team members: %w", countError)
	}
	if memberCount < 4 && cashfreeService.configuration.CashfreeAppID != "" {
		return nil, fmt.Errorf("team needs at least 4 members to initiate payment (current members: %d)", memberCount)
	}

	if teamRecord.PaymentStatus == "PAID" {
		return nil, errors.New("payment has already been completed for this team")
	}

	// Prepare order identifiers
	uniqueOrderIdentifier := fmt.Sprintf("order_%s_%d", teamIdentifier, time.Now().Unix())
	customerPhone := strings.TrimSpace(teamRecord.Contact)
	if len(customerPhone) < 10 {
		customerPhone = "9999999999"
	}

	customerEmail := teamRecord.Leader
	if customerEmail == "" {
		customerEmail = "participant@transfinitte.com"
	}

	customerName := teamName
	if customerName == "" {
		customerName = teamRecord.Name
	}

	returnURL := fmt.Sprintf("%s/team/%s?order_id={order_id}", cashfreeService.configuration.FrontendURL, teamIdentifier)

	cashfreeOrderPayload := map[string]interface{}{
		"order_id":       uniqueOrderIdentifier,
		"order_amount":   cashfreeService.configuration.PaymentAmount,
		"order_currency": "INR",
		"customer_details": map[string]interface{}{
			"customer_id":    teamIdentifier,
			"customer_email": customerEmail,
			"customer_phone": customerPhone,
			"customer_name":  customerName,
		},
		"order_meta": map[string]interface{}{
			"return_url": returnURL,
		},
		"order_note": fmt.Sprintf("Registration fee for team %s", teamRecord.Name),
	}

	// If Cashfree credentials are not configured, generate a mock session ID for sandbox testing
	if cashfreeService.configuration.CashfreeAppID == "" || cashfreeService.configuration.CashfreeSecretKey == "" {
		log.Println("Cashfree API credentials not configured in environment; providing sandbox demonstration session")
		mockSessionID := fmt.Sprintf("session_mock_%s_%d", teamIdentifier, time.Now().Unix())
		_ = cashfreeService.teamRepository.RecordPaymentLog(
			requestContext,
			uniqueOrderIdentifier,
			uniqueOrderIdentifier,
			teamIdentifier,
			cashfreeService.configuration.PaymentAmount,
			"CREATED",
			mockSessionID,
			"",
			"",
			"",
		)
		return &models.CashfreeOrderResponse{
			PaymentSessionID: mockSessionID,
			OrderID:          uniqueOrderIdentifier,
			OrderStatus:      "ACTIVE",
			OrderAmount:      cashfreeService.configuration.PaymentAmount,
			OrderCurrency:    "INR",
		}, nil
	}

	serializedPayload, serializationError := json.Marshal(cashfreeOrderPayload)
	if serializationError != nil {
		return nil, fmt.Errorf("failed to serialize Cashfree order request: %w", serializationError)
	}

	ordersEndpointURL := fmt.Sprintf("%s/orders", cashfreeService.getCashfreeBaseURL())
	httpRequest, requestBuildError := http.NewRequestWithContext(
		requestContext,
		http.MethodPost,
		ordersEndpointURL,
		bytes.NewBuffer(serializedPayload),
	)
	if requestBuildError != nil {
		return nil, fmt.Errorf("failed to create http request: %w", requestBuildError)
	}

	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("x-client-id", cashfreeService.configuration.CashfreeAppID)
	httpRequest.Header.Set("x-client-secret", cashfreeService.configuration.CashfreeSecretKey)
	httpRequest.Header.Set("x-api-version", cashfreeService.configuration.CashfreeApiVersion)

	httpResponse, requestExecutionError := cashfreeService.httpClient.Do(httpRequest)
	if requestExecutionError != nil {
		return nil, fmt.Errorf("failed to execute request to Cashfree API: %w", requestExecutionError)
	}
	defer httpResponse.Body.Close()

	responseBodyBytes, readingError := io.ReadAll(httpResponse.Body)
	if readingError != nil {
		return nil, fmt.Errorf("failed to read Cashfree response body: %w", readingError)
	}

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		return nil, fmt.Errorf("Cashfree API returned error status %d: %s", httpResponse.StatusCode, string(responseBodyBytes))
	}

	var parsedCashfreeResponse struct {
		PaymentSessionID string  `json:"payment_session_id"`
		OrderID          string  `json:"order_id"`
		OrderStatus      string  `json:"order_status"`
		OrderAmount      float64 `json:"order_amount"`
		OrderCurrency    string  `json:"order_currency"`
	}

	deserializationError := json.Unmarshal(responseBodyBytes, &parsedCashfreeResponse)
	if deserializationError != nil {
		return nil, fmt.Errorf("failed to parse Cashfree response: %w", deserializationError)
	}

	// Record payment record in database
	recordError := cashfreeService.teamRepository.RecordPaymentLog(
		requestContext,
		parsedCashfreeResponse.OrderID,
		parsedCashfreeResponse.OrderID,
		teamIdentifier,
		parsedCashfreeResponse.OrderAmount,
		parsedCashfreeResponse.OrderStatus,
		parsedCashfreeResponse.PaymentSessionID,
		"",
		"",
		string(responseBodyBytes),
	)
	if recordError != nil {
		log.Printf("Warning: failed to record payment log: %v", recordError)
	}

	return &models.CashfreeOrderResponse{
		PaymentSessionID: parsedCashfreeResponse.PaymentSessionID,
		OrderID:          parsedCashfreeResponse.OrderID,
		OrderStatus:      parsedCashfreeResponse.OrderStatus,
		OrderAmount:      parsedCashfreeResponse.OrderAmount,
		OrderCurrency:    parsedCashfreeResponse.OrderCurrency,
	}, nil
}

func (cashfreeService *CashfreeService) VerifyPaymentOrder(
	requestContext context.Context,
	orderIdentifier string,
	teamIdentifier string,
) (bool, error) {
	if cashfreeService.configuration.CashfreeAppID == "" || cashfreeService.configuration.CashfreeSecretKey == "" {
		// If credentials are empty in development/test, simulate success
		log.Printf("Sandbox verification simulated for order: %s", orderIdentifier)
		_ = cashfreeService.teamRepository.UpdatePaymentStatus(requestContext, teamIdentifier, "PAID")
		return true, nil
	}

	orderEndpointURL := fmt.Sprintf("%s/orders/%s", cashfreeService.getCashfreeBaseURL(), orderIdentifier)
	httpRequest, requestBuildError := http.NewRequestWithContext(
		requestContext,
		http.MethodGet,
		orderEndpointURL,
		nil,
	)
	if requestBuildError != nil {
		return false, fmt.Errorf("failed to create verification request: %w", requestBuildError)
	}

	httpRequest.Header.Set("x-client-id", cashfreeService.configuration.CashfreeAppID)
	httpRequest.Header.Set("x-client-secret", cashfreeService.configuration.CashfreeSecretKey)
	httpRequest.Header.Set("x-api-version", cashfreeService.configuration.CashfreeApiVersion)

	httpResponse, requestExecutionError := cashfreeService.httpClient.Do(httpRequest)
	if requestExecutionError != nil {
		return false, fmt.Errorf("failed to execute verification request: %w", requestExecutionError)
	}
	defer httpResponse.Body.Close()

	responseBodyBytes, readingError := io.ReadAll(httpResponse.Body)
	if readingError != nil {
		return false, fmt.Errorf("failed to read verification response: %w", readingError)
	}

	if httpResponse.StatusCode != http.StatusOK {
		return false, fmt.Errorf("Cashfree verification returned status %d: %s", httpResponse.StatusCode, string(responseBodyBytes))
	}

	var parsedVerificationResponse struct {
		OrderStatus string  `json:"order_status"`
		OrderAmount float64 `json:"order_amount"`
	}

	jsonError := json.Unmarshal(responseBodyBytes, &parsedVerificationResponse)
	if jsonError != nil {
		return false, fmt.Errorf("failed to parse verification response: %w", jsonError)
	}

	isPaid := strings.ToUpper(parsedVerificationResponse.OrderStatus) == "PAID"
	if isPaid {
		updatePaymentStatusError := cashfreeService.teamRepository.UpdatePaymentStatus(requestContext, teamIdentifier, "PAID")
		if updatePaymentStatusError != nil {
			return false, fmt.Errorf("failed to update team payment status: %w", updatePaymentStatusError)
		}
	}

	return isPaid, nil
}

func (cashfreeService *CashfreeService) ProcessWebhook(
	requestContext context.Context,
	rawWebhookPayload []byte,
) error {
	var webhookData struct {
		Data struct {
			Order struct {
				OrderID     string  `json:"order_id"`
				OrderAmount float64 `json:"order_amount"`
				OrderStatus string  `json:"order_status"`
			} `json:"order"`
			Payment struct {
				PaymentStatus string `json:"payment_status"`
				PaymentTime   string `json:"payment_time"`
				CfPaymentID   int64  `json:"cf_payment_id"`
			} `json:"payment"`
			CustomerDetails struct {
				CustomerID string `json:"customer_id"`
			} `json:"customer_details"`
		} `json:"data"`
		Type string `json:"type"`
	}

	unmarshalError := json.Unmarshal(rawWebhookPayload, &webhookData)
	if unmarshalError != nil {
		return fmt.Errorf("failed to parse webhook json: %w", unmarshalError)
	}

	orderIdentifier := webhookData.Data.Order.OrderID
	teamIdentifier := webhookData.Data.CustomerDetails.CustomerID
	orderStatus := strings.ToUpper(webhookData.Data.Order.OrderStatus)
	paymentStatus := strings.ToUpper(webhookData.Data.Payment.PaymentStatus)

	if orderStatus == "PAID" || paymentStatus == "SUCCESS" {
		log.Printf("Received Cashfree webhook: Payment successful for team %s (order: %s)", teamIdentifier, orderIdentifier)
		if teamIdentifier != "" {
			_ = cashfreeService.teamRepository.UpdatePaymentStatus(requestContext, teamIdentifier, "PAID")
		}
	}

	return nil
}
