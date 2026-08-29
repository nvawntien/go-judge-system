// Package resources checks local client capacity without changing host limits.
package resources

import (
	"fmt"

	"golang.org/x/sys/unix"
)

const noFileReserve uint64 = 256

// Observation describes the client-side connection model being qualified. It
// is deliberately an estimate, not a claim about one TCP connection per user:
// HTTP/2 may multiplex requests and streams while HTTP/1.1 may not.
type Observation string

const (
	ObservationTerminal      Observation = "terminal"
	ObservationAdmissionOnly Observation = "admission-only"
	ObservationRealistic     Observation = "realistic"
)

type NoFileStatus struct {
	SoftLimit                       uint64
	HardLimit                       uint64
	Required                        uint64
	Recommended                     uint64
	ConnectionsPerLogicalSubmission uint64
}

// EstimateNoFile reserves one possible POST connection and one SSE connection
// per logical in-flight submission, plus a fixed allowance for the process,
// DNS, collector files, and unrelated descriptors.
func EstimateNoFile(maxInFlight int) (uint64, error) {
	return EstimateNoFileForObservation(maxInFlight, ObservationTerminal)
}

// EstimateNoFileFor uses one connection slot for a POST-only logical request
// and adds a second slot only when terminal SSE observation is enabled.
func EstimateNoFileFor(maxInFlight int, terminalObservation bool) (uint64, error) {
	mode := ObservationAdmissionOnly
	if terminalObservation {
		mode = ObservationTerminal
	}
	return EstimateNoFileForObservation(maxInFlight, mode)
}

func EstimateNoFileForObservation(maxInFlight int, mode Observation) (uint64, error) {
	if maxInFlight < 1 {
		return 0, fmt.Errorf("max in-flight must be positive")
	}
	value := uint64(maxInFlight)
	slots := uint64(1)
	switch mode {
	case ObservationTerminal, ObservationRealistic:
		slots = 2
	case ObservationAdmissionOnly:
	default:
		return 0, fmt.Errorf("unknown observation mode %q", mode)
	}
	if value > (^(uint64(0))-noFileReserve)/slots {
		return 0, fmt.Errorf("max in-flight exceeds local descriptor estimate capacity")
	}
	return value*slots + noFileReserve, nil
}

// CheckNoFile fails closed rather than attempting to mutate the operator's
// process limits. The caller should surface the required `ulimit -n` value.
func CheckNoFile(maxInFlight int) (NoFileStatus, error) {
	return CheckNoFileForObservation(maxInFlight, ObservationTerminal)
}

func CheckNoFileFor(maxInFlight int, terminalObservation bool) (NoFileStatus, error) {
	mode := ObservationAdmissionOnly
	if terminalObservation {
		mode = ObservationTerminal
	}
	return CheckNoFileForObservation(maxInFlight, mode)
}

func CheckNoFileForObservation(maxInFlight int, mode Observation) (NoFileStatus, error) {
	required, err := EstimateNoFileForObservation(maxInFlight, mode)
	if err != nil {
		return NoFileStatus{}, err
	}
	var limit unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &limit); err != nil {
		return NoFileStatus{}, fmt.Errorf("read RLIMIT_NOFILE")
	}
	status, err := CheckNoFileLimit(limit.Cur, required)
	status.HardLimit = limit.Max
	status.ConnectionsPerLogicalSubmission = 1
	if mode == ObservationTerminal || mode == ObservationRealistic {
		status.ConnectionsPerLogicalSubmission = 2
	}
	// 262144 is only a starting recommendation for documented 100k runs; this
	// code never mutates process limits and records observed FD use separately.
	status.Recommended = required
	if maxInFlight >= 100000 && status.Recommended < 262144 {
		status.Recommended = 262144
	}
	return status, err
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
