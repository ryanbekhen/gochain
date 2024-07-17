package gochain

import "errors"

var (
	ErrFunctionNotFound            = errors.New("function not found")
	ErrInvalidResponse             = errors.New("invalid response")
	ErrConversationalHandlerNotSet = errors.New("conversational handler not set")
)
