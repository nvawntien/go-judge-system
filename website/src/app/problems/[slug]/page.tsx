"use client";

/**
 * Problem Detail — Split view: description (left) + code editor (right).
 * plan.md §6: Resizable split view, lazy-loaded editor, persisted layout state.
 */

import { useState, use } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { useToast } from "@/components/ui/Toast";
import { problemApi } from "@/lib/api-client";
import { useFetch } from "@/hooks/useFetch";
import { useSubmissionMachine } from "@/hooks/useSubmissionMachine";
import { useResizable } from "@/hooks/useResizable";
import { useCodeStorage } from "@/hooks/useCodeStorage";
import { Editor } from "@monaco-editor/react";
import { useTheme } from "@/lib/theme-context";
import { DifficultyBadge, StatusBadge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import {
  Play,
  ChevronDown,
  RefreshCw,
  AlertCircle,
  GripVertical,
} from "lucide-react";
import type { Language } from "@/types/api";

const DEFAULT_CODE: Record<Language, string> = {
  C: '#include <stdio.h>\n\nint main() {\n    // your code here\n    return 0;\n}',
  CPP: '#include <iostream>\nusing namespace std;\n\nint main() {\n    // your code here\n    return 0;\n}',
  JAVA: 'import java.util.Scanner;\n\npublic class Main {\n    public static void main(String[] args) {\n        // your code here\n    }\n}',
  PYTHON: 'def solve():\n    # your code here\n    pass\n\nif __name__ == "__main__":\n    solve()',
  GO: 'package main\n\nimport "fmt"\n\nfunc main() {\n    // your code here\n}',
  JAVASCRIPT:
    'function solve() {\n    // your code here\n}\n\nsolve();',
};

const LANGUAGES: Language[] = ["CPP", "PYTHON", "JAVA", "GO", "JAVASCRIPT", "C"];

const LANG_MONACO_MAP: Record<Language, string> = {
  C: "c",
  CPP: "cpp",
  JAVA: "java",
  PYTHON: "python",
  GO: "go",
  JAVASCRIPT: "javascript",
};

export default function ProblemDetailPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = use(params);
  const { state: authState } = useAuth();
  const { theme } = useTheme();
  const { addToast } = useToast();
  const router = useRouter();

  const [language, setLanguage] = useState<Language>("CPP");
  const { code, setCode, resetCode } = useCodeStorage(slug, language, DEFAULT_CODE[language]);

  // Resizable split view (plan.md §6)
  const { containerRef, handleMouseDown, leftStyle, rightStyle } = useResizable({
    storageKey: "oj-split-ratio",
    defaultRatio: 0.5,
    minRatio: 0.3,
    maxRatio: 0.7,
  });

  const {
    status,
    data: problem,
    error,
  } = useFetch(() => problemApi.getBySlug(slug), [slug]);

  const {
    state: submissionState,
    submit,
    reset: resetSubmission,
  } = useSubmissionMachine();

  const handleLanguageChange = (lang: Language) => {
    setLanguage(lang);
    // useCodeStorage auto-loads persisted code for this problem+lang
  };

  const handleSubmit = () => {
    if (authState.status !== "AUTHENTICATED") {
      addToast("error", "You must be logged in to submit code");
      router.push("/auth/login");
      return;
    }
    if (!problem) return;

    submit({
      problem_id: problem.id,
      problem_name: problem.title,
      language,
      source_code: code,
    });
  };

  // Loading state
  if (status === "LOADING" || status === "IDLE") {
    return (
      <div className="h-[calc(100vh-4rem)] flex items-center justify-center">
        <RefreshCw
          className="animate-spin text-[var(--oj-accent)]"
          size={32}
        />
      </div>
    );
  }

  // Error state
  if (status === "ERROR" || !problem) {
    return (
      <div className="h-[calc(100vh-4rem)] flex flex-col items-center justify-center text-center p-4">
        <AlertCircle size={48} className="text-[var(--oj-wa-txt)] mb-4" />
        <h2 className="text-2xl font-bold text-[var(--oj-text)] mb-2">
          Problem Not Found
        </h2>
        <p className="text-[var(--oj-muted)] mb-6">
          {error || "The requested problem does not exist."}
        </p>
        <Button onClick={() => router.push("/problems")}>
          Back to Problems
        </Button>
      </div>
    );
  }

  const isSubmitting =
    submissionState.status === "SUBMITTING" ||
    submissionState.status === "QUEUED" ||
    submissionState.status === "RUNNING";

  return (
    <div ref={containerRef} className="h-[calc(100vh-4rem)] flex flex-col md:flex-row overflow-hidden bg-[var(--oj-bg)]">
      {/* ━━━ Left Panel: Problem Description ━━━ */}
      <div className="overflow-y-auto border-r border-[var(--oj-border)] p-6 md:p-8 max-md:flex-1" style={leftStyle}>
        <div className="max-w-3xl mx-auto">
          {/* Title + Difficulty */}
          <div className="flex items-center gap-4 mb-4">
            <h1 className="text-3xl font-bold text-[var(--oj-text)]">
              {problem.title}
            </h1>
            <DifficultyBadge difficulty={problem.difficulty} />
          </div>

          {/* Limits bar */}
          <div className="flex flex-wrap gap-4 text-sm text-[var(--oj-muted)] mb-8 font-code p-3 bg-[var(--oj-surface)] rounded-md border border-[var(--oj-border)]">
            <div>
              <span className="font-semibold text-[var(--oj-body)]">
                Time Limit:
              </span>{" "}
              {problem.time_limit}s
            </div>
            <div>
              <span className="font-semibold text-[var(--oj-body)]">
                Memory Limit:
              </span>{" "}
              {problem.memory_limit}MB
            </div>
          </div>

          {/* Description */}
          <div className="prose prose-sm md:prose-base max-w-none text-[var(--oj-body)] mb-8">
            <div
              dangerouslySetInnerHTML={{
                __html: problem.description.replace(/\n/g, "<br/>"),
              }}
            />
          </div>

          {/* Examples */}
          {problem.examples && problem.examples.length > 0 && (
            <div className="space-y-6 mb-8">
              <h3 className="text-xl font-bold text-[var(--oj-text)]">
                Examples
              </h3>
              {problem.examples.map((ex, i) => (
                <div
                  key={i}
                  className="bg-[var(--oj-surface)] border border-[var(--oj-border)] rounded-lg overflow-hidden"
                >
                  <div className="bg-[var(--oj-panel)] px-4 py-2 border-b border-[var(--oj-border)] font-semibold text-sm text-[var(--oj-text)]">
                    Example {i + 1}
                  </div>
                  <div className="p-4 space-y-4 text-sm font-code">
                    <div>
                      <div className="text-[var(--oj-muted)] mb-1">Input:</div>
                      <div className="bg-[var(--oj-bg)] p-3 rounded border border-[var(--oj-border)] whitespace-pre-wrap text-[var(--oj-code-txt)]">
                        {ex.input}
                      </div>
                    </div>
                    <div>
                      <div className="text-[var(--oj-muted)] mb-1">Output:</div>
                      <div className="bg-[var(--oj-bg)] p-3 rounded border border-[var(--oj-border)] whitespace-pre-wrap text-[var(--oj-code-txt)]">
                        {ex.output}
                      </div>
                    </div>
                    {ex.explanation && (
                      <div className="text-[var(--oj-body)] font-sans mt-2">
                        <span className="font-semibold">Explanation:</span>{" "}
                        {ex.explanation}
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Constraints */}
          {problem.constraints && (
            <div className="mb-8">
              <h3 className="text-xl font-bold text-[var(--oj-text)] mb-3">
                Constraints
              </h3>
              <div className="bg-[var(--oj-surface)] border border-[var(--oj-border)] rounded-lg p-4 font-code text-sm whitespace-pre-wrap text-[var(--oj-code-txt)]">
                {problem.constraints}
              </div>
            </div>
          )}

          {/* Hints */}
          {problem.hints && problem.hints.length > 0 && (
            <div className="mb-8">
              <h3 className="text-xl font-bold text-[var(--oj-text)] mb-3">
                Hints
              </h3>
              <ul className="list-disc ml-5 space-y-2 text-[var(--oj-body)]">
                {problem.hints.map((hint, i) => (
                  <li key={i}>{hint}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      </div>

      {/* ━━━ Resizable Divider (plan.md §6) ━━━ */}
      <div
        onMouseDown={handleMouseDown}
        className="hidden md:flex items-center justify-center w-2 cursor-col-resize bg-[var(--oj-border)] hover:bg-[var(--oj-accent)] active:bg-[var(--oj-accent)] transition-colors group flex-shrink-0"
        role="separator"
        aria-label="Resize panels"
      >
        <GripVertical size={14} className="text-[var(--oj-muted)] group-hover:text-white" />
      </div>

      {/* ━━━ Right Panel: Code Editor ━━━ */}
      <div className="flex flex-col h-[50vh] md:h-auto border-t md:border-t-0 border-[var(--oj-border)] relative max-md:flex-1" style={rightStyle}>
        {/* Editor toolbar */}
        <div className="h-12 bg-[var(--oj-surface)] border-b border-[var(--oj-border)] flex items-center justify-between px-4">
          <select
            value={language}
            onChange={(e) => handleLanguageChange(e.target.value as Language)}
            className="cursor-pointer bg-[var(--oj-bg)] border border-[var(--oj-border)] rounded text-sm px-2 py-1 focus:outline-none focus:ring-1 focus:ring-[var(--oj-accent)] text-[var(--oj-text)] font-code"
            aria-label="Select language"
          >
            {LANGUAGES.map((l) => (
              <option key={l} value={l}>
                {l}
              </option>
            ))}
          </select>

          <div className="flex items-center gap-2">
            <Button
              size="sm"
              variant="secondary"
              icon={<RefreshCw size={14} />}
              onClick={resetCode}
              title="Reset code"
              aria-label="Reset code to template"
            />
            <Button
              size="sm"
              icon={<Play size={14} fill="currentColor" />}
              onClick={handleSubmit}
              loading={isSubmitting}
            >
              Submit
            </Button>
          </div>
        </div>

        {/* Editor */}
        <div className="flex-1 relative">
          <Editor
            height="100%"
            language={LANG_MONACO_MAP[language]}
            value={code}
            onChange={(val) => setCode(val || "")}
            theme={theme === "dark" ? "vs-dark" : "light"}
            options={{
              minimap: { enabled: false },
              fontSize: 14,
              fontFamily: '"JetBrains Mono", monospace',
              fontLigatures: true,
              scrollBeyondLastLine: false,
              automaticLayout: true,
              padding: { top: 16 },
            }}
          />
        </div>

        {/* ━━━ Verdict Slide-up Panel (plan.md §5) ━━━ */}
        {submissionState.status !== "IDLE" && (
          <div
            className="absolute bottom-0 left-0 right-0 bg-[var(--oj-surface)] border-t-2 border-[var(--oj-accent)] shadow-2xl oj-slide-up max-h-[50%] flex flex-col"
            style={{ zIndex: "var(--z-drawer)" } as React.CSSProperties}
          >
            <div className="flex items-center justify-between p-3 border-b border-[var(--oj-border)] bg-[var(--oj-panel)]">
              <h3 className="font-bold text-[var(--oj-text)] flex items-center gap-2">
                Submission Result
                {(submissionState.status === "QUEUED" ||
                  submissionState.status === "RUNNING") && (
                  <span className="text-xs font-normal text-[var(--oj-muted)] animate-pulse">
                    {submissionState.status === "QUEUED"
                      ? "In Queue..."
                      : "Running Tests..."}
                  </span>
                )}
              </h3>
              <button
                onClick={resetSubmission}
                className="cursor-pointer p-1 hover:bg-[var(--oj-border)] rounded transition-colors text-[var(--oj-muted)]"
                aria-label="Close verdict panel"
              >
                <ChevronDown size={18} />
              </button>
            </div>

            <div className="p-4 overflow-y-auto flex-1">
              {submissionState.status === "SUBMITTING" && (
                <div className="flex items-center gap-3 text-[var(--oj-pd-txt)]">
                  <div className="oj-spinner" /> Submitting...
                </div>
              )}

              {submissionState.status === "ERROR" && (
                <div className="text-[var(--oj-wa-txt)] bg-[var(--oj-wa-bg)] p-3 rounded border border-[var(--oj-wa-txt)]/20">
                  {submissionState.error}
                </div>
              )}

              {submissionState.status === "RESULT" && (
                <div className="space-y-4">
                  <div className="flex items-center gap-4">
                    <StatusBadge
                      status={submissionState.verdict}
                      className="text-base px-3 py-1"
                    />
                    <div className="flex gap-4 text-sm font-code text-[var(--oj-muted)]">
                      <div>
                        <span className="font-semibold text-[var(--oj-body)]">
                          Time:
                        </span>{" "}
                        {submissionState.detail.execution_time_ms ?? 0} ms
                      </div>
                      <div>
                        <span className="font-semibold text-[var(--oj-body)]">
                          Memory:
                        </span>{" "}
                        {submissionState.detail.memory_used_kb ?? 0} KB
                      </div>
                    </div>
                  </div>

                  {/* Compile output */}
                  {submissionState.detail.compile_output && (
                    <div>
                      <div className="font-semibold text-sm mb-1 text-[var(--oj-text)]">
                        Compiler Output:
                      </div>
                      <div className="bg-[var(--oj-code-bg)] text-[var(--oj-code-txt)] p-3 rounded font-code text-sm overflow-x-auto whitespace-pre">
                        {submissionState.detail.compile_output}
                      </div>
                    </div>
                  )}

                  {/* Failed test case */}
                  {submissionState.detail.failed_test && (
                    <div className="border border-[var(--oj-border)] rounded-lg overflow-hidden">
                      <div className="bg-[var(--oj-wa-bg)] px-3 py-2 text-sm font-semibold text-[var(--oj-wa-txt)] border-b border-[var(--oj-border)]">
                        Failed Test Case{" "}
                        {submissionState.detail.failed_test.test_index}
                      </div>
                      <div className="p-3 space-y-3 font-code text-sm bg-[var(--oj-bg)]">
                        <div>
                          <div className="text-[var(--oj-muted)] mb-1">
                            Input:
                          </div>
                          <div className="bg-[var(--oj-surface)] p-2 rounded border border-[var(--oj-border)] whitespace-pre-wrap">
                            {submissionState.detail.failed_test.input ||
                              "<hidden>"}
                          </div>
                        </div>
                        <div>
                          <div className="text-[var(--oj-muted)] mb-1">
                            Expected:
                          </div>
                          <div className="bg-[var(--oj-surface)] p-2 rounded border border-[var(--oj-border)] whitespace-pre-wrap">
                            {submissionState.detail.failed_test
                              .expected_output || "<hidden>"}
                          </div>
                        </div>
                        <div>
                          <div className="text-[var(--oj-wa-txt)] mb-1 font-semibold">
                            Actual:
                          </div>
                          <div className="bg-[var(--oj-wa-bg)] p-2 rounded border border-[var(--oj-border)] text-[var(--oj-wa-txt)] whitespace-pre-wrap">
                            {submissionState.detail.failed_test.actual_output ||
                              "<hidden>"}
                          </div>
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
