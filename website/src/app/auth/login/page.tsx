"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { PageLayout } from "@/components/layout/PageLayout";
import { Input } from "@/components/ui/Input";
import { Button } from "@/components/ui/Button";
import { useAuth } from "@/lib/auth-context";

export default function LoginPage() {
  const router = useRouter();
  const { state, login } = useAuth();

  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");

  useEffect(() => {
    if (state.status === "AUTHENTICATED") {
      router.push("/problems");
    }
  }, [state.status, router]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await login({ identifier, password });
    } catch {
      // Error displayed via state.error in UI
    }
  };

  if (state.status === "AUTHENTICATED") {
    return null;
  }

  return (
    <PageLayout className="flex items-center justify-center">
      <div className="w-full max-w-md p-8 rounded-2xl bg-[var(--oj-surface)] border border-[var(--oj-border)] shadow-xl">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-black text-[var(--oj-text)] mb-2">
            Welcome Back
          </h1>
          <p className="text-[var(--oj-muted)]">Sign in to your account</p>
        </div>

        {state.status === "ERROR" && (
          <div className="mb-6 p-3 rounded-lg bg-[var(--oj-wa-bg)] border border-[var(--oj-wa-txt)]/30 text-[var(--oj-wa-txt)] text-sm font-medium text-center" role="alert">
            {state.error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-5">
          <Input
            label="Username or Email"
            id="identifier"
            type="text"
            required
            value={identifier}
            onChange={(e) => setIdentifier(e.target.value)}
            placeholder="Enter your username or email"
          />

          <Input
            label="Password"
            id="password"
            type="password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Enter your password"
          />

          <Button
            type="submit"
            className="w-full mt-2"
            size="lg"
            loading={state.status === "AUTHENTICATING"}
          >
            Sign In
          </Button>
        </form>

        <div className="mt-8 text-center text-sm text-[var(--oj-muted)]">
          Don&apos;t have an account?{" "}
          <Link
            href="/auth/register"
            className="font-semibold text-[var(--oj-accent)] hover:underline"
          >
            Sign up
          </Link>
        </div>
      </div>
    </PageLayout>
  );
}
