import type { LanguageCode } from './types';

/**
 * Starter templates. The judge feeds test-case input on stdin and compares
 * stdout (pkg/gojudge/config.go builds/runs main.<ext> / Main.java), so every
 * template is a stdin → stdout skeleton.
 */
export const CODE_TEMPLATES: Record<LanguageCode, string> = {
  GO: `package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	var n int
	if _, err := fmt.Fscan(reader, &n); err != nil {
		return
	}

	fmt.Fprintln(writer, n)
}
`,
  CPP: `#include <bits/stdc++.h>
using namespace std;

int main() {
    ios::sync_with_stdio(false);
    cin.tie(nullptr);

    int n;
    if (!(cin >> n)) return 0;

    cout << n << "\\n";
    return 0;
}
`,
  PYTHON: `import sys


def main() -> None:
    data = sys.stdin.read().split()
    if not data:
        return

    n = int(data[0])
    print(n)


if __name__ == "__main__":
    main()
`,
  JAVA: `import java.io.*;
import java.util.*;

public class Main {
    public static void main(String[] args) throws IOException {
        BufferedReader br = new BufferedReader(new InputStreamReader(System.in));
        StreamTokenizer in = new StreamTokenizer(br);

        if (in.nextToken() == StreamTokenizer.TT_EOF) return;
        int n = (int) in.nval;

        System.out.println(n);
    }
}
`,
};

/** Key for the per-problem, per-language draft kept in localStorage. */
export function draftKey(problemId: number, language: LanguageCode): string {
  return `astra-draft:${problemId}:${language}`;
}
