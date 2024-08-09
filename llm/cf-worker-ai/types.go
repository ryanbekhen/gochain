package cfworkerai

type ChatResponse struct {
	Result  ChatResponseResult `json:"result"`
	Success bool               `json:"success"`
	Error   []string           `json:"error"`
	Message []string           `json:"message"`
}

type ChatResponseResult struct {
	Response string `json:"response"`
}
