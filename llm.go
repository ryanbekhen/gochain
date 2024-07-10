package gochain

import "context"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LLM interface {
	Name() string
	Chat(ctx context.Context, messages []Message, options ...map[string]interface{}) (string, error)
}
