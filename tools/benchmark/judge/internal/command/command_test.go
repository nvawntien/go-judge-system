package command

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/model"
)

func TestVersionDoesNotNeedTargetConfiguration(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := Execute(context.Background(), []string{"version"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout.String()) != model.BenchmarkVersion {
		t.Fatalf("version output=%q", stdout.String())
	}
}
