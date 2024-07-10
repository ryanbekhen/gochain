package ollama

import (
	"context"
	"github.com/ollama/ollama/api"
	"github.com/ryanbekhen/gochain"
	"net/http"
	"net/url"
)

type Ollama struct {
	api   *api.Client
	model string
}

func NewFromEnvironment() (*Ollama, error) {
	e, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, err
	}

	return &Ollama{api: e}, nil
}

func New(baseUrl string, httpClient *http.Client) (*Ollama, error) {
	uri, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}

	return &Ollama{api: api.NewClient(uri, httpClient)}, nil
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
	msgs := make([]api.Message, len(messages))
	for i, m := range messages {
		msgs[i] = api.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	var opts map[string]interface{}
	if len(options) > 0 {
		opts = options[0]
	}

	var format string
	if f, ok := opts["format"].(string); ok {
		format = f
	}

	req := &api.ChatRequest{
		Model:    o.model,
		Messages: msgs,
		Format:   format,
		Options: map[string]interface{}{
			"keep_alive": -1,
		},
	}

	var chatResponse string
	if err := o.api.Chat(ctx, req, func(resp api.ChatResponse) error {
		chatResponse += resp.Message.Content
		return nil
	}); err != nil {
		return "", err
	}

	return chatResponse, nil
}
