// Package external validates optional UTC correlation samples. These metrics
// are supplemental: client timestamps remain the only latency clock.
package external

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"time"
)

type Sample struct {
	Timestamp time.Time
	Values    map[string]string
}

// ReadCSV accepts collector CSV files with a RFC3339Nano `timestamp` column.
// It is deliberately schema-tolerant so container and Kafka collectors can
// evolve without changing benchmark semantics.
func ReadCSV(path string) ([]Sample, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}
	timestampIndex := -1
	for index, name := range header {
		if name == "timestamp" {
			timestampIndex = index
		}
	}
	if timestampIndex < 0 {
		return nil, fmt.Errorf("CSV has no timestamp column")
	}
	var samples []Sample
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read CSV row: %w", err)
		}
		if len(row) != len(header) {
			return nil, fmt.Errorf("CSV row has %d fields, want %d", len(row), len(header))
		}
		at, err := time.Parse(time.RFC3339Nano, row[timestampIndex])
		if err != nil {
			return nil, fmt.Errorf("parse UTC timestamp: %w", err)
		}
		values := make(map[string]string, len(header))
		for index, name := range header {
			values[name] = row[index]
		}
		samples = append(samples, Sample{Timestamp: at.UTC(), Values: values})
	}
	return samples, nil
}

// CountInRange aligns wall-clock auxiliary samples to the run's UTC bounds.
// It never uses cross-host wall clocks to calculate client latency.
func CountInRange(samples []Sample, start, end time.Time) int {
	count := 0
	for _, sample := range samples {
		if !sample.Timestamp.Before(start) && (end.IsZero() || !sample.Timestamp.After(end)) {
			count++
		}
	}
	return count
}
