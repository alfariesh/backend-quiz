package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const resendAPIURL = "https://api.resend.com/emails"

type ResendClient struct {
	apiKey    string
	from      string
	replyTo   string
	http      *http.Client
}

func NewResendClient(apiKey, from, replyTo string) *ResendClient {
	return &ResendClient{
		apiKey:  apiKey,
		from:    from,
		replyTo: replyTo,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html,omitempty"`
	Text    string   `json:"text,omitempty"`
	ReplyTo string   `json:"reply_to,omitempty"`
}

type resendError struct {
	Message string `json:"message"`
	Name    string `json:"name"`
}

func (c *ResendClient) Send(ctx context.Context, msg Message) error {
	body := resendRequest{
		From:    c.from,
		To:      []string{msg.To},
		Subject: msg.Subject,
		HTML:    msg.HTMLBody,
		Text:    msg.TextBody,
		ReplyTo: c.replyTo,
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resendAPIURL, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("resend request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	var apiErr resendError
	_ = json.Unmarshal(respBody, &apiErr)
	if apiErr.Message != "" {
		return fmt.Errorf("resend %d: %s", resp.StatusCode, apiErr.Message)
	}
	return fmt.Errorf("resend %d: %s", resp.StatusCode, string(respBody))
}
