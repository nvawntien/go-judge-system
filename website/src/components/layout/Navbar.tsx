"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { LogOut, User, Menu, X } from "lucide-react";
import { useState } from "react";
import { useAuth } from "@/lib/auth-context";
import { ThemeToggle } from "@/components/ui/ThemeToggle";

const NAV_LINKS = [
  { href: "/problems", label: "Problems" },
  { href: "/submissions", label: "Submissions" },
];

export function Navbar() {
  const pathname = usePathname();
  const { state, logout } = useAuth();
  const [mobileOpen, setMobileOpen] = useState(false);

  return (
    <nav
      className="sticky top-0 border-b border-[var(--oj-border)] bg-[var(--oj-bg)]/80 backdrop-blur-md"
      style={{ zIndex: "var(--z-sticky)" } as React.CSSProperties}
    >
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          {/* Logo + Desktop Nav */}
          <div className="flex items-center gap-8">
            <Link
              href="/"
              className="flex items-center gap-2 font-bold text-lg text-[var(--oj-text)]"
            >
              <span className="text-[var(--oj-accent)] text-2xl font-black font-code">
                /&gt;
              </span>
              Go Judge
            </Link>

            <div className="hidden md:flex items-center gap-6">
              {NAV_LINKS.map((link) => {
                const isActive = pathname.startsWith(link.href);
                return (
                  <Link
                    key={link.href}
                    href={link.href}
                    className={`text-sm font-medium transition-colors hover:text-[var(--oj-accent)] ${
                      isActive
                        ? "text-[var(--oj-accent)]"
                        : "text-[var(--oj-body)]"
                    }`}
                  >
                    {link.label}
                  </Link>
                );
              })}
            </div>
          </div>

          {/* Right side: theme + auth + mobile toggle */}
          <div className="flex items-center gap-3">
            <ThemeToggle />

            {state.status === "AUTHENTICATED" ? (
              <div className="hidden md:flex items-center gap-3">
                <Link
                  href="/my/submissions"
                  className={`text-sm font-medium transition-colors hover:text-[var(--oj-accent)] ${
                    pathname.startsWith("/my")
                      ? "text-[var(--oj-accent)]"
                      : "text-[var(--oj-body)]"
                  }`}
                >
                  My Submissions
                </Link>
                <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-[var(--oj-surface)] border border-[var(--oj-border)] text-sm font-medium text-[var(--oj-text)]">
                  <User size={16} className="text-[var(--oj-muted)]" />
                  {state.user.username}
                </div>
                <button
                  onClick={() => logout()}
                  className="cursor-pointer p-2 rounded-md text-[var(--oj-muted)] hover:text-[var(--oj-wa-txt)] hover:bg-[var(--oj-wa-bg)] transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--oj-wa-txt)]"
                  title="Logout"
                  aria-label="Logout"
                >
                  <LogOut size={18} />
                </button>
              </div>
            ) : (
              <div className="hidden md:flex items-center gap-3">
                <Link
                  href="/auth/login"
                  className="text-sm font-medium text-[var(--oj-body)] hover:text-[var(--oj-text)] transition-colors"
                >
                  Log in
                </Link>
                <Link
                  href="/auth/register"
                  className="px-4 py-2 rounded-md text-sm font-medium bg-[var(--oj-accent)] text-white hover:bg-[var(--oj-accent-dk)] transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--oj-accent)] focus-visible:ring-offset-2"
                >
                  Sign up
                </Link>
              </div>
            )}

            {/* Mobile hamburger */}
            <button
              onClick={() => setMobileOpen(!mobileOpen)}
              className="cursor-pointer md:hidden p-2 rounded-md text-[var(--oj-muted)] hover:bg-[var(--oj-surface)]"
              aria-label="Toggle mobile menu"
            >
              {mobileOpen ? <X size={20} /> : <Menu size={20} />}
            </button>
          </div>
        </div>
      </div>

      {/* Mobile drawer */}
      {mobileOpen && (
        <div className="md:hidden border-t border-[var(--oj-border)] bg-[var(--oj-surface)] p-4 space-y-3">
          {NAV_LINKS.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              onClick={() => setMobileOpen(false)}
              className="block text-sm font-medium text-[var(--oj-body)] hover:text-[var(--oj-accent)]"
            >
              {link.label}
            </Link>
          ))}
          {state.status === "AUTHENTICATED" ? (
            <>
              <Link
                href="/my/submissions"
                onClick={() => setMobileOpen(false)}
                className="block text-sm font-medium text-[var(--oj-body)] hover:text-[var(--oj-accent)]"
              >
                My Submissions
              </Link>
              <button
                onClick={() => {
                  logout();
                  setMobileOpen(false);
                }}
                className="cursor-pointer block w-full text-left text-sm font-medium text-[var(--oj-wa-txt)]"
              >
                Logout
              </button>
            </>
          ) : (
            <>
              <Link
                href="/auth/login"
                onClick={() => setMobileOpen(false)}
                className="block text-sm font-medium text-[var(--oj-body)]"
              >
                Log in
              </Link>
              <Link
                href="/auth/register"
                onClick={() => setMobileOpen(false)}
                className="block text-sm font-medium text-[var(--oj-accent)]"
              >
                Sign up
              </Link>
            </>
          )}
        </div>
      )}
    </nav>
  );
}
