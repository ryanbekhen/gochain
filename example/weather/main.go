package main

import (
	"context"
	"fmt"
	"github.com/ryanbekhen/gochain"
	"github.com/ryanbekhen/gochain/llm/ollama"
)

func main() {
	engine, err := ollama.NewFromEnvironment()
	if err != nil {
		panic(err)
	}

	engine.SetModel("phi3")
	chain := gochain.New(engine)
	chain.RegisterFunction("get_current_weather", "Get the current weather in a given location", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"location": map[string]interface{}{
				"type":        "string",
				"description": "The city and state, e.g. San Francisco, CA",
			},
			"unit": map[string]interface{}{
				"type": "string",
				"enum": []string{"celsius", "fahrenheit"},
			},
		},
		"required": []string{"location"},
	}, func(params interface{}) error {
		fmt.Println("test function called")
		fmt.Println(params)
		return nil
	})

	chain.RegisterConversationalFunction(func(response string) {
		fmt.Println("test conversational function called")
		fmt.Println(response)
	})

	if err := chain.Invoke(context.Background(), "what is the weather in Singapore?"); err != nil {
		fmt.Println(err)
	}
}
