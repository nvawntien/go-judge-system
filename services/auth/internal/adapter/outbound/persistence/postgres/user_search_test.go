package postgres

import "testing"

func TestEscapeLikePatternTreatsWildcardsAndEscapeAsLiterals(t *testing.T) {
	if got, want := escapeLikePattern(`ada%_\lovelace`), `ada\%\_\\lovelace`; got != want {
		t.Fatalf("escapeLikePattern() = %q, want %q", got, want)
	}
}
