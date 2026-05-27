"use client";

import { useState } from "react";
import Link from "next/link";
import { PageLayout } from "@/components/layout/PageLayout";
import { Input } from "@/components/ui/Input";
import { Button } from "@/components/ui/Button";
import { authApi, ApiError } from "@/lib/api-client";
import { CheckCircle2 } from "lucide-react";

export default function RegisterPage() {
  const [formData, setFormData] = useState({
    full_name: "",
    username: "",
    email: "",
    password: "",
  });

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData((prev) => ({
      ...prev,
      [e.target.id]: e.target.value,
    }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      await authApi.register(formData);
      setSuccess(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Registration failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <PageLayout className="flex items-center justify-center">
      <div className="w-full max-w-md p-8 rounded-2xl bg-[var(--oj-surface)] border border-[var(--oj-border)] shadow-xl">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-black text-[var(--oj-text)] mb-2">
            Create Account
          </h1>
          <p className="text-[var(--oj-muted)]">Join the Go Judge System</p>
        </div>

        {success ? (
          <div className="flex flex-col items-center text-center space-y-4 py-8">
            <CheckCircle2 size={64} className="text-[var(--oj-ac-txt)]" />
            <h2 className="text-2xl font-bold text-[var(--oj-text)]">
              Registration Successful!
            </h2>
            <p className="text-[var(--oj-body)]">
              Please check your email to verify your account before logging in.
            </p>
            <Link href="/auth/login">
              <Button className="mt-4">Go to Login</Button>
            </Link>
          </div>
        ) : (
          <>
            {error && (
              <div className="mb-6 p-3 rounded-lg bg-[var(--oj-wa-bg)] border border-[var(--oj-wa-txt)]/30 text-[var(--oj-wa-txt)] text-sm font-medium text-center" role="alert">
                {error}
              </div>
            )}

            <form onSubmit={handleSubmit} className="space-y-4">
              <Input
                label="Full Name"
                id="full_name"
                type="text"
                required
                value={formData.full_name}
                onChange={handleChange}
                placeholder="John Doe"
              />

              <Input
                label="Username"
                id="username"
                type="text"
                required
                minLength={3}
                value={formData.username}
                onChange={handleChange}
                placeholder="johndoe"
              />

              <Input
                label="Email"
                id="email"
                type="email"
                required
                value={formData.email}
                onChange={handleChange}
                placeholder="john@example.com"
              />

              <Input
                label="Password"
                id="password"
                type="password"
                required
                minLength={6}
                value={formData.password}
                onChange={handleChange}
                placeholder="Choose a strong password"
              />

              <Button
                type="submit"
                className="w-full mt-4"
                size="lg"
                loading={loading}
              >
                Sign Up
              </Button>
            </form>

            <div className="mt-8 text-center text-sm text-[var(--oj-muted)]">
              Already have an account?{" "}
              <Link
                href="/auth/login"
                className="font-semibold text-[var(--oj-accent)] hover:underline"
              >
                Log in
              </Link>
            </div>
          </>
        )}
      </div>
    </PageLayout>
  );
}
