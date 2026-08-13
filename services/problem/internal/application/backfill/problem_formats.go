// Package backfill contains explicitly-invoked migrations for legacy Problem
// content. It is deliberately not wired into the Problem service startup path.
package backfill

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"go-judge-system/services/problem/internal/domain/entity"
)

var formatHeaderPattern = regexp.MustCompile(`(?i)^(?:#{1,6}\s+)?(input(?:\s+format)?|output(?:\s+format)?)\s*:?\s*$`)
var sourceAttributionPattern = regexp.MustCompile(`(?i)^source\s*:`)

type ProblemFormatRepository interface {
	ListForFormatBackfill(ctx context.Context) ([]*entity.Problem, error)
	UpdateFormatsForBackfill(ctx context.Context, id int64, description, inputFormat, outputFormat string) (bool, error)
}

type Result struct {
	Scanned           int
	Migrated          int
	SkippedPopulated  int
	SkippedNoHeaders  int
	SkippedAmbiguous  int
	SkippedEmptyPart  int
	ConcurrentSkipped int
}

func (r Result) Skipped() int {
	return r.SkippedPopulated + r.SkippedNoHeaders + r.SkippedAmbiguous + r.SkippedEmptyPart + r.ConcurrentSkipped
}

// SplitLegacyDescription recognizes only standalone Input and Output headings.
// It intentionally rejects duplicates, inverted sections, and missing content.
func SplitLegacyDescription(description string) (string, string, string, string) {
	lines := strings.Split(strings.ReplaceAll(description, "\r\n", "\n"), "\n")
	inputIndex, outputIndex := -1, -1

	for index, line := range lines {
		match := formatHeaderPattern.FindStringSubmatch(strings.TrimSpace(line))
		if match == nil {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(match[1])) {
		case "input", "input format":
			if inputIndex >= 0 {
				return "", "", "", "ambiguous: multiple Input headings"
			}
			inputIndex = index
		case "output", "output format":
			if outputIndex >= 0 {
				return "", "", "", "ambiguous: multiple Output headings"
			}
			outputIndex = index
		}
	}

	if inputIndex < 0 || outputIndex < 0 {
		return "", "", "", "no standalone Input and Output headings"
	}
	if inputIndex >= outputIndex {
		return "", "", "", "ambiguous: Input must precede Output"
	}

	leading := trimSection(lines[:inputIndex])
	input := trimSection(lines[inputIndex+1 : outputIndex])
	outputLines := lines[outputIndex+1:]
	output, attribution := splitTrailingAttribution(outputLines)
	if attribution != "" {
		leading = joinSections(leading, attribution)
	}
	if leading == "" || input == "" || output == "" {
		return "", "", "", "empty description, input format, or output format"
	}
	return leading, input, output, ""
}

func splitTrailingAttribution(lines []string) (string, string) {
	for index, line := range lines {
		if sourceAttributionPattern.MatchString(strings.TrimSpace(line)) {
			return trimSection(lines[:index]), trimSection(lines[index:])
		}
	}
	return trimSection(lines), ""
}

func trimSection(lines []string) string {
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func joinSections(parts ...string) string {
	nonEmpty := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			nonEmpty = append(nonEmpty, trimmed)
		}
	}
	return strings.Join(nonEmpty, "\n\n")
}

// Run records only identifiers and reasons. With apply=false it never writes.
func Run(ctx context.Context, repository ProblemFormatRepository, apply bool, out io.Writer) (Result, error) {
	problems, err := repository.ListForFormatBackfill(ctx)
	if err != nil {
		return Result{}, err
	}

	result := Result{Scanned: len(problems)}
	for _, problem := range problems {
		if strings.TrimSpace(problem.InputFormat) != "" || strings.TrimSpace(problem.OutputFormat) != "" {
			result.SkippedPopulated++
			fmt.Fprintf(out, "SKIP id=%d slug=%s reason=formats already populated\n", problem.ID, problem.TitleSlug)
			continue
		}

		description, inputFormat, outputFormat, reason := SplitLegacyDescription(problem.Description)
		if reason != "" {
			switch {
			case strings.HasPrefix(reason, "no standalone"):
				result.SkippedNoHeaders++
			case strings.HasPrefix(reason, "empty"):
				result.SkippedEmptyPart++
			default:
				result.SkippedAmbiguous++
			}
			fmt.Fprintf(out, "SKIP id=%d slug=%s reason=%s\n", problem.ID, problem.TitleSlug, reason)
			continue
		}

		if !apply {
			fmt.Fprintf(out, "READY id=%d slug=%s\n", problem.ID, problem.TitleSlug)
			continue
		}

		updated, err := repository.UpdateFormatsForBackfill(ctx, problem.ID, description, inputFormat, outputFormat)
		if err != nil {
			return result, fmt.Errorf("update id=%d slug=%s: %w", problem.ID, problem.TitleSlug, err)
		}
		if !updated {
			result.ConcurrentSkipped++
			fmt.Fprintf(out, "SKIP id=%d slug=%s reason=formats changed concurrently\n", problem.ID, problem.TitleSlug)
			continue
		}
		result.Migrated++
		fmt.Fprintf(out, "MIGRATED id=%d slug=%s\n", problem.ID, problem.TitleSlug)
	}
	return result, nil
}
