package domain

import "strings"

func NormalizeOutput(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")

	lines := strings.Split(value, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}

	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func OutputEqual(actual, expected string) bool {
	return NormalizeOutput(actual) == NormalizeOutput(expected)
}
