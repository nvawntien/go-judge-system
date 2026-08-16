package gojudge

import (
	"slices"
	"testing"
)

func TestJavaLanguageConfigUsesJarArtifact(t *testing.T) {
	sourceFile := GetSourceFileName("JAVA")
	exeFile := GetExeFileName("JAVA")

	if sourceFile != "Main.java" {
		t.Fatalf("unexpected Java source file: %q", sourceFile)
	}

	if exeFile != "main.jar" {
		t.Fatalf("unexpected Java executable artifact: %q", exeFile)
	}

	cfg, ok := GetLanguageConfig("JAVA", sourceFile, exeFile)
	if !ok {
		t.Fatal("JAVA language config not found")
	}

	if cfg.Compile == nil {
		t.Fatal("JAVA must have a compile command")
	}

	if len(cfg.Compile.Command) < 3 {
		t.Fatalf("unexpected Java compile command: %#v", cfg.Compile.Command)
	}

	compileScript := cfg.Compile.Command[len(cfg.Compile.Command)-1]

	requiredCompileFragments := []string{
		"javac",
		"-d classes",
		"Main.java",
		"main.jar",
		"-C classes .",
	}

	for _, fragment := range requiredCompileFragments {
		if !containsSubstring(compileScript, fragment) {
			t.Fatalf(
				"Java compile command %q does not contain %q",
				compileScript,
				fragment,
			)
		}
	}

	wantRun := []string{
		"/usr/bin/java",
		"-cp",
		"main.jar",
		"Main",
	}

	if !slices.Equal(cfg.Run.Command, wantRun) {
		t.Fatalf(
			"unexpected Java run command: got %#v want %#v",
			cfg.Run.Command,
			wantRun,
		)
	}
}

func containsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}

	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
