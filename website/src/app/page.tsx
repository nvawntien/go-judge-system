import Link from "next/link";
import { PageLayout } from "@/components/layout/PageLayout";
import { Code2, Trophy, Zap } from "lucide-react";

export default function Home() {
  return (
    <PageLayout className="flex flex-col items-center justify-center text-center py-20">
      {/* Hero icon */}
      <div className="mb-8 p-4 rounded-full bg-[var(--oj-accent-fill)] text-[var(--oj-accent)] inline-flex">
        <Zap size={32} />
      </div>

      <h1 className="text-5xl font-black mb-6 text-[var(--oj-text)] tracking-tight">
        Master Your <span className="text-[var(--oj-accent)]">Algorithms</span>
      </h1>

      <p className="text-xl text-[var(--oj-muted)] mb-12 max-w-2xl">
        A high-performance online judge for competitive programming. Practice
        problems, submit code in multiple languages, and track your progress.
      </p>

      <div className="flex gap-4 mb-20">
        <Link
          href="/problems"
          className="px-8 py-3 rounded-lg font-bold text-white bg-[var(--oj-accent)] hover:bg-[var(--oj-accent-dk)] transition-all shadow-lg hover:shadow-xl hover:-translate-y-0.5"
        >
          Explore Problems
        </Link>
        <Link
          href="/submissions"
          className="px-8 py-3 rounded-lg font-bold text-[var(--oj-text)] bg-[var(--oj-surface)] border border-[var(--oj-border)] hover:bg-[var(--oj-panel)] transition-colors shadow-sm"
        >
          Recent Submissions
        </Link>
      </div>

      {/* Feature cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-8 w-full max-w-4xl text-left">
        <div className="p-6 rounded-xl border border-[var(--oj-border)] bg-[var(--oj-surface)] hover:border-[var(--oj-border-acc)] transition-colors">
          <Code2 className="text-[var(--oj-accent)] mb-4" size={28} />
          <h3 className="text-lg font-bold text-[var(--oj-text)] mb-2">
            Multiple Languages
          </h3>
          <p className="text-[var(--oj-body)]">
            Support for C, C++, Java, Python, Go, and JavaScript with precise
            execution environments.
          </p>
        </div>
        <div className="p-6 rounded-xl border border-[var(--oj-border)] bg-[var(--oj-surface)] hover:border-[var(--oj-border-acc)] transition-colors">
          <Zap className="text-[var(--oj-accent)] mb-4" size={28} />
          <h3 className="text-lg font-bold text-[var(--oj-text)] mb-2">
            Lightning Fast
          </h3>
          <p className="text-[var(--oj-body)]">
            Event-driven architecture ensures your submissions are judged in
            milliseconds.
          </p>
        </div>
        <div className="p-6 rounded-xl border border-[var(--oj-border)] bg-[var(--oj-surface)] hover:border-[var(--oj-border-acc)] transition-colors">
          <Trophy className="text-[var(--oj-accent)] mb-4" size={28} />
          <h3 className="text-lg font-bold text-[var(--oj-text)] mb-2">
            Track Progress
          </h3>
          <p className="text-[var(--oj-body)]">
            Detailed execution time, memory usage, and test case feedback for
            every submission.
          </p>
        </div>
      </div>
    </PageLayout>
  );
}
