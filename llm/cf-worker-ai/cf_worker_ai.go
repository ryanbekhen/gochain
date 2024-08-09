package cfworkerai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ryanbekhen/gochain"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type CFWorkerAI struct {
	base  *url.URL
	model string
	http  *http.Client
}

type tokenTransport struct {
	Transport http.RoundTripper
	Token     string
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", "Bearer "+t.Token)
	return t.Transport.RoundTrip(req)
}

func NewFromEnvironment() (*CFWorkerAI, error) {
	baseURL := "https://api.cloudflare.com/client/v4/accounts"
	accountID := os.Getenv("CF_WORKER_AI_ACCOUNT_ID")
	token := os.Getenv("CF_WORKER_AI_TOKEN")
	model := os.Getenv("CF_WORKER_AI_MODEL")

	if model == "" {
		model = "@cf/meta/llama-3-8b-instruct"
	}

	if accountID == "" || token == "" {
		return nil, errors.New("CF_WORKER_ACCOUNT_ID or CF_WORKER_AI_TOKEN environment variables are not set")
	}

	baseURL += "/" + accountID + "/ai/run/" + model

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
		base:  base,
		http:  client,
		model: model,
	}, nil
}

func (c *CFWorkerAI) Name() string {
	return "cf-worker-ai"
}

func (c *CFWorkerAI) Model() string {
	return c.model
}

func (c *CFWorkerAI) Chat(ctx context.Context, messages []gochain.Message, options ...map[string]interface{}) (string, error) {
	reqJSON, _ := json.Marshal(map[string]interface{}{
		"messages": messages,
	})
	resp, err := c.http.Post(c.base.String(), "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("error with status code: " + resp.Status)
	}

	var body *ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}

	if !body.Success {
		fmt.Println(resp.Body)
		return "", errors.New("chat failed: " + strings.Join(body.Error, ", "))
	}
	return body.Result.Response, nil
}
