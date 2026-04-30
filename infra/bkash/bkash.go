package bkash

import (
	"bytes"
	"encoding/json"
	"eraya/config"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	config config.BKashConfig
}

func NewClient(cfg config.BKashConfig) *Client {
	return &Client{
		config: cfg,
	}
}

type TokenResponse struct {
	IdToken   string `json:"id_token"`
	TokenType string `json:"token_type"`
	ExpiresIn int    `json:"expires_in"`
	Message   string `json:"statusMessage"`
}

func (c *Client) GetToken() (string, error) {
	url := c.config.BaseURL + "/tokenized/checkout/token/grant"
	reqBody, _ := json.Marshal(map[string]string{
		"app_key":    c.config.AppKey,
		"app_secret": c.config.AppSecret,
	})

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("username", c.config.Username)
	req.Header.Set("password", c.config.Password)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var tr TokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", err
	}

	if tr.IdToken == "" {
		return "", fmt.Errorf("failed to get token: %s", string(body))
	}

	return tr.IdToken, nil
}

type CreatePaymentResponse struct {
	StatusCode    string `json:"statusCode"`
	StatusMessage string `json:"statusMessage"`
	PaymentID     string `json:"paymentID"`
	BkashURL      string `json:"bkashURL"`
	CallbackURL   string `json:"callbackURL"`
	Amount        string `json:"amount"`
}

func (c *Client) CreatePayment(amount float64, invoiceNumber string, callbackURL string) (*CreatePaymentResponse, error) {
	token, err := c.GetToken()
	if err != nil {
		return nil, err
	}

	url := c.config.BaseURL + "/tokenized/checkout/create"
	reqBody, _ := json.Marshal(map[string]string{
		"mode":                  "0011",
		"payerReference":        "1",
		"callbackURL":           callbackURL,
		"amount":                fmt.Sprintf("%.2f", amount),
		"currency":              "BDT",
		"intent":                "sale",
		"merchantInvoiceNumber": invoiceNumber,
	})

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-app-key", c.config.AppKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var cpr CreatePaymentResponse
	if err := json.Unmarshal(body, &cpr); err != nil {
		return nil, err
	}

	if cpr.StatusCode != "0000" {
		return nil, fmt.Errorf("failed to create payment: %s", string(body))
	}

	return &cpr, nil
}

type ExecutePaymentResponse struct {
	StatusCode            string `json:"statusCode"`
	StatusMessage         string `json:"statusMessage"`
	PaymentID             string `json:"paymentID"`
	TrxID                 string `json:"trxID"`
	Amount                string `json:"amount"`
	MerchantInvoiceNumber string `json:"merchantInvoiceNumber"`
}

func (c *Client) ExecutePayment(paymentID string) (*ExecutePaymentResponse, error) {
	// Bypass API for mock sandbox payments
	if len(paymentID) > 5 && paymentID[:5] == "MOCK_" {
		return &ExecutePaymentResponse{
			StatusCode:            "0000",
			StatusMessage:         "Mock Payment Successful",
			PaymentID:             paymentID,
			TrxID:                 "TRXMOCK" + paymentID[5:],
			Amount:                "0",
			MerchantInvoiceNumber: paymentID[5:],
		}, nil
	}

	token, err := c.GetToken()
	if err != nil {
		return nil, err
	}

	url := c.config.BaseURL + "/tokenized/checkout/execute"
	reqBody, _ := json.Marshal(map[string]string{
		"paymentID": paymentID,
	})

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-app-key", c.config.AppKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var epr ExecutePaymentResponse
	if err := json.Unmarshal(body, &epr); err != nil {
		return nil, err
	}

	// 0000 means successful, 2023 means insufficient balance etc.
	if epr.StatusCode != "0000" {
		return nil, fmt.Errorf("failed to execute payment: %s", string(body))
	}

	return &epr, nil
}
