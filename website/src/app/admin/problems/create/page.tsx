"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { PageLayout } from "@/components/layout/PageLayout";
import { Input } from "@/components/ui/Input";
import { Button } from "@/components/ui/Button";
import { problemApi, ApiError } from "@/lib/api-client";
import { useAuth } from "@/lib/auth-context";
import { useToast } from "@/components/ui/Toast";
import { ArrowLeft, Plus, Trash2, Save } from "lucide-react";
import Link from "next/link";
import type { CreateProblemRequest, ProblemExampleDTO, Difficulty } from "@/types/api";

export default function CreateProblemPage() {
  const router = useRouter();
  const { state: authState } = useAuth();
  const { addToast } = useToast();

  const [loading, setLoading] = useState(false);
  const [formData, setFormData] = useState<CreateProblemRequest>({
    title: "",
    slug: "",
    description: "",
    difficulty: "EASY",
    examples: [{ input: "", output: "", explanation: "" }],
    constraints: "",
    hints: [""],
    time_limit: 1.0,
    memory_limit: 256,
  });

  const isAdmin =
    authState.status === "AUTHENTICATED" &&
    (authState.user.role === "admin" || authState.user.role === "super_admin");

  if (authState.status !== "AUTHENTICATING" && !isAdmin) {
    router.push("/");
    return null;
  }

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>
  ) => {
    const { id, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [id]: id === "time_limit" || id === "memory_limit" ? Number(value) : value,
    }));
  };

  const handleExampleChange = (index: number, field: keyof ProblemExampleDTO, value: string) => {
    setFormData((prev) => {
      const newExamples = [...prev.examples];
      newExamples[index] = { ...newExamples[index], [field]: value };
      return { ...prev, examples: newExamples };
    });
  };

  const handleHintChange = (index: number, value: string) => {
    setFormData((prev) => {
      const newHints = [...(prev.hints || [])];
      newHints[index] = value;
      return { ...prev, hints: newHints };
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);

    try {
      // Filter out empty hints
      const submitData = {
        ...formData,
        hints: formData.hints?.filter((h) => h.trim() !== ""),
      };
      
      await problemApi.create(submitData);
      addToast("success", "Problem created successfully.");
      router.push("/admin/problems");
    } catch (err) {
      addToast(
        "error",
        err instanceof ApiError ? err.message : "Failed to create problem"
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <PageLayout>
      <div className="max-w-4xl mx-auto space-y-6 pb-20">
        <div className="flex items-center gap-4">
          <Link
            href="/admin/problems"
            className="p-2 -ml-2 rounded-md hover:bg-[var(--oj-surface)] text-[var(--oj-muted)] hover:text-[var(--oj-text)] transition-colors"
          >
            <ArrowLeft size={20} />
          </Link>
          <div>
            <h1 className="text-3xl font-bold text-[var(--oj-text)]">
              Create New Problem
            </h1>
            <p className="text-[var(--oj-muted)] mt-1">
              Add a new competitive programming challenge.
            </p>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="space-y-8 bg-[var(--oj-surface)] border border-[var(--oj-border)] rounded-xl p-6 md:p-8">
          
          <div className="space-y-4">
            <h2 className="text-xl font-semibold text-[var(--oj-text)] border-b border-[var(--oj-border)] pb-2">
              General Information
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Input
                label="Problem Title"
                id="title"
                required
                value={formData.title}
                onChange={handleChange}
                placeholder="e.g. Two Sum"
              />
              <Input
                label="Slug (URL identifier)"
                id="slug"
                required
                value={formData.slug}
                onChange={handleChange}
                placeholder="e.g. two-sum"
              />
            </div>
            
            <div>
              <label className="block text-sm font-medium text-[var(--oj-text)] mb-1">
                Difficulty
              </label>
              <select
                id="difficulty"
                value={formData.difficulty}
                onChange={handleChange}
                className="w-full h-10 px-3 rounded-lg bg-[var(--oj-bg)] border border-[var(--oj-border)] text-[var(--oj-text)] focus:outline-none focus:ring-2 focus:ring-[var(--oj-accent)]"
              >
                <option value="EASY">Easy</option>
                <option value="MEDIUM">Medium</option>
                <option value="HARD">Hard</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-[var(--oj-text)] mb-1">
                Description (Markdown)
              </label>
              <textarea
                id="description"
                required
                rows={10}
                value={formData.description}
                onChange={handleChange}
                className="w-full p-3 rounded-lg bg-[var(--oj-bg)] border border-[var(--oj-border)] text-[var(--oj-text)] focus:outline-none focus:ring-2 focus:ring-[var(--oj-accent)] resize-y font-mono text-sm"
                placeholder="Describe the problem..."
              />
            </div>
          </div>

          <div className="space-y-4">
            <h2 className="text-xl font-semibold text-[var(--oj-text)] border-b border-[var(--oj-border)] pb-2">
              Limits & Constraints
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Input
                label="Time Limit (Seconds)"
                id="time_limit"
                type="number"
                step="0.1"
                min="0.1"
                max="10"
                required
                value={formData.time_limit}
                onChange={handleChange}
              />
              <Input
                label="Memory Limit (MB)"
                id="memory_limit"
                type="number"
                min="16"
                max="1024"
                required
                value={formData.memory_limit}
                onChange={handleChange}
              />
            </div>
            
            <div>
              <label className="block text-sm font-medium text-[var(--oj-text)] mb-1">
                Constraints (Markdown)
              </label>
              <textarea
                id="constraints"
                rows={3}
                value={formData.constraints}
                onChange={handleChange}
                className="w-full p-3 rounded-lg bg-[var(--oj-bg)] border border-[var(--oj-border)] text-[var(--oj-text)] focus:outline-none focus:ring-2 focus:ring-[var(--oj-accent)] font-mono text-sm"
                placeholder="- `1 <= nums.length <= 10^4`"
              />
            </div>
          </div>

          <div className="space-y-4">
            <div className="flex items-center justify-between border-b border-[var(--oj-border)] pb-2">
              <h2 className="text-xl font-semibold text-[var(--oj-text)]">
                Examples
              </h2>
              <Button
                type="button"
                variant="secondary"
                size="sm"
                icon={<Plus size={14} />}
                onClick={() => setFormData(prev => ({
                  ...prev,
                  examples: [...prev.examples, { input: "", output: "", explanation: "" }]
                }))}
              >
                Add Example
              </Button>
            </div>
            
            {formData.examples.map((example, idx) => (
              <div key={idx} className="p-4 bg-[var(--oj-panel)] border border-[var(--oj-border)] rounded-lg relative">
                <div className="absolute top-4 right-4">
                  <button
                    type="button"
                    onClick={() => setFormData(prev => ({
                      ...prev,
                      examples: prev.examples.filter((_, i) => i !== idx)
                    }))}
                    className="text-[var(--oj-muted)] hover:text-[var(--oj-wa-txt)] transition-colors"
                  >
                    <Trash2 size={16} />
                  </button>
                </div>
                <h3 className="font-semibold text-sm mb-3 text-[var(--oj-text)]">Example {idx + 1}</h3>
                <div className="space-y-3">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    <div>
                      <label className="block text-xs text-[var(--oj-muted)] mb-1">Input</label>
                      <textarea
                        required
                        value={example.input}
                        onChange={(e) => handleExampleChange(idx, "input", e.target.value)}
                        className="w-full p-2 rounded bg-[var(--oj-bg)] border border-[var(--oj-border)] text-[var(--oj-text)] font-mono text-xs focus:ring-2 focus:ring-[var(--oj-accent)]"
                        rows={3}
                      />
                    </div>
                    <div>
                      <label className="block text-xs text-[var(--oj-muted)] mb-1">Output</label>
                      <textarea
                        required
                        value={example.output}
                        onChange={(e) => handleExampleChange(idx, "output", e.target.value)}
                        className="w-full p-2 rounded bg-[var(--oj-bg)] border border-[var(--oj-border)] text-[var(--oj-text)] font-mono text-xs focus:ring-2 focus:ring-[var(--oj-accent)]"
                        rows={3}
                      />
                    </div>
                  </div>
                  <div>
                    <label className="block text-xs text-[var(--oj-muted)] mb-1">Explanation (Optional)</label>
                    <Input
                      id={`exp-${idx}`}
                      value={example.explanation || ""}
                      onChange={(e) => handleExampleChange(idx, "explanation", e.target.value)}
                      placeholder="e.g. Because nums[0] + nums[1] == 9, we return [0, 1]."
                    />
                  </div>
                </div>
              </div>
            ))}
          </div>

          <div className="space-y-4">
            <div className="flex items-center justify-between border-b border-[var(--oj-border)] pb-2">
              <h2 className="text-xl font-semibold text-[var(--oj-text)]">
                Hints
              </h2>
              <Button
                type="button"
                variant="secondary"
                size="sm"
                icon={<Plus size={14} />}
                onClick={() => setFormData(prev => ({
                  ...prev,
                  hints: [...(prev.hints || []), ""]
                }))}
              >
                Add Hint
              </Button>
            </div>
            {formData.hints?.map((hint, idx) => (
              <div key={idx} className="flex gap-2 items-center">
                <Input
                  id={`hint-${idx}`}
                  value={hint}
                  onChange={(e) => handleHintChange(idx, e.target.value)}
                  placeholder={`Hint ${idx + 1}`}
                  className="flex-1"
                />
                <button
                  type="button"
                  onClick={() => setFormData(prev => ({
                    ...prev,
                    hints: prev.hints?.filter((_, i) => i !== idx)
                  }))}
                  className="p-2 text-[var(--oj-muted)] hover:text-[var(--oj-wa-txt)] transition-colors mt-6"
                >
                  <Trash2 size={18} />
                </button>
              </div>
            ))}
          </div>

          <div className="pt-6 border-t border-[var(--oj-border)] flex justify-end gap-3">
            <Link href="/admin/problems">
              <Button type="button" variant="secondary">Cancel</Button>
            </Link>
            <Button type="submit" loading={loading} icon={<Save size={16} />}>
              Create Problem
            </Button>
          </div>
        </form>
      </div>
    </PageLayout>
  );
}
