package result

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/model"
)

func TestCreateIsImmutableAndRestrictive(t *testing.T) {
	root := filepath.Join(t.TempDir(), "results")
	writer, err := Create(root, "B1-R1-20260101T000000Z")
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteSummary(model.RunSummary{RunID: "x"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(writer.Dir, "summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode=%o", info.Mode().Perm())
	}
	if _, err := Create(root, "B1-R1-20260101T000000Z"); err == nil {
		t.Fatal("existing run directory overwritten")
	}
}

func TestRedactorKeepsSentinelsOutOfOutput(t *testing.T) {
	redactor := NewRedactor("access-secret", "refresh-secret", "ticket-secret", "source-secret", "real-user-id")
	value := redactor.Sanitize("access-secret refresh-secret ticket-secret source-secret real-user-id")
	for _, secret := range []string{"access-secret", "refresh-secret", "ticket-secret", "source-secret", "real-user-id"} {
		if strings.Contains(value, secret) {
			t.Fatalf("secret leaked: %s", secret)
		}
	}
}

func TestCreateRejectsTraversalControlAndCSVFormulaFields(t *testing.T) {
	root := t.TempDir()
	for _, runID := range []string{"../escape", "a/b", `a\\b`, "\x01bad"} {
		if _, err := Create(root, runID); err == nil {
			t.Fatalf("unsafe run ID accepted: %q", runID)
		}
	}
	writer, err := Create(root, "safe-run")
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteSubmissions([]model.SubmissionRecord{{RunID: "safe-run", UserAlias: "=formula", Error: "+unsafe"}}); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteWindows([]model.WindowRecord{{RunID: "@unsafe"}}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(writer.Dir, "submissions.csv"))
	if err != nil || !strings.Contains(string(data), "'=formula") || !strings.Contains(string(data), "'+unsafe") {
		t.Fatalf("CSV formula mitigation missing: %q err=%v", data, err)
	}
	windows, err := os.ReadFile(filepath.Join(writer.Dir, "windows.csv"))
	if err != nil || !strings.Contains(string(windows), "'@unsafe") {
		t.Fatalf("window CSV formula mitigation missing: %q err=%v", windows, err)
	}
}
