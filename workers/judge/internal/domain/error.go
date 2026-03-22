package domain

import "errors"

var (
	ErrJobNotFound       = errors.New("job not found")
	ErrInvalidLanguage   = errors.New("invalid programming language")
	ErrExecutionTimeout  = errors.New("execution timeout")
	ErrGoJudgeConnection = errors.New("cannot connect to go-judge")
	ErrInvalidJobMessage = errors.New("invalid job message format")
)
