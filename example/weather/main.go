package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ryanbekhen/gochain"
	"github.com/ryanbekhen/gochain/llm/ollama"
	"golang.org/x/net/html"
	"io"
	"net/http"
)

func ExtractText(n *html.Node, buf *bytes.Buffer) {
	// Remove style tags
	if n.Type == html.ElementNode && n.Data == "style" {
		return
	}
	if n.Type == html.TextNode {
		buf.WriteString(n.Data)
	}
	// Remove style attribute
	if n.Type == html.ElementNode {
		var i int
		for _, attr := range n.Attr {
			if attr.Key == "style" {
				continue
			}
			n.Attr[i] = attr
			i++
		}
		n.Attr = n.Attr[:i]
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		ExtractText(c, buf)
	}
}

func CleanHTML(htmlString string) (string, error) {
	doc, err := html.Parse(bytes.NewBufferString(htmlString))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	ExtractText(doc, &buf)
	return buf.String(), nil
}

func getWeather(location string, unit ...string) (string, error) {
	urlApi := fmt.Sprintf("https://wttr.in/%s?0", location)
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
	engine, err := ollama.NewFromEnvironment()
	if err != nil {
		panic(err)
	}

	message := "What is the weather in Jakarta?"

	engine.SetModel("llama3")
	chain := gochain.New(engine)
	chain.RegisterFunction("getCurrentWeather", "Get the current weather in a given location", map[string]interface{}{
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

		weather, err = CleanHTML(weather)
		if err != nil {
			return err
		}

		engine2, err := ollama.NewFromEnvironment()
		if err != nil {
			return err
		}

		prompt := "Using this data: " + weather + ". Respond to this prompt: " + message
		_, err = engine2.Embedding(context.Background(), &ollama.EmbeddingRequest{
			Model:  "mxbai-embed-large",
			Prompt: prompt,
		})
		if err != nil {
			return err
		}

		stream := true
		_ = engine2.SendChat(context.Background(), &ollama.ChatRequest{
			Model: "llama3",
			Messages: []gochain.Message{
				{
					Role:    "system",
					Content: "You are assistant for customer service weather, you must response with user's language only.",
				},
				{
					Role:    "user",
					Content: message,
				},
			},
			Stream: &stream,
		}, func(resp ollama.ChatResponse) error {
			fmt.Print(resp.Message.Content)
			return nil
		})

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
