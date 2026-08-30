// Package msg talks to a running messenger hub — the local, broker-free
// channel bridge (telegram / whatsapp / teams / webhook) that agents on this
// machine share. It speaks the hub's HTTP API rather than re-implementing any
// channel transport.
package msg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// DefaultBaseURL is where `messenger serve` listens.
const DefaultBaseURL = "http://127.0.0.1:14310"

// TokenEnv names the environment variable holding the hub's bearer token. The
// value is read straight into the Authorization header and never printed.
const TokenEnv = "MESSENGER_SERVE_TOKEN"

// Client is a hub connection.
type Client struct {
	BaseURL string
	token   string
	http    *http.Client
}

// New builds a client from the environment. AGENTIC_OS_MESSENGER_URL wins over
// MESSENGER_URL, so a session can point at a non-default hub without disturbing
// other tools that read the same variables.
func New(env func(string) string) *Client {
	base := env("AGENTIC_OS_MESSENGER_URL")
	if base == "" {
		base = env("MESSENGER_URL")
	}
	if base == "" {
		base = DefaultBaseURL
	}
	return &Client{
		BaseURL: base,
		token:   env(TokenEnv),
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Health is the hub's self-report.
type Health struct {
	OK       bool              `json:"ok"`
	Service  string            `json:"service"`
	Channels map[string]string `json:"channels"`
}

// SendRequest is one outbound message. Text or File is required.
type SendRequest struct {
	Channel string `json:"channel"`
	Text    string `json:"text,omitempty"`
	To      string `json:"to,omitempty"`
	ReplyTo string `json:"reply_to,omitempty"`
	File    string `json:"file,omitempty"`
}

// SendResponse carries the provider's message id, which threads a later reply.
type SendResponse struct {
	OK bool   `json:"ok"`
	ID string `json:"id"`
}

// Inbox is one page of inbound envelopes. Next is the cursor to pass as `since`
// on the following call; an unchanged Next means nothing new arrived.
type Inbox struct {
	Messages []json.RawMessage `json:"messages"`
	Next     int64             `json:"next"`
}

func (c *Client) Health(ctx context.Context) (Health, error) {
	var health Health
	err := c.do(ctx, http.MethodGet, "/health", nil, &health)
	return health, err
}

func (c *Client) Send(ctx context.Context, req SendRequest) (SendResponse, error) {
	var res SendResponse
	err := c.do(ctx, http.MethodPost, "/send", req, &res)
	return res, err
}

func (c *Client) Inbox(ctx context.Context, since int64) (Inbox, error) {
	var inbox Inbox
	path := "/inbox?" + url.Values{"since": {strconv.FormatInt(since, 10)}}.Encode()
	err := c.do(ctx, http.MethodGet, path, nil, &inbox)
	return inbox, err
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var payload *bytes.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(encoded)
	} else {
		payload = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, payload)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("messenger hub at %s: %w", c.BaseURL, err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var detail struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(res.Body).Decode(&detail)
		if detail.Error != "" {
			return fmt.Errorf("messenger hub: %s (%s)", detail.Error, res.Status)
		}
		return fmt.Errorf("messenger hub: %s", res.Status)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}

// HealthWithin is Health with its own deadline, for callers that want a fast
// answer about a hub that may simply not be running.
func (c *Client) HealthWithin(timeout time.Duration) (Health, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.Health(ctx)
}
