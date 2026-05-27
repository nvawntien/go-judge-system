"use client";

import { useState } from "react";
import { PageLayout } from "@/components/layout/PageLayout";
import { Input } from "@/components/ui/Input";
import { Button } from "@/components/ui/Button";
import { useAuth } from "@/lib/auth-context";
import { useToast } from "@/components/ui/Toast";
import { authApi, ApiError } from "@/lib/api-client";
import { Lock, ShieldAlert, CheckCircle2 } from "lucide-react";
import { useRouter } from "next/navigation";

export default function SecuritySettingsPage() {
  const { state: authState, logout } = useAuth();
  const { addToast } = useToast();
  const router = useRouter();

  const [passwords, setPasswords] = useState({
    current_password: "",
    new_password: "",
    confirm_password: "",
  });
  const [loading, setLoading] = useState(false);

  // Logout All
  const [loggingOutAll, setLoggingOutAll] = useState(false);

  // If not authenticated, the Navbar and layout handle it implicitly or we redirect.
  // We can just render a fallback here.
  if (authState.status !== "AUTHENTICATED") {
    return <PageLayout>Loading...</PageLayout>;
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setPasswords((prev) => ({
      ...prev,
      [e.target.id]: e.target.value,
    }));
  };

  const handlePasswordChange = async (e: React.FormEvent) => {
    e.preventDefault();
    if (passwords.new_password !== passwords.confirm_password) {
      addToast("error", "New passwords do not match.");
      return;
    }

    setLoading(true);
    try {
      await authApi.changePassword(passwords);
      addToast("success", "Password updated successfully.");
      setPasswords({
        current_password: "",
        new_password: "",
        confirm_password: "",
      });
    } catch (err) {
      addToast(
        "error",
        err instanceof ApiError ? err.message : "Failed to change password"
      );
    } finally {
      setLoading(false);
    }
  };

  const handleLogoutAll = async () => {
    if (!window.confirm("Are you sure you want to log out from ALL devices?")) {
      return;
    }
    setLoggingOutAll(true);
    try {
      await authApi.logoutAll();
      addToast("success", "Logged out from all devices.");
      await logout(); // Trigger local logout context
      router.push("/auth/login");
    } catch (err) {
      addToast(
        "error",
        err instanceof ApiError ? err.message : "Failed to logout from all devices"
      );
    } finally {
      setLoggingOutAll(false);
    }
  };

  return (
    <PageLayout>
      <div className="max-w-3xl mx-auto space-y-8">
        <div>
          <h1 className="text-3xl font-bold text-[var(--oj-text)]">
            Security Settings
          </h1>
          <p className="text-[var(--oj-muted)] mt-1">
            Manage your account security and active sessions.
          </p>
        </div>

        {/* Change Password Section */}
        <div className="bg-[var(--oj-surface)] border border-[var(--oj-border)] rounded-xl overflow-hidden shadow-sm">
          <div className="px-6 py-4 border-b border-[var(--oj-border)] bg-[var(--oj-panel)] flex items-center gap-2">
            <Lock size={18} className="text-[var(--oj-accent)]" />
            <h2 className="text-lg font-semibold text-[var(--oj-text)]">
              Change Password
            </h2>
          </div>
          <div className="p-6">
            <form onSubmit={handlePasswordChange} className="space-y-4 max-w-md">
              <Input
                label="Current Password"
                id="current_password"
                type="password"
                required
                value={passwords.current_password}
                onChange={handleChange}
              />
              <Input
                label="New Password"
                id="new_password"
                type="password"
                required
                minLength={8}
                value={passwords.new_password}
                onChange={handleChange}
              />
              <Input
                label="Confirm New Password"
                id="confirm_password"
                type="password"
                required
                minLength={8}
                value={passwords.confirm_password}
                onChange={handleChange}
              />
              <div className="pt-2">
                <Button type="submit" loading={loading}>
                  Update Password
                </Button>
              </div>
            </form>
          </div>
        </div>

        {/* Active Sessions / Logout All */}
        <div className="bg-[var(--oj-wa-bg)] border border-[var(--oj-wa-txt)]/30 rounded-xl overflow-hidden shadow-sm">
          <div className="px-6 py-4 border-b border-[var(--oj-wa-txt)]/20 flex items-center gap-2">
            <ShieldAlert size={18} className="text-[var(--oj-wa-txt)]" />
            <h2 className="text-lg font-semibold text-[var(--oj-wa-txt)]">
              Danger Zone
            </h2>
          </div>
          <div className="p-6">
            <p className="text-sm text-[var(--oj-body)] mb-4">
              If you notice suspicious activity on your account, you can
              immediately revoke all active sessions across all devices. You will
              be required to log in again.
            </p>
            <Button
              variant="secondary"
              onClick={handleLogoutAll}
              loading={loggingOutAll}
              className="border-[var(--oj-wa-txt)] text-[var(--oj-wa-txt)] hover:bg-[var(--oj-wa-txt)] hover:text-white"
            >
              Log out from all devices
            </Button>
          </div>
        </div>
      </div>
    </PageLayout>
  );
}
