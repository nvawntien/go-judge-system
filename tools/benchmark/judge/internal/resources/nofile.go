// Package resources checks local client capacity without changing host limits.
package resources

import (
	"fmt"

	"golang.org/x/sys/unix"
)

const noFileReserve uint64 = 256

type NoFileStatus struct {
	SoftLimit uint64
	Required  uint64
}

// EstimateNoFile reserves one possible POST connection and one SSE connection
// per logical in-flight submission, plus a fixed allowance for the process,
// DNS, collector files, and unrelated descriptors.
func EstimateNoFile(maxInFlight int) (uint64, error) {
	if maxInFlight < 1 {
		return 0, fmt.Errorf("max in-flight must be positive")
	}
	value := uint64(maxInFlight)
	if value > (^(uint64(0))-noFileReserve)/2 {
		return 0, fmt.Errorf("max in-flight exceeds local descriptor estimate capacity")
	}
	return value*2 + noFileReserve, nil
}

// CheckNoFile fails closed rather than attempting to mutate the operator's
// process limits. The caller should surface the required `ulimit -n` value.
func CheckNoFile(maxInFlight int) (NoFileStatus, error) {
	required, err := EstimateNoFile(maxInFlight)
	if err != nil {
		return NoFileStatus{}, err
	}
	var limit unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &limit); err != nil {
		return NoFileStatus{}, fmt.Errorf("read RLIMIT_NOFILE")
	}
	return CheckNoFileLimit(limit.Cur, required)
}

// CheckNoFileLimit is pure so the preflight threshold can be tested without
// mutating or depending on the local process limit.
func CheckNoFileLimit(softLimit, required uint64) (NoFileStatus, error) {
	status := NoFileStatus{SoftLimit: softLimit, Required: required}
	if status.SoftLimit < status.Required {
		return status, fmt.Errorf("RLIMIT_NOFILE soft limit %d is below required %d; run `ulimit -n %d` before the benchmark", status.SoftLimit, status.Required, status.Required)
	}
	return status, nil
}
