package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
)

type SendGridProvider struct {
	Endpoint string
	APIKey   string
	Client   *http.Client
}

func (p *SendGridProvider) Name() string { return "sendgrid" }

func (p *SendGridProvider) Send(ctx context.Context, msg Message) error {
	payload := map[string]any{
		"template_id":           msg.Template,
		"personalizations":      []any{msg.Payload["to"]},
		"dynamic_template_data": msg.Payload["data"],
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint+"/mail/send", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("sendgrid temporary error: %s", resp.Status)
	}
	if resp.StatusCode >= 400 {
		return backoff.Permanent(fmt.Errorf("sendgrid permanent error: %s", resp.Status))
	}
	return nil
}
