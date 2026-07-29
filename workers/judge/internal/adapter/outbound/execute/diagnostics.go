package execute

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"go-judge-system/workers/judge/internal/application/port/outbound"
)

const (
	diagnosticKindCompile = "compile"
	diagnosticKindRuntime = "runtime"
	diagnosticSeverityErr = "error"
	maxRuntimeErrorBytes  = 4096
)

var (
	goCompileDiagnostic     = regexp.MustCompile(`(?m)(?:^|\n)\s*(?:\./)?(?:.*/)?main\.go:(\d+):(\d+):\s*(.+)`)
	cppCompileDiagnostic    = regexp.MustCompile(`(?m)(?:^|\n)\s*(?:\./)?(?:.*/)?main\.cpp:(\d+):(\d+):\s*(?:fatal\s+)?(?:error|warning):\s*(.+)`)
	javaCompileDiagnostic   = regexp.MustCompile(`(?m)(?:^|\n)\s*(?:\./)?(?:.*/)?Main\.java:(\d+):\s*(?:error|warning):\s*(.+)`)
	goRuntimeFrame          = regexp.MustCompile(`(?m)(?:^|\n)\s*(?:\./)?(?:.*/)?main\.go:(\d+)(?:\s|\+|$)`)
	cppRuntimeFrame         = regexp.MustCompile(`(?m)(?:^|\n).*?(?:\./)?(?:.*/)?main\.cpp:(\d+):?(\d+)?`)
	javaRuntimeFrame        = regexp.MustCompile(`(?m)\bat\s+Main\.[^(]*\(Main\.java:(\d+)\)`)
	pythonRuntimeFrame      = regexp.MustCompile(`(?m)File\s+"(?:.*/)?main\.py",\s+line\s+(\d+)`)
	internalPathWithSource  = regexp.MustCompile(`(?:/tmp/|/w/|/workspace/|/app/workspace/|/judge/)[^:\s"]*/(main\.go|main\.cpp|main\.py|Main\.java)`)
	remainingInternalPrefix = regexp.MustCompile(`(?:/tmp/|/w/|/workspace/|/app/workspace/|/judge/)+`)
	relativeSourcePath      = regexp.MustCompile(`\./(main\.go|main\.cpp|main\.py|Main\.java)`)
)

func parseCompileDiagnostics(language string, output string) []outbound.CodeDiagnostic {
	output = sanitizeOutput(output)
	switch strings.ToUpper(language) {
	case "GO":
		return parseLineColumnDiagnostics(output, goCompileDiagnostic, diagnosticKindCompile, 1, 2, 3)
	case "CPP":
		return parseLineColumnDiagnostics(output, cppCompileDiagnostic, diagnosticKindCompile, 1, 2, 3)
	case "JAVA":
		return parseLineColumnDiagnostics(output, javaCompileDiagnostic, diagnosticKindCompile, 1, 0, 2)
	default:
		return nil
	}
}

func parseRuntimeDiagnostics(language string, testcaseID string, output string) []outbound.CodeDiagnostic {
	output = sanitizeOutput(output)
	if strings.TrimSpace(output) == "" {
		return nil
	}

	line, column, ok := runtimeSourceLocation(language, output)
	if !ok {
		return nil
	}

	testID := testcaseID
	return []outbound.CodeDiagnostic{{
		TestCaseID: &testID,
		Kind:       diagnosticKindRuntime,
		Severity:   diagnosticSeverityErr,
		Message:    runtimeHeadline(output),
		Line:       line,
		Column:     column,
	}}
}

func parseLineColumnDiagnostics(output string, pattern *regexp.Regexp, kind string, lineGroup int, columnGroup int, messageGroup int) []outbound.CodeDiagnostic {
	matches := pattern.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return nil
	}

	diagnostics := make([]outbound.CodeDiagnostic, 0, len(matches))
	for _, match := range matches {
		line := parsePositiveInt(match[lineGroup], 0)
		if line <= 0 {
			continue
		}

		column := 1
		if columnGroup > 0 {
			column = parsePositiveInt(match[columnGroup], 1)
		}

		message := strings.TrimSpace(match[messageGroup])
		if message == "" {
			message = strings.TrimSpace(output)
		}
		diagnostics = append(diagnostics, outbound.CodeDiagnostic{
			Kind:     kind,
			Severity: diagnosticSeverityErr,
			Message:  sanitizeOutput(message),
			Line:     line,
			Column:   column,
		})
	}
	return diagnostics
}

func runtimeSourceLocation(language string, output string) (int, int, bool) {
	switch strings.ToUpper(language) {
	case "GO":
		return parseRuntimeFrame(goRuntimeFrame, output, 1, 0)
	case "CPP":
		return parseRuntimeFrame(cppRuntimeFrame, output, 1, 2)
	case "JAVA":
		return parseRuntimeFrame(javaRuntimeFrame, output, 1, 0)
	case "PYTHON":
		return parseRuntimeFrame(pythonRuntimeFrame, output, 1, 0)
	default:
		return 0, 0, false
	}
}

func parseRuntimeFrame(pattern *regexp.Regexp, output string, lineGroup int, columnGroup int) (int, int, bool) {
	match := pattern.FindStringSubmatch(output)
	if len(match) == 0 {
		return 0, 0, false
	}
	line := parsePositiveInt(match[lineGroup], 0)
	if line <= 0 {
		return 0, 0, false
	}
	column := 1
	if columnGroup > 0 && columnGroup < len(match) {
		column = parsePositiveInt(match[columnGroup], 1)
	}
	return line, column, true
}

func runtimeHeadline(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "goroutine ") ||
			strings.HasPrefix(line, "Traceback ") ||
			strings.HasPrefix(line, "File ") ||
			strings.HasPrefix(line, "at ") {
			continue
		}
		return line
	}
	return strings.TrimSpace(output)
}

func runtimeErrorMessage(stderr string, diagnostics []outbound.CodeDiagnostic) string {
	for _, diagnostic := range diagnostics {
		if diagnostic.Kind == diagnosticKindRuntime {
			if message := publicErrorMessage(diagnostic.Message); message != "" {
				return message
			}
		}
	}
	return publicErrorMessage(runtimeHeadline(sanitizeOutput(stderr)))
}

func publicErrorMessage(message string) string {
	message = sanitizeOutput(message)
	message = strings.ToValidUTF8(message, "")
	message = stripControlCharacters(message)
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	if len(message) <= maxRuntimeErrorBytes {
		return message
	}
	truncated := message[:maxRuntimeErrorBytes]
	for !utf8.ValidString(truncated) && len(truncated) > 0 {
		truncated = truncated[:len(truncated)-1]
	}
	return strings.TrimSpace(truncated) + "…"
}

func stripControlCharacters(value string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return r
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, value)
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func sanitizeOutput(output string) string {
	output = internalPathWithSource.ReplaceAllString(output, "$1")
	output = relativeSourcePath.ReplaceAllString(output, "$1")
	output = remainingInternalPrefix.ReplaceAllString(output, "")
	return strings.TrimSpace(output)
}
