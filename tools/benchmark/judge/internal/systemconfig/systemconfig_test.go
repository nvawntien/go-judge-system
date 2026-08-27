package systemconfig

import (
	"os"
	"path/filepath"
	"testing"
)

const valid = `{"label":"test-config","release":"test-release","app":{"nodes":1,"cpu_cores_per_node":2,"memory_mib_per_node":1024},"judge":{"nodes":1,"cpu_cores_per_node":2,"memory_mib_per_node":1024,"worker_pool_size":1,"worker_memory_limit_mib":512,"sandbox_memory_limit_mib":1024}}`

func TestLoadStrictSafeSystemConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "system.json")
	if err := os.WriteFile(path, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	value, err := Load(path)
	if err != nil || value.Label != "test-config" || value.Judge.WorkerPoolSize != 1 {
		t.Fatalf("value=%+v err=%v", value, err)
	}
}

func TestLoadRejectsUnknownOrUnsafeSystemConfig(t *testing.T) {
	for _, content := range []string{
		`{"label":"x","release":"r","app":{"nodes":1,"cpu_cores_per_node":1,"memory_mib_per_node":1},"judge":{"nodes":1,"cpu_cores_per_node":1,"memory_mib_per_node":1,"worker_pool_size":1,"worker_memory_limit_mib":1,"sandbox_memory_limit_mib":1},"DATABASE_PASSWORD":"secret"}`,
		`{"label":"x","release":"r","app":{"nodes":0,"cpu_cores_per_node":1,"memory_mib_per_node":1},"judge":{"nodes":1,"cpu_cores_per_node":1,"memory_mib_per_node":1,"worker_pool_size":1,"worker_memory_limit_mib":1,"sandbox_memory_limit_mib":1}}`,
	} {
		path := filepath.Join(t.TempDir(), "system.json")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatal("expected invalid system config")
		}
	}
}

func TestLoadRejectsOversizedSystemConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "system.json")
	if err := os.WriteFile(path, append([]byte(valid), make([]byte, 16*1024)...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected oversized system config rejection")
	}
}
