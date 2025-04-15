package shair

import (
	"errors"
	"fmt"
)

var (
	StatFileError          = errors.New("Received an invalid file path")
	SendFileError          = errors.New("Failed to send file")
	ConnectionDroppedError = errors.New("Tcp connexion dropped")
	TransferRejected       = errors.New("Target rejected the file transfer")
	UnexpectedError        = errors.New("Something unexpected happened")
)

type Error struct {
	Code          error
	Message       string
	UnderlyingErr error
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.UnderlyingErr)
}

func (e Error) Unwrap() error {
	return e.Code
}

func NewError(code error, msg string, err error) error {
	return Error{
		Code:          code,
		Message:       msg,
		UnderlyingErr: err,
	}
}
