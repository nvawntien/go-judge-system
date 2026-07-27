package domain

import "testing"

func TestOutputEqualNormalizesLineEndingsAndTrailingWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
		want     bool
	}{
		{name: "plain", actual: "10", expected: "10\n", want: true},
		{name: "trailing spaces", actual: "10  \n", expected: "10", want: true},
		{name: "crlf", actual: "a b\r\nc\t\r\n", expected: "a b\nc", want: true},
		{name: "middle whitespace matters", actual: "a b", expected: "ab", want: false},
		{name: "internal repeated spaces matter", actual: "a  b", expected: "a b", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := OutputEqual(tt.actual, tt.expected); got != tt.want {
				t.Fatalf("OutputEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
