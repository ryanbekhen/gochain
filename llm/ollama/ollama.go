package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ryanbekhen/gochain"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Ollama struct {
	base  *url.URL
	model string
	http  *http.Client
}

type ChatRequest struct {
	Model     string                 `json:"model"`
	Messages  []gochain.Message      `json:"messages"`
	Stream    *bool                  `json:"stream,omitempty"`
	Format    string                 `json:"format"`
	KeepAlive time.Duration          `json:"keep_alive,omitempty"`
	Options   map[string]interface{} `json:"options"`
}

type ChatResponse struct {
	Model      string          `json:"model"`
	CreatedAt  time.Time       `json:"created_at"`
	Message    gochain.Message `json:"message"`
	DoneReason string          `json:"done_reason,omitempty"`

	Done bool `json:"done"`

	Metrics
}

type StatusError struct {
	StatusCode   int
	Status       string
	ErrorMessage string `json:"error"`
}

type Metrics struct {
	TotalDuration      time.Duration `json:"total_duration,omitempty"`
	LoadDuration       time.Duration `json:"load_duration,omitempty"`
	PromptEvalCount    int           `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration time.Duration `json:"prompt_eval_duration,omitempty"`
	EvalCount          int           `json:"eval_count,omitempty"`
	EvalDuration       time.Duration `json:"eval_duration,omitempty"`
}

func (e StatusError) Error() string {
	switch {
	case e.Status != "" && e.ErrorMessage != "":
		return fmt.Sprintf("%s: %s", e.Status, e.ErrorMessage)
	case e.Status != "":
		return e.Status
	case e.ErrorMessage != "":
		return e.ErrorMessage
	default:
		// this should not happen
		return "something went wrong, please see the ollama server logs for details"
	}
}

func NewFromEnvironment() (*Ollama, error) {
	ollamaEndpoint := "http://localhost:11434"

	if e := os.Getenv("OLLAMA_HOST"); e != "" {
		ollamaEndpoint = e
	}

	base, err := url.Parse(ollamaEndpoint)
	if err != nil {
		return nil, err
	}

	return &Ollama{
		base: base,
		http: http.DefaultClient,
	}, nil
}

func New(baseUrl string, httpClient *http.Client) (*Ollama, error) {
	base, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}

	return &Ollama{base: base, http: httpClient}, nil
}

func (o *Ollama) Name() string {
	return "ollama"
}

func (o *Ollama) SetModel(model string) {
	o.model = model
}

func (o *Ollama) Model() string {
	return o.model
}

func (o *Ollama) Chat(ctx context.Context, messages []gochain.Message, options ...map[string]interface{}) (string, error) {
	var opts map[string]interface{}
	if len(options) > 0 {
		opts = options[0]
	}

	var formatResponse string
	if f, ok := opts["format"].(string); ok {
		formatResponse = f
		delete(opts, "format")
	}

	var keepAlive time.Duration
	if k, ok := opts["keep_alive"].(time.Duration); ok {
		keepAlive = k
		delete(opts, "keep_alive")
	}

	stream := true
	req := &ChatRequest{
		Model:     o.model,
		Messages:  messages,
		Format:    formatResponse,
		KeepAlive: keepAlive,
		Stream:    &stream,
		Options:   opts,
	}

	var chatResponse string
	if err := o.chat(ctx, req, func(resp ChatResponse) error {
		chatResponse += resp.Message.Content
		return nil
	}); err != nil {
		return "", err
	}

	return chatResponse, nil
}

// maxBufferSize is the maximum buffer size for the scanner (512 KB)
const maxBufferSize = 512 * 1024

func (o *Ollama) stream(ctx context.Context, method, path string, data any, fn func([]byte) error) error {
	var buf *bytes.Buffer
	if data != nil {
		bts, err := json.Marshal(data)
		if err != nil {
			return err
		}

		buf = bytes.NewBuffer(bts)
	}

	requestURL := o.base.JoinPath(path)
	request, err := http.NewRequestWithContext(ctx, method, requestURL.String(), buf)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/x-ndjson")
	request.Header.Set("User-Agent", "ollama")

	response, err := o.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	scanner := bufio.NewScanner(response.Body)
	scanBuf := make([]byte, 0, maxBufferSize)
	scanner.Buffer(scanBuf, maxBufferSize)
	for scanner.Scan() {
		var errorResponse struct {
			Error string `json:"error,omitempty"`
		}

		bts := scanner.Bytes()
		if err := json.Unmarshal(bts, &errorResponse); err != nil {
			return fmt.Errorf("unmarshal: %w", err)
		}

		if errorResponse.Error != "" {
			return fmt.Errorf(errorResponse.Error)
		}

		if response.StatusCode >= http.StatusBadRequest {
			return StatusError{
				StatusCode:   response.StatusCode,
				Status:       response.Status,
				ErrorMessage: errorResponse.Error,
			}
		}

		if err := fn(bts); err != nil {
			return err
		}
	}

	return nil
}

func (o *Ollama) chat(ctx context.Context, req *ChatRequest, fn func(ChatResponse) error) error {
	return o.stream(ctx, http.MethodPost, "/api/chat", req, func(bts []byte) error {
		var resp ChatResponse
		if err := json.Unmarshal(bts, &resp); err != nil {
			return err
		}

		return fn(resp)
	})
}
