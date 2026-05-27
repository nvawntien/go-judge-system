"use client";

import { useState, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { PageLayout } from "@/components/layout/PageLayout";
import { Input } from "@/components/ui/Input";
import { Button } from "@/components/ui/Button";
import { authApi, ApiError } from "@/lib/api-client";
import { CheckCircle2, AlertCircle } from "lucide-react";
import Link from "next/link";

function ResetPasswordForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token");

  const [passwords, setPasswords] = useState({
    new_password: "",
    confirm_password: "",
  });
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState("");

  if (!token) {
    return (
      <div className="flex flex-col items-center text-center space-y-4 py-8">
        <AlertCircle size={48} className="text-[var(--oj-wa-txt)]" />
        <h2 className="text-2xl font-bold text-[var(--oj-text)]">
          Invalid Reset Link
        </h2>
        <p className="text-[var(--oj-body)]">
          The password reset link is missing or invalid. Please request a new
          one.
        </p>
        <Link href="/auth/forgot-password">
          <Button className="mt-4">Request New Link</Button>
        </Link>
      </div>
    );
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setPasswords((prev) => ({
      ...prev,
      [e.target.id]: e.target.value,
    }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (passwords.new_password !== passwords.confirm_password) {
      setError("Passwords do not match.");
      return;
    }

    setLoading(true);
    setError("");

    try {
      await authApi.resetPassword({
        token,
        new_password: passwords.new_password,
        confirm_password: passwords.confirm_password,
      });
      setSuccess(true);
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : "Failed to reset password"
      );
    } finally {
      setLoading(false);
    }
  };

  if (success) {
    return (
      <div className="flex flex-col items-center text-center space-y-4 py-8">
        <CheckCircle2 size={64} className="text-[var(--oj-ac-txt)]" />
        <h2 className="text-2xl font-bold text-[var(--oj-text)]">
          Password Reset Successfully
        </h2>
        <p className="text-[var(--oj-body)]">
          Your password has been securely updated. You can now log in.
        </p>
        <Link href="/auth/login">
          <Button className="mt-4">Go to Login</Button>
        </Link>
      </div>
    );
  }

  return (
    <>
      <div className="text-center mb-8">
        <h1 className="text-3xl font-black text-[var(--oj-text)] mb-2">
          Set New Password
        </h1>
        <p className="text-[var(--oj-muted)]">
          Please enter your new password below.
        </p>
      </div>

      {error && (
        <div
          className="mb-6 p-3 rounded-lg bg-[var(--oj-wa-bg)] border border-[var(--oj-wa-txt)]/30 text-[var(--oj-wa-txt)] text-sm font-medium text-center"
          role="alert"
        >
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-5">
        <Input
          label="New Password"
          id="new_password"
          type="password"
          required
          minLength={8}
          value={passwords.new_password}
          onChange={handleChange}
          placeholder="At least 8 characters"
        />

        <Input
          label="Confirm New Password"
          id="confirm_password"
          type="password"
          required
          minLength={8}
          value={passwords.confirm_password}
          onChange={handleChange}
          placeholder="Confirm your password"
        />

        <Button
          type="submit"
          className="w-full mt-2"
          size="lg"
          loading={loading}
        >
          Reset Password
        </Button>
      </form>
    </>
  );
}

export default function ResetPasswordPage() {
  return (
    <PageLayout className="flex items-center justify-center">
      <div className="w-full max-w-md p-8 rounded-2xl bg-[var(--oj-surface)] border border-[var(--oj-border)] shadow-xl">
        <Suspense fallback={<div className="text-center p-8">Loading...</div>}>
          <ResetPasswordForm />
        </Suspense>
      </div>
    </PageLayout>
  );
}
