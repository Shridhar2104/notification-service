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

type SESProvider struct {
	Endpoint string
	APIKey   string
	Client   *http.Client
}

func (p *SESProvider) Name() string { return "ses" }

func (p *SESProvider) Send(ctx context.Context, msg Message) error {
	payload := map[string]any{
		"template_id": msg.Template,
		"to":          msg.Payload["to"],
		"data":        msg.Payload["data"],
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint+"/send", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", p.APIKey)

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
		return fmt.Errorf("ses temporary error: %s", resp.Status)
	}
	if resp.StatusCode >= 400 {
		return backoff.Permanent(fmt.Errorf("ses permanent error: %s", resp.Status))
	}
	return nil
}
