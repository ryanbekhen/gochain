package gochain

import "context"

type Message struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type LLM interface {
	Name() string
	Chat(ctx context.Context, messages []Message, options ...map[string]interface{}) (string, error)
}
