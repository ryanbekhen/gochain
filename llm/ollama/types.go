package ollama

import (
	"fmt"
	"github.com/ryanbekhen/gochain"
	"time"
)

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

type EmbeddingRequest struct {
	Model     string                 `json:"model"`
	Prompt    string                 `json:"prompt"`
	KeepAlive time.Duration          `json:"keep_alive,omitempty"`
	Options   map[string]interface{} `json:"options"`
}

type EmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
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
