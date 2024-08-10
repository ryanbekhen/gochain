package gochain

import (
	"context"
	"encoding/json"
	"github.com/ryanbekhen/gochain/internal/prompt"
	"sort"
	"strings"
)

type Chain struct {
	llm         LLM
	fnPrompt    string
	fn          []*Function
	convHandler ConversationalFunctionHandler
}

func New(llm LLM) *Chain {
	return &Chain{
		llm: llm,
		fn: []*Function{
			{
				Name:        "conversationalResponse",
				Description: "Respond conversationally if no other tools should be called for a given query.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"response": map[string]interface{}{
							"type":        "string",
							"description": "Conversational response to the user with using same language.",
						},
					},
					"required": []string{"response"},
				},
			},
		},
	}
}

func (a *Chain) RegisterFunction(name string, description string, parameters interface{}, fn FunctionHandler) {
	sort.Slice(a.fn, func(i, j int) bool {
		n1 := a.fn[i].Name
		n2 := a.fn[j].Name
		return n1 != "conversationalResponse" && (n2 == "conversationalResponse" || n1 < n2)
	})

	a.fn = append(a.fn, &Function{
		Name:        name,
		Description: description,
		Parameters:  parameters,
		Function:    fn,
	})
}

func (a *Chain) RegisterConversationalFunction(h ConversationalFunctionHandler) {
	a.convHandler = h
}

func (a *Chain) Invoke(ctx context.Context, message string) error {
	functions, err := json.Marshal(a.fn)
	if err != nil {
		return err
	}

	promptContent := strings.Replace(prompt.FunctionsToCall, "{functionsToCall}", string(functions), -1)

	opts := map[string]interface{}{}

	if a.llm.Name() == "ollama" {
		opts["format"] = "json"
	}

	response, err := a.llm.Chat(ctx, []Message{
		{Role: "system", Content: promptContent},
		{Role: "user", Content: message},
	}, opts)
	if err != nil {
		return err
	}

	fr, err := a.parseResponse(response)
	if err != nil {
		return ErrInvalidResponse
	}

	if fr.Tool == "conversationalResponse" {
		resp, ok := fr.ToolInput["response"].(string)
		if !ok {
			return ErrInvalidResponse
		}

		if a.convHandler == nil {
			return ErrConversationalHandlerNotSet
		}

		a.convHandler(resp)

		return nil
	}

	handler, err := a.getHandler(fr.Tool)
	if err != nil {
		return err
	}

	return handler(fr.ToolInput)
}

func (a *Chain) getHandler(tool string) (FunctionHandler, error) {
	for _, f := range a.fn {
		if f.Name == tool {
			return f.Function, nil
		}
	}

	return nil, ErrFunctionNotFound
}

func (a *Chain) parseResponse(response string) (*FunctionResponse, error) {
	var fr FunctionResponse
	if err := json.Unmarshal([]byte(response), &fr); err != nil {
		return nil, err
	}

	return &fr, nil
}
