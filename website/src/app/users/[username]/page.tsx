"use client";

import { useState, useEffect } from "react";
import { useParams } from "next/navigation";
import { PageLayout } from "@/components/layout/PageLayout";
import { authApi, submissionApi } from "@/lib/api-client";
import type { ProfileResponse, ListSubmissionsResponse } from "@/types/api";
import { Skeleton } from "@/components/ui/Skeleton";
import { User, Activity, Trophy, Calendar } from "lucide-react";
import { StatusBadge } from "@/components/ui/Badge";
import Link from "next/link";

export default function UserProfilePage() {
  const params = useParams();
  const username = String(params.username);

  const [profile, setProfile] = useState<ProfileResponse | null>(null);
  const [submissions, setSubmissions] = useState<ListSubmissionsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  useEffect(() => {
    if (!username) return;

    Promise.all([
      authApi.getProfile(username),
      submissionApi.listAll({ user_id: username, limit: 10 })
    ])
      .then(([profData, subData]) => {
        setProfile(profData);
        setSubmissions(subData);
      })
      .catch(() => {
        setError(true);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [username]);

  if (loading) {
    return (
      <PageLayout>
        <div className="max-w-4xl mx-auto space-y-6">
          <Skeleton className="h-48 w-full rounded-2xl" />
          <Skeleton className="h-64 w-full rounded-2xl" />
        </div>
      </PageLayout>
    );
  }

  if (error || !profile) {
    return (
      <PageLayout>
        <div className="max-w-4xl mx-auto py-12 text-center text-[var(--oj-muted)]">
          <User size={64} className="mx-auto mb-4 opacity-50" />
          <h2 className="text-2xl font-bold mb-2">User Not Found</h2>
          <p>The requested profile does not exist or is unavailable.</p>
        </div>
      </PageLayout>
    );
  }

  return (
    <PageLayout>
      <div className="max-w-4xl mx-auto space-y-8">
        
        {/* Profile Header */}
        <div className="bg-[var(--oj-surface)] border border-[var(--oj-border)] rounded-2xl p-8 flex items-start sm:items-center gap-6 flex-col sm:flex-row relative overflow-hidden">
          {/* Subtle background decoration */}
          <div className="absolute top-0 right-0 w-64 h-64 bg-[var(--oj-accent)]/5 rounded-full blur-3xl pointer-events-none -translate-y-1/2 translate-x-1/3" />
          
          <div className="w-24 h-24 rounded-full bg-[var(--oj-panel)] border-2 border-[var(--oj-accent)]/20 flex items-center justify-center flex-shrink-0 text-[var(--oj-muted)]">
            <User size={48} />
          </div>
          <div className="flex-1 space-y-3">
            <div>
              <h1 className="text-3xl font-black text-[var(--oj-text)]">
                {profile.username}
              </h1>
              {profile.email && (
                <p className="text-[var(--oj-muted)] font-mono text-sm mt-1">{profile.email}</p>
              )}
            </div>
            <div className="flex flex-wrap gap-4 pt-2">
              <div className="flex items-center gap-1.5 text-sm font-medium text-[var(--oj-text)] bg-[var(--oj-panel)] px-3 py-1.5 rounded-full border border-[var(--oj-border)]">
                <Trophy size={16} className="text-[var(--oj-accent)]" />
                Rating: <span className="font-bold">{profile.rating}</span>
              </div>
              <div className="flex items-center gap-1.5 text-sm text-[var(--oj-muted)] px-2 py-1.5">
                <Calendar size={16} />
                Joined {new Date(profile.created_at).toLocaleDateString()}
              </div>
            </div>
          </div>
        </div>

        {/* Recent Activity */}
        <div className="bg-[var(--oj-surface)] border border-[var(--oj-border)] rounded-2xl overflow-hidden">
          <div className="px-6 py-4 border-b border-[var(--oj-border)] bg-[var(--oj-panel)] flex items-center gap-2">
            <Activity size={18} className="text-[var(--oj-accent)]" />
            <h2 className="text-lg font-semibold text-[var(--oj-text)]">
              Recent Activity
            </h2>
          </div>
          
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="border-b border-[var(--oj-border)] bg-[var(--oj-panel)]/50">
                  <th className="p-4 font-semibold text-sm text-[var(--oj-muted)]">Time</th>
                  <th className="p-4 font-semibold text-sm text-[var(--oj-muted)]">Problem</th>
                  <th className="p-4 font-semibold text-sm text-[var(--oj-muted)]">Status</th>
                  <th className="p-4 font-semibold text-sm text-[var(--oj-muted)]">Language</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[var(--oj-border)]">
                {submissions?.items.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="p-8 text-center text-[var(--oj-muted)]">
                      No recent submissions.
                    </td>
                  </tr>
                ) : (
                  submissions?.items.map((sub) => (
                    <tr key={sub.id} className="hover:bg-[var(--oj-panel)] transition-colors">
                      <td className="p-4 text-sm text-[var(--oj-muted)]">
                        {new Date(sub.created_at).toLocaleString()}
                      </td>
                      <td className="p-4 font-medium">
                        <Link href={`/problems/${sub.problem_id}`} className="text-[var(--oj-text)] hover:text-[var(--oj-accent)] transition-colors">
                          {sub.problem_name}
                        </Link>
                      </td>
                      <td className="p-4">
                        <StatusBadge status={sub.status as any} />
                      </td>
                      <td className="p-4 text-sm font-mono text-[var(--oj-muted)]">
                        {sub.language}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
          
          {submissions && submissions.items.length >= 10 && (
            <div className="p-4 border-t border-[var(--oj-border)] bg-[var(--oj-panel)]/50 text-center">
              <Link 
                href={`/submissions?user_id=${username}`} 
                className="text-sm font-medium text-[var(--oj-accent)] hover:underline"
              >
                View all submissions →
              </Link>
            </div>
          )}
        </div>
      </div>
    </PageLayout>
  );
}
