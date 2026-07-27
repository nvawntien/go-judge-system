package dto

import "time"

type RunCodeLimits struct {
	MaxTestCases           int
	MaxSourceCodeBytes     int
	MaxStdinBytes          int
	MaxExpectedOutputBytes int
	MaxCapturedOutputBytes int
	RequestTimeout         time.Duration
	DefaultTimeLimit       time.Duration
	DefaultMemoryLimitKB   int64
	DefaultOutputLimit     int64
}
