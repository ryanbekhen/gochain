package cfworkerai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/ryanbekhen/gochain"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type CFWorkerAI struct {
	base      *url.URL
	model     string
	accountId string
	http      *http.Client
}

type tokenTransport struct {
	Transport http.RoundTripper
	Token     string
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", "Bearer "+t.Token)
	return t.Transport.RoundTrip(req)
}

func New(accountID, token, model string, httpClient *http.Client) (*CFWorkerAI, error) {
	if model == "" {
		model = "@cf/meta/llama-3.1-8b-instruct"
	}

	if accountID == "" || token == "" {
		return nil, errors.New("CF_WORKER_ACCOUNT_ID or CF_WORKER_AI_TOKEN environment variables are not set")
	}

	base, err := url.Parse("https://api.cloudflare.com/client/v4/accounts")
	if err != nil {
		return nil, err
	}

	return &CFWorkerAI{
		base:      base,
		model:     model,
		accountId: accountID,
		http:      httpClient,
	}, nil
}

func NewFromEnvironment() (*CFWorkerAI, error) {
	baseURL := "https://api.cloudflare.com/client/v4/accounts"
	accountID := os.Getenv("CF_WORKER_AI_ACCOUNT_ID")
	token := os.Getenv("CF_WORKER_AI_TOKEN")
	model := os.Getenv("CF_WORKER_AI_MODEL")

	if model == "" {
		model = "@cf/meta/llama-3.1-8b-instruct"
	}

	if accountID == "" || token == "" {
		return nil, errors.New("CF_WORKER_ACCOUNT_ID or CF_WORKER_AI_TOKEN environment variables are not set")
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Transport: &tokenTransport{
			Transport: http.DefaultTransport,
			Token:     token,
		},
	}

	return &CFWorkerAI{
		base:      base,
		http:      client,
		accountId: accountID,
		model:     model,
	}, nil
}

func (c *CFWorkerAI) Name() string {
	return "cf-worker-ai"
}

func (c *CFWorkerAI) Model() string {
	return c.model
}

func (c *CFWorkerAI) SetModel(model string) {
	c.model = model
}

func (c *CFWorkerAI) Embedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	reqJSON, _ := json.Marshal(map[string]interface{}{
		"text": req.Prompt,
	})

	baseURL := c.base.String() + "/" + c.accountId + "/ai/run/" + req.Model
	resp, err := c.http.Post(baseURL, "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *CFWorkerAI) Chat(ctx context.Context, messages []gochain.Message, options ...map[string]interface{}) (string, error) {
	reqJSON, _ := json.Marshal(map[string]interface{}{
		"messages": messages,
	})

	baseURL := c.base.String() + "/" + c.accountId + "/ai/run/" + c.model
	resp, err := c.http.Post(baseURL, "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("error with status code: " + resp.Status)
	}

	var response ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if !response.Success {
		return "", errors.New("chat failed: " + strings.Join(response.Error, ", "))
	}

	return response.Result.Response, nil
}
