package domain

import "errors"

var (
	ErrJobNotFound       = errors.New("job not found")
	ErrInvalidLanguage   = errors.New("invalid programming language")
	ErrExecutionTimeout  = errors.New("execution timeout")
	ErrGoJudgeConnection = errors.New("cannot connect to go-judge")
	ErrInvalidJobMessage = errors.New("invalid job message format")
)

type nonRetryableError struct {
	err error
}

func (e nonRetryableError) Error() string {
	return e.err.Error()
}

func (e nonRetryableError) Unwrap() error {
	return e.err
}

func MarkNonRetryable(err error) error {
	if err == nil {
		return nil
	}
	if IsNonRetryable(err) {
		return err
	}
	return nonRetryableError{err: err}
}

func IsNonRetryable(err error) bool {
	var target nonRetryableError
	return errors.As(err, &target)
}
