import type { ReactNode } from "react";

import { AppNavbar } from "@/components/layout/app-navbar";

type AppShellProps = {
  children: ReactNode;
};

export function AppShell({ children }: AppShellProps) {
  return (
    <div className="app-surface min-h-screen text-foreground">
      <AppNavbar />
      <main className="container py-8">{children}</main>
    </div>
  );
}
