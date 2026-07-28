package execute

import "testing"

func TestParseGoCompileDiagnostics(t *testing.T) {
	output := "./main.go:19:9: make (built-in) must be called\nmain.go:20:2: undefined: value"

	got := parseCompileDiagnostics("GO", output)
	if len(got) != 2 {
		t.Fatalf("diagnostics length = %d, want 2: %+v", len(got), got)
	}
	if got[0].Kind != "compile" || got[0].Severity != "error" || got[0].Line != 19 || got[0].Column != 9 ||
		got[0].Message != "make (built-in) must be called" {
		t.Fatalf("first diagnostic = %+v", got[0])
	}
	if got[1].Line != 20 || got[1].Column != 2 || got[1].Message != "undefined: value" {
		t.Fatalf("second diagnostic = %+v", got[1])
	}
}

func TestParseRuntimeDiagnosticsUsesUserSourceFrameAndSanitizesPath(t *testing.T) {
	output := "panic: runtime error: index out of range [1] with length 0\n\ngoroutine 1 [running]:\nmain.main()\n\t/tmp/judge/run/main.go:24 +0x39\nruntime.main()\n\t/usr/local/go/src/runtime/proc.go:285 +0x29d"

	got := parseRuntimeDiagnostics("GO", "custom-1", output)
	if len(got) != 1 {
		t.Fatalf("diagnostics length = %d, want 1: %+v", len(got), got)
	}
	if got[0].Kind != "runtime" || got[0].Line != 24 || got[0].Column != 1 ||
		got[0].Message != "panic: runtime error: index out of range [1] with length 0" ||
		got[0].TestCaseID == nil || *got[0].TestCaseID != "custom-1" {
		t.Fatalf("diagnostic = %+v", got[0])
	}
}

func TestDiagnosticsMalformedOutputDoesNotPanic(t *testing.T) {
	if got := parseCompileDiagnostics("GO", "not a compiler location"); len(got) != 0 {
		t.Fatalf("compile diagnostics = %+v, want empty", got)
	}
	if got := parseRuntimeDiagnostics("GO", "case-1", "panic without frame"); len(got) != 0 {
		t.Fatalf("runtime diagnostics = %+v, want empty", got)
	}
}

func TestSanitizeOutputRemovesInternalPathPrefixes(t *testing.T) {
	got := sanitizeOutput("/tmp/judge/abc/main.go:6:2: undefined: x\n/w/sandbox/main.cpp:7:3: error: boom")
	want := "main.go:6:2: undefined: x\nmain.cpp:7:3: error: boom"
	if got != want {
		t.Fatalf("sanitizeOutput() = %q, want %q", got, want)
	}
}
