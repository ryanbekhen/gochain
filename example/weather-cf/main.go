package main

import (
	"context"
	"fmt"
	"github.com/ryanbekhen/gochain"
	cfworkerai "github.com/ryanbekhen/gochain/llm/cf-worker-ai"
	"io"
	"net/http"
	"strings"
)

func getWeather(location string, unit ...string) (string, error) {
	urlApi := fmt.Sprintf("https://wttr.in/%s", location)
	resp, err := http.Get(urlApi)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func main() {
	engine, err := cfworkerai.NewFromEnvironment()
	if err != nil {
		panic(err)
	}

	message := "What is the weather in Jakarta?"

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
		p, ok := params.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid parameters")
		}

		location, ok := p["location"].(string)
		if !ok {
			return fmt.Errorf("invalid location")
		}

		unit, ok := p["unit"].(string)
		if !ok {
			unit = "celsius"
		}

		weather, err := getWeather(location, unit)
		if err != nil {
			return err
		}

		systemPrompt := `
		You are assistant for customer service weather, you must response with user's language only.
		Response from API:
		{response}
		`
		systemPrompt = strings.Replace("{message}", systemPrompt, weather, -1)

		response, err := engine.Chat(context.Background(), []gochain.Message{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: message,
			},
		})
		if err != nil {
			return err
		}
		fmt.Println(response)

		return nil
	})

	chain.RegisterConversationalFunction(func(response string) {
		fmt.Println("test conversational function called")
		fmt.Println(response)
	})

	if err := chain.Invoke(context.Background(), message); err != nil {
		fmt.Println(err)
	}
}
