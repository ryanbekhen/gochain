package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ryanbekhen/gochain"
	"io"
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

func checkError(resp *http.Response, body []byte) error {
	if resp.StatusCode < http.StatusBadRequest {
		return nil
	}

	apiError := StatusError{StatusCode: resp.StatusCode}

	err := json.Unmarshal(body, &apiError)
	if err != nil {
		// Use the full body as the message if we fail to decode a response.
		apiError.ErrorMessage = string(body)
	}

	return apiError
}

func (o *Ollama) do(ctx context.Context, method, path string, reqData, respData any) error {
	var reqBody io.Reader
	var data []byte
	var err error

	switch reqData := reqData.(type) {
	case io.Reader:
		// reqData is already an io.Reader
		reqBody = reqData
	case nil:
		// noop
	default:
		data, err = json.Marshal(reqData)
		if err != nil {
			return err
		}

		reqBody = bytes.NewReader(data)
	}

	requestURL := o.base.JoinPath(path)
	request, err := http.NewRequestWithContext(ctx, method, requestURL.String(), reqBody)
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", fmt.Sprintf("gochain"))

	respObj, err := o.http.Do(request)
	if err != nil {
		return err
	}
	defer respObj.Body.Close()

	respBody, err := io.ReadAll(respObj.Body)
	if err != nil {
		return err
	}

	if err := checkError(respObj, respBody); err != nil {
		return err
	}

	if len(respBody) > 0 && respData != nil {
		if err := json.Unmarshal(respBody, respData); err != nil {
			return err
		}
	}
	return nil

}

func (o *Ollama) Embedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	var resp EmbeddingResponse
	if err := o.do(ctx, http.MethodPost, "/api/embeddings", req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
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
	if err := o.SendChat(ctx, req, func(resp ChatResponse) error {
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
	request.Header.Set("User-Agent", "gochain")

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

func (o *Ollama) SendChat(ctx context.Context, req *ChatRequest, fn func(ChatResponse) error) error {
	return o.stream(ctx, http.MethodPost, "/api/chat", req, func(bts []byte) error {
		var resp ChatResponse
		if err := json.Unmarshal(bts, &resp); err != nil {
			return err
		}

		return fn(resp)
	})
}
