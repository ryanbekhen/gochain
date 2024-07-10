package gochain

import "github.com/pkg/errors"

var (
	ErrFunctionNotFound            = errors.New("function not found")
	ErrInvalidResponse             = errors.New("invalid response")
	ErrChatFailed                  = errors.New("internal chat failed")
	ErrConversationalHandlerNotSet = errors.New("conversational handler not set")
)
