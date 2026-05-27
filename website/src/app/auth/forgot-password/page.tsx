"use client";

import { useState } from "react";
import Link from "next/link";
import { PageLayout } from "@/components/layout/PageLayout";
import { Input } from "@/components/ui/Input";
import { Button } from "@/components/ui/Button";
import { authApi, ApiError } from "@/lib/api-client";
import { Mail, ArrowLeft, CheckCircle2 } from "lucide-react";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      await authApi.forgotPassword({ email });
      setSuccess(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to request password reset");
    } finally {
      setLoading(false);
    }
  };

  return (
    <PageLayout className="flex items-center justify-center">
      <div className="w-full max-w-md p-8 rounded-2xl bg-[var(--oj-surface)] border border-[var(--oj-border)] shadow-xl">
        <Link
          href="/auth/login"
          className="text-[var(--oj-muted)] hover:text-[var(--oj-text)] inline-flex items-center gap-2 mb-6 transition-colors text-sm font-medium"
        >
          <ArrowLeft size={16} />
          Back to Login
        </Link>

        {success ? (
          <div className="flex flex-col items-center text-center space-y-4 py-4">
            <div className="p-4 rounded-full bg-[var(--oj-ac-bg)] text-[var(--oj-ac-txt)] mb-2">
              <Mail size={32} />
            </div>
            <h2 className="text-2xl font-bold text-[var(--oj-text)]">
              Check Your Email
            </h2>
            <p className="text-[var(--oj-body)]">
              We&apos;ve sent password reset instructions to{" "}
              <span className="font-semibold text-[var(--oj-text)]">{email}</span>
            </p>
            <p className="text-sm text-[var(--oj-muted)] mt-4">
              Didn&apos;t receive it? Check your spam folder or try again later.
            </p>
          </div>
        ) : (
          <>
            <div className="mb-8">
              <h1 className="text-3xl font-black text-[var(--oj-text)] mb-2">
                Forgot Password
              </h1>
              <p className="text-[var(--oj-muted)]">
                Enter your email address and we&apos;ll send you a link to reset your
                password.
              </p>
            </div>

            {error && (
              <div className="mb-6 p-3 rounded-lg bg-[var(--oj-wa-bg)] border border-[var(--oj-wa-txt)]/30 text-[var(--oj-wa-txt)] text-sm font-medium text-center" role="alert">
                {error}
              </div>
            )}

            <form onSubmit={handleSubmit} className="space-y-5">
              <Input
                label="Email Address"
                id="email"
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="john@example.com"
              />

              <Button
                type="submit"
                className="w-full mt-2"
                size="lg"
                loading={loading}
              >
                Send Reset Link
              </Button>
            </form>
          </>
        )}
      </div>
    </PageLayout>
  );
}
