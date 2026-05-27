"use client";

import { useState, useEffect } from "react";
import { PageLayout } from "@/components/layout/PageLayout";
import { useAuth } from "@/lib/auth-context";
import { problemApi } from "@/lib/api-client";
import { ListProblemsResponse, ProblemResponse } from "@/types/api";
import { StatusBadge, DifficultyBadge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { useToast } from "@/components/ui/Toast";
import { Skeleton } from "@/components/ui/Skeleton";
import { Eye, EyeOff, Edit, Trash2, Plus, UploadCloud } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";

export default function AdminProblemsPage() {
  const { state: authState } = useAuth();
  const router = useRouter();
  const { addToast } = useToast();

  const [data, setData] = useState<ListProblemsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");

  const isAdmin =
    authState.status === "AUTHENTICATED" &&
    (authState.user.role === "admin" || authState.user.role === "super_admin");

  useEffect(() => {
    if (authState.status === "AUTHENTICATED" && !isAdmin) {
      router.push("/");
    }
  }, [authState, isAdmin, router]);

  const loadProblems = async () => {
    setLoading(true);
    try {
      const result = await problemApi.listAdmin({ page, limit: 20, search });
      setData(result);
    } catch (err) {
      addToast("error", "Failed to load admin problems");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (isAdmin) {
      loadProblems();
    }
  }, [isAdmin, page, search]);

  const handleToggleVisibility = async (problem: ProblemResponse) => {
    try {
      if (problem.is_hidden) {
        await problemApi.publish(problem.id);
        addToast("success", "Problem published successfully.");
      } else {
        await problemApi.hide(problem.id);
        addToast("success", "Problem hidden successfully.");
      }
      loadProblems();
    } catch (err) {
      addToast("error", "Failed to change visibility.");
    }
  };

  const handleDelete = async (id: number) => {
    if (!window.confirm("Are you sure you want to delete this problem?")) return;
    try {
      await problemApi.delete(id);
      addToast("success", "Problem deleted.");
      loadProblems();
    } catch (err) {
      addToast("error", "Failed to delete problem.");
    }
  };

  if (authState.status === "AUTHENTICATING" || !isAdmin) {
    return (
      <PageLayout>
        <div className="flex justify-center p-12">
          <Skeleton className="h-8 w-32" />
        </div>
      </PageLayout>
    );
  }

  return (
    <PageLayout>
      <div className="max-w-6xl mx-auto space-y-6">
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
          <div>
            <h1 className="text-3xl font-bold text-[var(--oj-text)]">
              Problem Management
            </h1>
            <p className="text-[var(--oj-muted)]">
              Administer system problems and test cases.
            </p>
          </div>
          <Link href="/admin/problems/create">
            <Button icon={<Plus size={16} />}>Create Problem</Button>
          </Link>
        </div>

        {/* Filters */}
        <div className="flex gap-4 p-4 bg-[var(--oj-surface)] border border-[var(--oj-border)] rounded-xl">
          <Input
            placeholder="Search by title or slug..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full sm:max-w-sm"
            aria-label="Search problems"
          />
        </div>

        {/* Table */}
        <div className="bg-[var(--oj-surface)] border border-[var(--oj-border)] rounded-xl overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="border-b border-[var(--oj-border)] bg-[var(--oj-panel)]">
                  <th className="p-4 font-semibold text-sm text-[var(--oj-muted)]">ID</th>
                  <th className="p-4 font-semibold text-sm text-[var(--oj-muted)]">Title & Slug</th>
                  <th className="p-4 font-semibold text-sm text-[var(--oj-muted)]">Difficulty</th>
                  <th className="p-4 font-semibold text-sm text-[var(--oj-muted)]">Visibility</th>
                  <th className="p-4 font-semibold text-sm text-[var(--oj-muted)] text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[var(--oj-border)]">
                {loading ? (
                  Array.from({ length: 5 }).map((_, i) => (
                    <tr key={i}>
                      <td className="p-4"><Skeleton className="h-4 w-8" /></td>
                      <td className="p-4"><Skeleton className="h-4 w-32" /></td>
                      <td className="p-4"><Skeleton className="h-4 w-16" /></td>
                      <td className="p-4"><Skeleton className="h-4 w-16" /></td>
                      <td className="p-4"><Skeleton className="h-8 w-24 ml-auto" /></td>
                    </tr>
                  ))
                ) : data?.items.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="p-8 text-center text-[var(--oj-muted)]">
                      No problems found.
                    </td>
                  </tr>
                ) : (
                  data?.items.map((p) => (
                    <tr key={p.id} className="hover:bg-[var(--oj-panel)] transition-colors">
                      <td className="p-4 text-sm text-[var(--oj-muted)]">#{p.id}</td>
                      <td className="p-4">
                        <div className="font-medium text-[var(--oj-text)]">
                          {p.title}
                        </div>
                        <div className="text-xs text-[var(--oj-muted)]">
                          {p.slug}
                        </div>
                      </td>
                      <td className="p-4">
                        <DifficultyBadge difficulty={p.difficulty as any} />
                      </td>
                      <td className="p-4">
                        {p.is_hidden ? (
                          <span className="inline-flex items-center gap-1.5 px-2 py-1 rounded text-xs font-medium bg-[var(--oj-wa-bg)] text-[var(--oj-wa-txt)]">
                            <EyeOff size={12} /> Hidden
                          </span>
                        ) : (
                          <span className="inline-flex items-center gap-1.5 px-2 py-1 rounded text-xs font-medium bg-[var(--oj-ac-bg)] text-[var(--oj-ac-txt)]">
                            <Eye size={12} /> Public
                          </span>
                        )}
                      </td>
                      <td className="p-4">
                        <div className="flex items-center justify-end gap-2">
                          <Button
                            variant="secondary"
                            size="sm"
                            icon={p.is_hidden ? <Eye size={14} /> : <EyeOff size={14} />}
                            onClick={() => handleToggleVisibility(p)}
                            title={p.is_hidden ? "Publish" : "Hide"}
                          />
                          <Link href={`/admin/problems/${p.id}/edit`}>
                            <Button
                              variant="secondary"
                              size="sm"
                              icon={<Edit size={14} />}
                              title="Edit"
                            />
                          </Link>
                          <Link href={`/admin/problems/${p.id}/testcases`}>
                            <Button
                              variant="secondary"
                              size="sm"
                              icon={<UploadCloud size={14} />}
                              title="Test Cases"
                            />
                          </Link>
                          <Button
                            variant="danger"
                            size="sm"
                            icon={<Trash2 size={14} />}
                            onClick={() => handleDelete(p.id)}
                            title="Delete"
                          />
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          {data && data.total > data.limit && (
            <div className="p-4 border-t border-[var(--oj-border)] flex items-center justify-between">
              <span className="text-sm text-[var(--oj-muted)]">
                Showing {((page - 1) * data.limit) + 1} to {Math.min(page * data.limit, data.total)} of {data.total} problems
              </span>
              <div className="flex gap-2">
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={page === 1 || loading}
                  onClick={() => setPage(page - 1)}
                >
                  Previous
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={page * data.limit >= data.total || loading}
                  onClick={() => setPage(page + 1)}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </div>
      </div>
    </PageLayout>
  );
}
