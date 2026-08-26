package external

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadCSVRequiresUTCTimestampColumn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "samples.csv")
	if err := os.WriteFile(path, []byte("timestamp,value\n2026-08-25T12:00:00.000000001Z,1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	samples, err := ReadCSV(path)
	if err != nil || len(samples) != 1 || samples[0].Timestamp.Location().String() != "UTC" {
		t.Fatalf("samples=%+v err=%v", samples, err)
	}
}

func TestCountInRangeUsesWallClockOnlyForCorrelation(t *testing.T) {
	start := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	samples := []Sample{{Timestamp: start.Add(-time.Nanosecond)}, {Timestamp: start}, {Timestamp: start.Add(time.Second)}}
	if got := CountInRange(samples, start, start); got != 1 {
		t.Fatalf("count=%d", got)
	}
}
