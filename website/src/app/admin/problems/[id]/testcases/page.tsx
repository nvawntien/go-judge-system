"use client";

import { useState, useEffect } from "react";
import { useRouter, useParams } from "next/navigation";
import { PageLayout } from "@/components/layout/PageLayout";
import { Button } from "@/components/ui/Button";
import { problemApi, ApiError } from "@/lib/api-client";
import { useAuth } from "@/lib/auth-context";
import { useToast } from "@/components/ui/Toast";
import { ArrowLeft, UploadCloud, FileArchive, AlertCircle } from "lucide-react";
import Link from "next/link";
import { Skeleton } from "@/components/ui/Skeleton";
import type { ProblemResponse } from "@/types/api";

export default function TestcasesUploadPage() {
  const router = useRouter();
  const params = useParams();
  const id = Number(params.id);
  
  const { state: authState } = useAuth();
  const { addToast } = useToast();

  const [loading, setLoading] = useState(false);
  const [initialLoading, setInitialLoading] = useState(true);
  const [problem, setProblem] = useState<ProblemResponse | null>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);

  const isAdmin =
    authState.status === "AUTHENTICATED" &&
    (authState.user.role === "admin" || authState.user.role === "super_admin");

  useEffect(() => {
    if (authState.status === "AUTHENTICATED" && !isAdmin) {
      router.push("/");
    }
  }, [authState, isAdmin, router]);

  useEffect(() => {
    if (!isAdmin || isNaN(id)) return;
    
    problemApi.getAdmin(id)
      .then((data) => {
        setProblem(data);
      })
      .catch(() => {
        addToast("error", "Failed to load problem details");
        router.push("/admin/problems");
      })
      .finally(() => {
        setInitialLoading(false);
      });
  }, [id, isAdmin, addToast, router]);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      const file = e.target.files[0];
      if (file.name.endsWith(".zip")) {
        setSelectedFile(file);
      } else {
        addToast("error", "Please select a .zip file");
        setSelectedFile(null);
        e.target.value = "";
      }
    }
  };

  const handleUpload = async () => {
    if (!selectedFile) return;
    setLoading(true);

    try {
      await problemApi.uploadTestcase(id, selectedFile);
      addToast("success", "Testcases uploaded and extracted successfully.");
      setSelectedFile(null);
      // reset file input
      const fileInput = document.getElementById("testcase-file") as HTMLInputElement;
      if (fileInput) fileInput.value = "";
    } catch (err) {
      addToast(
        "error",
        err instanceof ApiError ? err.message : "Failed to upload testcases"
      );
    } finally {
      setLoading(false);
    }
  };

  if (authState.status === "AUTHENTICATING" || !isAdmin || initialLoading) {
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
      <div className="max-w-3xl mx-auto space-y-6">
        <div className="flex items-center gap-4">
          <Link
            href="/admin/problems"
            className="p-2 -ml-2 rounded-md hover:bg-[var(--oj-surface)] text-[var(--oj-muted)] hover:text-[var(--oj-text)] transition-colors"
          >
            <ArrowLeft size={20} />
          </Link>
          <div>
            <h1 className="text-3xl font-bold text-[var(--oj-text)]">
              Manage Test Cases
            </h1>
            <p className="text-[var(--oj-muted)] mt-1">
              Upload test data for <span className="font-semibold text-[var(--oj-text)]">{problem?.title}</span> (#{id})
            </p>
          </div>
        </div>

        <div className="bg-[var(--oj-surface)] border border-[var(--oj-border)] rounded-xl overflow-hidden">
          <div className="px-6 py-4 border-b border-[var(--oj-border)] bg-[var(--oj-panel)] flex items-center gap-2">
            <UploadCloud size={18} className="text-[var(--oj-accent)]" />
            <h2 className="text-lg font-semibold text-[var(--oj-text)]">
              Upload Testcases Archive
            </h2>
          </div>
          
          <div className="p-6 space-y-6">
            <div className="p-4 rounded-lg bg-[var(--oj-panel)] border border-[var(--oj-border)] flex gap-3 text-sm text-[var(--oj-body)]">
              <AlertCircle size={18} className="text-[var(--oj-accent)] flex-shrink-0 mt-0.5" />
              <div>
                <p className="font-semibold text-[var(--oj-text)] mb-1">Archive Format Requirements:</p>
                <ul className="list-disc list-inside space-y-1 text-[var(--oj-muted)]">
                  <li>File must be a <strong>.zip</strong> archive.</li>
                  <li>Pairs of input/output files should have the same prefix (e.g. <code>1.in</code> and <code>1.out</code>, or <code>test1.in</code> and <code>test1.out</code>).</li>
                  <li>The system will automatically parse and upload the extracted files to the MinIO testcase bucket.</li>
                </ul>
              </div>
            </div>

            <div className="border-2 border-dashed border-[var(--oj-border)] rounded-xl p-8 text-center hover:bg-[var(--oj-panel)] transition-colors">
              <FileArchive size={48} className="mx-auto text-[var(--oj-muted)] mb-4" />
              <div className="mb-4">
                <input
                  type="file"
                  id="testcase-file"
                  accept=".zip"
                  onChange={handleFileChange}
                  className="hidden"
                />
                <label
                  htmlFor="testcase-file"
                  className="cursor-pointer inline-flex items-center justify-center gap-2 px-4 py-2 text-sm font-medium rounded-md bg-[var(--oj-accent)] text-white hover:bg-[var(--oj-accent-dk)] transition-colors"
                >
                  Select .zip File
                </label>
              </div>
              {selectedFile && (
                <div className="text-sm font-medium text-[var(--oj-text)]">
                  Selected: {selectedFile.name} ({(selectedFile.size / 1024).toFixed(1)} KB)
                </div>
              )}
            </div>

            <div className="flex justify-end pt-4 border-t border-[var(--oj-border)]">
              <Button
                onClick={handleUpload}
                disabled={!selectedFile}
                loading={loading}
                icon={<UploadCloud size={16} />}
              >
                Upload Testcases
              </Button>
            </div>
          </div>
        </div>
      </div>
    </PageLayout>
  );
}
