"use client";

import { useState, useEffect, Suspense, useRef } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { PageLayout } from "@/components/layout/PageLayout";
import { Button } from "@/components/ui/Button";
import { authApi, ApiError } from "@/lib/api-client";
import { CheckCircle2, AlertCircle, RefreshCw } from "lucide-react";
import Link from "next/link";

function VerifyEmailContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token");

  const [status, setStatus] = useState<"VERIFYING" | "SUCCESS" | "ERROR">("VERIFYING");
  const [error, setError] = useState("");
  const hasVerified = useRef(false);

  useEffect(() => {
    if (!token) {
      setStatus("ERROR");
      setError("Verification token is missing.");
      return;
    }

    if (hasVerified.current) return;
    hasVerified.current = true;

    authApi
      .verifyEmail({ token })
      .then(() => {
        setStatus("SUCCESS");
      })
      .catch((err) => {
        setStatus("ERROR");
        setError(err instanceof ApiError ? err.message : "Failed to verify email");
      });
  }, [token]);

  if (status === "VERIFYING") {
    return (
      <div className="flex flex-col items-center text-center space-y-4 py-8">
        <RefreshCw className="animate-spin text-[var(--oj-accent)]" size={48} />
        <h2 className="text-2xl font-bold text-[var(--oj-text)]">
          Verifying your email...
        </h2>
        <p className="text-[var(--oj-muted)]">Please wait a moment.</p>
      </div>
    );
  }

  if (status === "SUCCESS") {
    return (
      <div className="flex flex-col items-center text-center space-y-4 py-8">
        <CheckCircle2 size={64} className="text-[var(--oj-ac-txt)]" />
        <h2 className="text-2xl font-bold text-[var(--oj-text)]">
          Email Verified!
        </h2>
        <p className="text-[var(--oj-body)]">
          Your email address has been successfully verified. You can now log in
          and access all features.
        </p>
        <Link href="/auth/login">
          <Button className="mt-4">Go to Login</Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="flex flex-col items-center text-center space-y-4 py-8">
      <AlertCircle size={64} className="text-[var(--oj-wa-txt)]" />
      <h2 className="text-2xl font-bold text-[var(--oj-text)]">
        Verification Failed
      </h2>
      <p className="text-[var(--oj-body)]">{error}</p>
      
      {/* Could add a Resend Verification button here in a real app */}
      <Link href="/auth/login">
        <Button className="mt-4" variant="secondary">Back to Login</Button>
      </Link>
    </div>
  );
}

export default function VerifyEmailPage() {
  return (
    <PageLayout className="flex items-center justify-center">
      <div className="w-full max-w-md p-8 rounded-2xl bg-[var(--oj-surface)] border border-[var(--oj-border)] shadow-xl">
        <Suspense fallback={<div className="text-center p-8">Loading...</div>}>
          <VerifyEmailContent />
        </Suspense>
      </div>
    </PageLayout>
  );
}
