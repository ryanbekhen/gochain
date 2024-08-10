package gochain

type Function struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  interface{}     `json:"parameters"`
	Function    FunctionHandler `json:"-"`
}

type FunctionResponse struct {
	Tool      string                 `json:"tool"`
	ToolInput map[string]interface{} `json:"toolInput"`
}

type ConversationalFunctionResponse struct {
	Response string `json:"response"`
}

type FunctionHandler func(params interface{}) error
type ConversationalFunctionHandler func(response string)
