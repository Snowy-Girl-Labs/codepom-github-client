package queue

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// CoreProvider implements the QueueProvider interface by forwarding directly
// to the central CodePom Core queue server via POST /api/v1/jobs/submit.
type CoreProvider struct {
	coreURL    string
	jwtToken   string
	httpClient *http.Client
}

func NewCoreProvider() *CoreProvider {
	coreURL := os.Getenv("CODEPOM_CORE_URL")
	if coreURL == "" {
		coreURL = "http://localhost:8000"
	}
	jwtToken := os.Getenv("CODEPOM_JWT_TOKEN")
	if jwtToken == "" {
		jwtToken = "production-jwt-token"
	}
	return &CoreProvider{
		coreURL:    coreURL,
		jwtToken:   jwtToken,
		httpClient: &http.Client{},
	}
}

type submitRequest struct {
	JobType        string          `json:"job_type"`
	Payload        json.RawMessage `json:"payload"`
	IdempotencyKey string          `json:"idempotency_key"`
}

type submitResponse struct {
	JobID   interface{} `json:"job_id"`
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
}

func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func (c *CoreProvider) Enqueue(ctx context.Context, jobType string, payload []byte) (string, error) {
	// Parse raw payload to make sure it's valid JSON
	var rawPayload json.RawMessage = payload
	if !json.Valid(payload) {
		// If payload is not valid JSON, wrap it as a JSON string to avoid schema validation failure
		escaped, err := json.Marshal(string(payload))
		if err != nil {
			return "", fmt.Errorf("failed to escape payload: %w", err)
		}
		rawPayload = escaped
	}

	reqBody := submitRequest{
		JobType:        jobType,
		Payload:        rawPayload,
		IdempotencyKey: generateUUID(),
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/jobs/submit", c.coreURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.jwtToken))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned status code %d: %s", resp.StatusCode, string(respBytes))
	}

	var submitResp submitResponse
	if err := json.Unmarshal(respBytes, &submitResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	jobIDStr := fmt.Sprintf("%v", submitResp.JobID)
	return jobIDStr, nil
}

func (c *CoreProvider) Dequeue(ctx context.Context) (*Job, error) {
	return nil, fmt.Errorf("dequeue not supported in CoreProvider")
}

func (c *CoreProvider) Ack(ctx context.Context, jobID string) error {
	return fmt.Errorf("ack not supported in CoreProvider")
}

func (c *CoreProvider) Nack(ctx context.Context, jobID string, err error) error {
	return fmt.Errorf("nack not supported in CoreProvider")
}
