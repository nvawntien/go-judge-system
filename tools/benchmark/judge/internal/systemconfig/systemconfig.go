// Package systemconfig loads explicitly allowlisted system-under-test metadata.
package systemconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/model"
)

const maxBytes = 16 * 1024

func Load(path string) (*model.SystemConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New("read system config")
	}
	defer file.Close()
	contents, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, errors.New("read system config")
	}
	if len(contents) > maxBytes {
		return nil, errors.New("system config exceeds size limit")
	}
	decoder := json.NewDecoder(strings.NewReader(string(contents)))
	decoder.DisallowUnknownFields()
	var value model.SystemConfig
	if err := decoder.Decode(&value); err != nil {
		return nil, errors.New("decode system config")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, errors.New("system config contains trailing JSON values")
	}
	if err := validate(value); err != nil {
		return nil, err
	}
	return &value, nil
}

func validate(value model.SystemConfig) error {
	if !safeText(value.Label) || !safeText(value.Release) {
		return errors.New("system config label and release are required")
	}
	for _, item := range []struct {
		name string
		v    int
	}{
		{"app.nodes", value.App.Nodes}, {"app.cpu_cores_per_node", value.App.CPUCoresPerNode}, {"app.memory_mib_per_node", value.App.MemoryMiBPerNode},
		{"judge.nodes", value.Judge.Nodes}, {"judge.cpu_cores_per_node", value.Judge.CPUCoresPerNode}, {"judge.memory_mib_per_node", value.Judge.MemoryMiBPerNode},
		{"judge.worker_pool_size", value.Judge.WorkerPoolSize}, {"judge.worker_memory_limit_mib", value.Judge.WorkerMemoryLimitMiB}, {"judge.sandbox_memory_limit_mib", value.Judge.SandboxMemoryLimitMiB},
	} {
		if item.v <= 0 {
			return fmt.Errorf("system config %s must be positive", item.name)
		}
	}
	return nil
}

func safeText(value string) bool {
	return strings.TrimSpace(value) == value && value != "" && len(value) <= 128 && !strings.ContainsAny(value, "\r\n\x00")
}
