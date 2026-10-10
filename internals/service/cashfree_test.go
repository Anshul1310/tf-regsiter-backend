package service_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/Anshul1310/tf-register/config"
	"github.com/Anshul1310/tf-register/internals/service"
)

func TestVerifyWebhookSignature(t *testing.T) {
	testSecretKey := "cf_test_secret_key_12345"
	cfg := &config.Config{
		PaymentAmount:       250.0,
		CashfreeSecretKey:   testSecretKey,
		CashfreeAppID:       "cf_app_test",
		CashfreeEnvironment: "SANDBOX",
		CashfreeApiVersion:  "2023-08-01",
	}

	cashfreeService := service.NewCashfreeService(cfg, nil, nil)

	rawPayload := []byte(`{"data":{"order":{"order_id":"order_test_123","order_amount":250,"order_status":"PAID"},"payment":{"payment_status":"SUCCESS","cf_payment_id":998877},"customer_details":{"customer_id":"team_xyz"}},"type":"PAYMENT_SUCCESS_WEBHOOK"}`)
	timestamp := "1710000000"

	// 1. Correct signature generation
	messageToSign := timestamp + string(rawPayload)
	mac := hmac.New(sha256.New, []byte(testSecretKey))
	mac.Write([]byte(messageToSign))
	validSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// Verify valid signature passes
	err := cashfreeService.VerifyWebhookSignature(rawPayload, validSignature, timestamp)
	if err != nil {
		t.Fatalf("Expected valid signature to succeed, got error: %v", err)
	}

	// 2. Tampered signature should fail
	invalidSignature := "aW52YWxpZF9zaWduYXR1cmVfZXhhbXBsZQ=="
	err = cashfreeService.VerifyWebhookSignature(rawPayload, invalidSignature, timestamp)
	if err == nil {
		t.Fatalf("Expected invalid signature to fail, but it succeeded")
	}

	// 3. Tampered payload should fail
	tamperedPayload := []byte(`{"data":{"order":{"order_id":"order_test_123","order_amount":10,"order_status":"PAID"}}}`)
	err = cashfreeService.VerifyWebhookSignature(tamperedPayload, validSignature, timestamp)
	if err == nil {
		t.Fatalf("Expected tampered payload to fail signature verification, but it succeeded")
	}

	// 4. Missing headers should fail
	err = cashfreeService.VerifyWebhookSignature(rawPayload, "", timestamp)
	if err == nil {
		t.Fatalf("Expected missing signature header to fail")
	}

	err = cashfreeService.VerifyWebhookSignature(rawPayload, validSignature, "")
	if err == nil {
		t.Fatalf("Expected missing timestamp header to fail")
	}

	// 5. Configured payment amount and getters
	if cashfreeService.GetPaymentAmount() != 250.0 {
		t.Errorf("Expected payment amount 250.0, got %v", cashfreeService.GetPaymentAmount())
	}
	if cashfreeService.GetEnvironment() != "SANDBOX" {
		t.Errorf("Expected environment SANDBOX, got %v", cashfreeService.GetEnvironment())
	}
}
